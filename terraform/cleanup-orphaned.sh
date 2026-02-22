#!/bin/bash

echo "🔧 Gestion intelligente des ressources AWS orphelines..."

# Fonction pour importer une ressource si elle existe
safe_import() {
    local resource_type=$1
    local terraform_name=$2
    local aws_id=$3
    
    if terraform state show "$resource_type.$terraform_name" &>/dev/null; then
        echo "✅ $resource_type.$terraform_name déjà dans le state"
        return 0
    fi
    
    echo "📥 Import de $resource_type.$terraform_name (ID: $aws_id)..."
    if terraform import -input=false "$resource_type.$terraform_name" "$aws_id" 2>&1 | grep -q "successfully imported\|Import successful"; then
        echo "✅ Importé avec succès"
        return 0
    else
        echo "ℹ️  N'existe pas ou import échoué"
        return 1
    fi
}

echo ""
echo "🔍 Phase 1: Gestion du VPC et réseau..."

# Trouver le VPC auth-service s'il existe
VPC_ID=$(aws ec2 describe-vpcs --region "$AWS_REGION" \
    --filters "Name=tag:Name,Values=auth-service-vpc" \
    --query "Vpcs[0].VpcId" --output text 2>/dev/null)

if [ "$VPC_ID" != "None" ] && [ -n "$VPC_ID" ]; then
    echo "⚠️  VPC 'auth-service-vpc' trouvé: $VPC_ID"
    safe_import "aws_vpc" "main" "$VPC_ID"
    
    # Import Internet Gateway
    IGW_ID=$(aws ec2 describe-internet-gateways --region "$AWS_REGION" \
        --filters "Name=attachment.vpc-id,Values=$VPC_ID" \
        --query "InternetGateways[0].InternetGatewayId" --output text 2>/dev/null)
    if [ "$IGW_ID" != "None" ] && [ -n "$IGW_ID" ]; then
        safe_import "aws_internet_gateway" "main" "$IGW_ID"
    fi
    
    # Import Public Subnet
    PUBLIC_SUBNET_ID=$(aws ec2 describe-subnets --region "$AWS_REGION" \
        --filters "Name=vpc-id,Values=$VPC_ID" "Name=tag:Name,Values=auth-service-public-subnet" \
        --query "Subnets[0].SubnetId" --output text 2>/dev/null)
    if [ "$PUBLIC_SUBNET_ID" != "None" ] && [ -n "$PUBLIC_SUBNET_ID" ]; then
        safe_import "aws_subnet" "public" "$PUBLIC_SUBNET_ID"
    fi
    
    # Import Private Subnets
    PRIVATE_SUBNETS=$(aws ec2 describe-subnets --region "$AWS_REGION" \
        --filters "Name=vpc-id,Values=$VPC_ID" "Name=tag:Name,Values=auth-service-private-subnet-*" \
        --query "Subnets[].SubnetId" --output text 2>/dev/null)
    
    if [ -n "$PRIVATE_SUBNETS" ]; then
        INDEX=0
        for SUBNET_ID in $PRIVATE_SUBNETS; do
            safe_import "aws_subnet" "private[$INDEX]" "$SUBNET_ID"
            INDEX=$((INDEX + 1))
        done
    fi
    
    # Import Route Table
    RT_ID=$(aws ec2 describe-route-tables --region "$AWS_REGION" \
        --filters "Name=vpc-id,Values=$VPC_ID" "Name=tag:Name,Values=auth-service-public-rt" \
        --query "RouteTables[0].RouteTableId" --output text 2>/dev/null)
    if [ "$RT_ID" != "None" ] && [ -n "$RT_ID" ]; then
        safe_import "aws_route_table" "public" "$RT_ID"
        
        # Import Route Table Association
        ASSOC_ID=$(aws ec2 describe-route-tables --region "$AWS_REGION" \
            --route-table-ids "$RT_ID" \
            --query "RouteTables[0].Associations[?SubnetId=='$PUBLIC_SUBNET_ID'].RouteTableAssociationId" \
            --output text 2>/dev/null)
        if [ "$ASSOC_ID" != "None" ] && [ -n "$ASSOC_ID" ]; then
            safe_import "aws_route_table_association" "public" "$ASSOC_ID"
        fi
    fi
    
    # Import Security Groups
    K3S_SG_ID=$(aws ec2 describe-security-groups --region "$AWS_REGION" \
        --filters "Name=vpc-id,Values=$VPC_ID" "Name=group-name,Values=auth-service-k3s-*" \
        --query "SecurityGroups[0].GroupId" --output text 2>/dev/null)
    if [ "$K3S_SG_ID" != "None" ] && [ -n "$K3S_SG_ID" ]; then
        safe_import "aws_security_group" "k3s" "$K3S_SG_ID"
    fi
    
    RDS_SG_ID=$(aws ec2 describe-security-groups --region "$AWS_REGION" \
        --filters "Name=vpc-id,Values=$VPC_ID" "Name=group-name,Values=auth-service-rds-*" \
        --query "SecurityGroups[0].GroupId" --output text 2>/dev/null)
    if [ "$RDS_SG_ID" != "None" ] && [ -n "$RDS_SG_ID" ]; then
        safe_import "aws_security_group" "rds" "$RDS_SG_ID"
    fi
