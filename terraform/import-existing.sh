#!/bin/bash
set -e

echo "🔍 Vérification des ressources existantes..."

# Fonction pour vérifier et importer une ressource si elle existe
import_if_exists() {
    local resource_type=$1
    local resource_name=$2
    local aws_id=$3
    
    # Vérifier si la ressource est déjà dans le state
    if terraform state show "$resource_type.$resource_name" &>/dev/null; then
        echo "✅ $resource_type.$resource_name déjà dans le state"
        return 0
    fi
    
    # Essayer d'importer la ressource
    echo "🔄 Tentative d'import de $resource_type.$resource_name..."
    if terraform import "$resource_type.$resource_name" "$aws_id" 2>/dev/null; then
        echo "✅ $resource_type.$resource_name importé avec succès"
        return 0
    else
        echo "ℹ️  $resource_type.$resource_name n'existe pas encore dans AWS"
        return 1
    fi
}

# Importer la clé SSH si elle existe
import_if_exists "aws_key_pair" "deployer" "auth-service-key"

# Importer le DB subnet group si il existe
import_if_exists "aws_db_subnet_group" "main" "auth-service-db-subnet-group"

echo "✅ Vérification terminée"
