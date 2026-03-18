variable "aws_region" {
  description = "Région AWS"
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Environnement (prod, staging, dev)"
  type        = string
  default     = "prod"
}

variable "project_name" {
  description = "Nom du projet"
  type        = string
  default     = "auth-service"
}

# ═══════════════════════════════════════════════════════════
# VPC Configuration
# ═══════════════════════════════════════════════════════════

variable "vpc_cidr" {
  description = "CIDR block pour le VPC"
  type        = string
  default     = "10.0.0.0/16"
}
variable "public_subnets" {
  description = "CIDR blocks pour les subnets publics"
  type        = list(string)
  default     = ["10.0.1.0/24"]
}

variable "private_subnets" {
  description = "CIDR blocks pour les subnets privés (RDS)"
  type        = list(string)
  default     = ["10.0.10.0/24", "10.0.11.0/24"]
}

# ═══════════════════════════════════════════════════════════
# EC2 Configuration
# ═══════════════════════════════════════════════════════════

variable "instance_type" {
  description = "Type d'instance EC2"
  type        = string
  default     = "t3.medium"
}

variable "key_name" {
  description = "Nom de la clé SSH"
  type        = string
  default     = "auth-service-key"
}

# ═══════════════════════════════════════════════════════════
# RDS Configuration
# ═══════════════════════════════════════════════════════════

variable "db_allocated_storage" {
  description = "Taille du stockage RDS (GB)"
  type        = number
  default     = 20
}

variable "db_instance_class" {
  description = "Type d'instance RDS"
  type        = string
  default     = "db.t3.micro"
}

variable "db_name" {
  description = "Nom de la base de données"
  type        = string
  default     = "authdb"
}

variable "db_username" {
  description = "Username pour la base de données"
  type        = string
  default     = "authuser"
  sensitive   = true
}

# ═══════════════════════════════════════════════════════════
# Application Configuration
# ═══════════════════════════════════════════════════════════

variable "docker_username" {
  description = "Docker Hub username"
  type        = string
}

variable "app_port" {
  description = "Port de l'application"
  type        = number
  default     = 8081
}

variable "webhook_url" {
  description = "URL du webhook pour les notifications"
  type        = string
  default     = "https://webhook.site/your-unique-id"
}