else
    echo "✅ Aucun VPC auth-service trouvé - Terraform en créera un nouveau"
fi

echo ""
echo "🔍 Phase 2: Gestion des ressources compute..."

# SSH Key Pair - Supprimer car sera recréée
if terraform state show "aws_key_pair.deployer" &>/dev/null; then
    echo "✅ aws_key_pair.deployer déjà dans le state"
else
    if aws ec2 describe-key-pairs --key-names "auth-service-key" --region "$AWS_REGION" &>/dev/null 2>&1; then
        echo "⚠️  Clé SSH 'auth-service-key' existe - Suppression (sera recréée)..."
        aws ec2 delete-key-pair --key-name "auth-service-key" --region "$AWS_REGION" 2>/dev/null
        echo "✅ Clé supprimée"
    else
        echo "✅ Clé SSH n'existe pas - OK"
    fi
fi

# EIP
EIP_ALLOC_ID=$(aws ec2 describe-addresses --region "$AWS_REGION" \
    --filters "Name=tag:Name,Values=auth-service-eip" \
    --query "Addresses[0].AllocationId" --output text 2>/dev/null)
if [ "$EIP_ALLOC_ID" != "None" ] && [ -n "$EIP_ALLOC_ID" ]; then
    safe_import "aws_eip" "k3s" "$EIP_ALLOC_ID"
fi

# EC2 Instance
INSTANCE_ID=$(aws ec2 describe-instances --region "$AWS_REGION" \
    --filters "Name=tag:Name,Values=auth-service-k3s" "Name=instance-state-name,Values=running,stopped,stopping" \
    --query "Reservations[0].Instances[0].InstanceId" --output text 2>/dev/null)
if [ "$INSTANCE_ID" != "None" ] && [ -n "$INSTANCE_ID" ]; then
    safe_import "aws_instance" "k3s_server" "$INSTANCE_ID"
fi

echo ""
echo "🔍 Phase 3: Gestion de la base de données..."

# RDS Instance
if safe_import "aws_db_instance" "postgres" "auth-service-postgres"; then
    echo "✅ RDS instance importée"
fi

# DB Subnet Group
if aws rds describe-db-subnet-groups --db-subnet-group-name "auth-service-db-subnet-group" --region "$AWS_REGION" &>/dev/null 2>&1; then
    safe_import "aws_db_subnet_group" "main" "auth-service-db-subnet-group"
fi

echo ""
echo "✅ Gestion des ressources orphelines terminée"
echo "📊 Résumé: Les ressources existantes ont été importées dans le state Terraform"
