#!/bin/bash

echo "🧹 Nettoyage des ressources AWS orphelines..."

# Fonction pour supprimer une ressource AWS si elle existe et n'est pas dans le state
cleanup_resource() {
    local resource_type=$1
    local terraform_name=$2
    local aws_name=$3
    
    # Vérifier si la ressource est dans le state Terraform
    if terraform state show "$resource_type.$terraform_name" &>/dev/null; then
        echo "✅ $resource_type.$terraform_name est déjà géré par Terraform"
        return 0
    fi
    
    echo "🔍 Vérification de $aws_name dans AWS..."
    
    case "$resource_type" in
        "aws_key_pair")
            if aws ec2 describe-key-pairs --key-names "$aws_name" --region "$AWS_REGION" &>/dev/null; then
                echo "⚠️  Clé SSH '$aws_name' existe dans AWS mais pas dans Terraform state"
                echo "🗑️  Suppression de la clé orpheline..."
                aws ec2 delete-key-pair --key-name "$aws_name" --region "$AWS_REGION"
                echo "✅ Clé supprimée - Terraform la recréera"
            else
                echo "✅ Clé '$aws_name' n'existe pas - OK"
            fi
            ;;
            
        "aws_db_subnet_group")
            if aws rds describe-db-subnet-groups --db-subnet-group-name "$aws_name" --region "$AWS_REGION" &>/dev/null; then
                echo "⚠️  DB Subnet Group '$aws_name' existe dans AWS mais pas dans Terraform state"
                echo "🗑️  Suppression du subnet group orphelin..."
                aws rds delete-db-subnet-group --db-subnet-group-name "$aws_name" --region "$AWS_REGION"
                echo "✅ Subnet group supprimé - Terraform le recréera"
            else
                echo "✅ Subnet group '$aws_name' n'existe pas - OK"
            fi
            ;;
    esac
}

# Nettoyer les ressources orphelines
cleanup_resource "aws_key_pair" "deployer" "auth-service-key"
cleanup_resource "aws_db_subnet_group" "main" "auth-service-db-subnet-group"

echo "✅ Nettoyage terminé - Terraform peut maintenant créer proprement les ressources"
