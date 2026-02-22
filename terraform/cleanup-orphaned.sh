#!/bin/bash

echo "🔧 Gestion intelligente des ressources AWS orphelines..."

# Fonction pour gérer une ressource orpheline (import ou suppression)
handle_orphaned_resource() {
    local resource_type=$1
    local terraform_name=$2
    local aws_id=$3
    
    # Vérifier si la ressource est dans le state Terraform
    if terraform state show "$resource_type.$terraform_name" &>/dev/null; then
        echo "✅ $resource_type.$terraform_name est déjà géré par Terraform"
        return 0
    fi
    
    echo "🔍 Vérification de $aws_id dans AWS..."
    
    case "$resource_type" in
        "aws_key_pair")
            if aws ec2 describe-key-pairs --key-names "$aws_id" --region "$AWS_REGION" &>/dev/null; then
                echo "⚠️  Clé SSH '$aws_id' existe dans AWS mais pas dans Terraform state"
                echo "🗑️  Suppression de la clé orpheline (sera recréée)..."
                aws ec2 delete-key-pair --key-name "$aws_id" --region "$AWS_REGION" || true
                echo "✅ Clé supprimée"
            else
                echo "✅ Clé '$aws_id' n'existe pas - OK"
            fi
            ;;
            
        "aws_db_subnet_group")
            if aws rds describe-db-subnet-groups --db-subnet-group-name "$aws_id" --region "$AWS_REGION" &>/dev/null 2>&1; then
                echo "⚠️  DB Subnet Group '$aws_id' existe dans AWS mais pas dans Terraform state"
                
                # Vérifier si une RDS l'utilise
                if aws rds describe-db-instances --region "$AWS_REGION" --query "DBInstances[?DBSubnetGroup.DBSubnetGroupName=='$aws_id'].DBInstanceIdentifier" --output text 2>/dev/null | grep -q .; then
                    echo "⚠️  Une RDS utilise ce subnet group - Import dans le state au lieu de supprimer"
                    echo "📥 Import de $resource_type.$terraform_name..."
                    
                    # On essaye l'import (peut échouer si variables manquantes, mais on continue)
                    if terraform import -input=false "$resource_type.$terraform_name" "$aws_id" 2>&1 | grep -q "successfully imported\|Import successful"; then
                        echo "✅ Subnet group importé avec succès"
                    else
                        echo "⚠️  Import échoué - Terraform le gérera lors du plan/apply"
                    fi
                else
                    echo "🗑️  Aucune RDS ne l'utilise - Suppression..."
                    aws rds delete-db-subnet-group --db-subnet-group-name "$aws_id" --region "$AWS_REGION" || true
                    echo "✅ Subnet group supprimé"
                fi
            else
                echo "✅ Subnet group '$aws_id' n'existe pas - OK"
            fi
            ;;
            
        "aws_db_instance")
            # Pour la RDS, on ne fait qu'importer si elle existe
            if aws rds describe-db-instances --db-instance-identifier "$aws_id" --region "$AWS_REGION" &>/dev/null 2>&1; then
                echo "⚠️  Instance RDS '$aws_id' existe dans AWS mais pas dans Terraform state"
                echo "📥 Import de $resource_type.$terraform_name..."
                
                if terraform import -input=false "$resource_type.$terraform_name" "$aws_id" 2>&1 | grep -q "successfully imported\|Import successful"; then
                    echo "✅ Instance RDS importée avec succès"
                else
                    echo "⚠️  Import échoué - Terraform le gérera lors du plan/apply"
                fi
            else
                echo "✅ Instance RDS '$aws_id' n'existe pas - OK"
            fi
            ;;
    esac
}

# Gérer les ressources orphelines
handle_orphaned_resource "aws_key_pair" "deployer" "auth-service-key"
handle_orphaned_resource "aws_db_instance" "postgres" "auth-service-postgres"
handle_orphaned_resource "aws_db_subnet_group" "main" "auth-service-db-subnet-group"

echo "✅ Gestion des ressources orphelines terminée"
