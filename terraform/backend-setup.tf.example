# ═══════════════════════════════════════════════════════════
# Configuration du backend S3 pour l'état Terraform
# ═══════════════════════════════════════════════════════════
# Ce fichier crée les ressources nécessaires pour le backend S3
# À exécuter AVANT d'activer le backend dans provider.tf
# ═══════════════════════════════════════════════════════════

# Bucket S3 pour stocker l'état Terraform
resource "aws_s3_bucket" "terraform_state" {
  bucket = "${var.project_name}-terraform-state-${var.environment}"

  tags = {
    Name        = "Terraform State Bucket"
    Environment = var.environment
  }
}

# Versioning du bucket pour garder l'historique des états
resource "aws_s3_bucket_versioning" "terraform_state" {
  bucket = aws_s3_bucket.terraform_state.id

  versioning_configuration {
    status = "Enabled"
  }
}

# Chiffrement du bucket
resource "aws_s3_bucket_server_side_encryption_configuration" "terraform_state" {
  bucket = aws_s3_bucket.terraform_state.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# Bloquer l'accès public
resource "aws_s3_bucket_public_access_block" "terraform_state" {
  bucket = aws_s3_bucket.terraform_state.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Table DynamoDB pour le verrouillage de l'état
resource "aws_dynamodb_table" "terraform_state_lock" {
  name           = "${var.project_name}-terraform-lock-${var.environment}"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "LockID"

  attribute {
    name = "LockID"
    type = "S"
  }

  tags = {
    Name        = "Terraform State Lock Table"
    Environment = var.environment
  }
}

# Outputs pour utiliser dans provider.tf
output "terraform_state_bucket" {
  value       = aws_s3_bucket.terraform_state.bucket
  description = "Nom du bucket S3 pour l'état Terraform"
}

output "terraform_state_lock_table" {
  value       = aws_dynamodb_table.terraform_state_lock.name
  description = "Nom de la table DynamoDB pour le verrouillage"
}
