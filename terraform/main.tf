# ═══════════════════════════════════════════════════════════
# Data Sources
# ═══════════════════════════════════════════════════════════

# Récupérer les zones de disponibilité disponibles dans la région
data "aws_availability_zones" "available" {
  state = "available"
}

# ═══════════════════════════════════════════════════════════
# VPC et Configuration Réseau
# ═══════════════════════════════════════════════════════════

resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "${var.project_name}-vpc"
  }
}

# Subnet Public (pour le serveur K3s avec accès Internet)
resource "aws_subnet" "public" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = var.public_subnets[0]
  availability_zone       = data.aws_availability_zones.available.names[0]
  map_public_ip_on_launch = true

  tags = {
    Name = "${var.project_name}-public-subnet"
  }
}

# Subnets Privés (pour RDS Multi-AZ)
resource "aws_subnet" "private" {
  count             = 2
  vpc_id            = aws_vpc.main.id
  cidr_block        = var.private_subnets[count.index]
  availability_zone = data.aws_availability_zones.available.names[count.index]

  tags = {
    Name = "${var.project_name}-private-subnet-${count.index + 1}"
  }
}

# Internet Gateway (pour accès Internet depuis le subnet public)
resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "${var.project_name}-igw"
  }
}

# Table de routage publique
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }

  tags = {
    Name = "${var.project_name}-public-rt"
  }
}

# Association subnet public <-> route table
resource "aws_route_table_association" "public" {
  subnet_id      = aws_subnet.public.id
  route_table_id = aws_route_table.public.id
}

# ═══════════════════════════════════════════════════════════
# Security Groups
# ═══════════════════════════════════════════════════════════

# Security Group pour le serveur K3s
resource "aws_security_group" "k3s" {
  name_prefix = "${var.project_name}-k3s-"
  description = "Security group pour le serveur K3s"
  vpc_id      = aws_vpc.main.id

  # SSH depuis Internet
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "SSH"
  }

  # HTTP/HTTPS depuis Internet
  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "HTTP"
  }

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "HTTPS"
  }

  # Port de l'application
  ingress {
    from_port   = var.app_port
    to_port     = var.app_port
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Application"
  }

  # API Kubernetes
  ingress {
    from_port   = 6443
    to_port     = 6443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Kubernetes API"
  }

  # Tout le trafic sortant
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "All outbound traffic"
  }

  tags = {
    Name = "${var.project_name}-k3s-sg"
  }
}

# Security Group pour RDS
resource "aws_security_group" "rds" {
  name_prefix = "${var.project_name}-rds-"
  description = "Security group pour RDS PostgreSQL"
  vpc_id      = aws_vpc.main.id

  # PostgreSQL depuis K3s uniquement
  ingress {
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.k3s.id]
    description     = "PostgreSQL depuis K3s"
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.project_name}-rds-sg"
  }
}

# ═══════════════════════════════════════════════════════════
# Clé SSH (générée automatiquement)
# ═══════════════════════════════════════════════════════════

resource "tls_private_key" "deployer" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "aws_key_pair" "deployer" {
  key_name   = var.key_name
  public_key = tls_private_key.deployer.public_key_openssh

  tags = {
    Name = "${var.project_name}-key"
  }

  lifecycle {
    create_before_destroy = true
  }
}

# ═══════════════════════════════════════════════════════════
# Mot de passe base de données (généré automatiquement)
# ═══════════════════════════════════════════════════════════

resource "random_password" "db_password" {
  length  = 32
  special = true
  # Éviter certains caractères spéciaux qui peuvent poser problème dans les URLs
  override_special = "!#$%&*()-_=+[]{}:?"
}

# ═══════════════════════════════════════════════════════════
# Base de données RDS PostgreSQL
# ═══════════════════════════════════════════════════════════

resource "aws_db_subnet_group" "main" {
  name       = "${var.project_name}-db-subnet-group"
  subnet_ids = aws_subnet.private[*].id

  tags = {
    Name = "${var.project_name}-db-subnet-group"
  }
}

resource "aws_db_instance" "postgres" {
  identifier        = "${var.project_name}-postgres"
  allocated_storage = var.db_allocated_storage
  engine            = "postgres"
  engine_version    = "17"
  instance_class    = var.db_instance_class
  db_name           = var.db_name

  username = var.db_username
  password = random_password.db_password.result

  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [aws_security_group.rds.id]
  publicly_accessible    = false

  backup_retention_period = 7
  skip_final_snapshot     = true

  tags = {
    Name = "${var.project_name}-postgres"
  }
}

# ═══════════════════════════════════════════════════════════
# AMI Amazon Linux 2023
# ═══════════════════════════════════════════════════════════

data "aws_ami" "amazon_linux" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-*-x86_64"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

# ═══════════════════════════════════════════════════════════
# Serveur EC2 avec K3s (Kubernetes)
# ═══════════════════════════════════════════════════════════

resource "aws_instance" "k3s_server" {
  ami                    = data.aws_ami.amazon_linux.id
  instance_type          = var.instance_type
  key_name               = aws_key_pair.deployer.key_name
  subnet_id              = aws_subnet.public.id
  vpc_security_group_ids = [aws_security_group.k3s.id]

  root_block_device {
    volume_size = 30
    volume_type = "gp3"
  }

  user_data = templatefile("${path.module}/user-data.sh", {
    db_endpoint      = aws_db_instance.postgres.endpoint
    db_name          = var.db_name
    db_username      = var.db_username
    db_password      = random_password.db_password.result
    docker_username  = var.docker_username
    app_port         = var.app_port
    webhook_url      = var.webhook_url
  })

  tags = {
    Name = "${var.project_name}-k3s-server"
  }

  depends_on = [aws_db_instance.postgres]
}

# Elastic IP pour avoir une IP fixe
resource "aws_eip" "k3s" {
  instance = aws_instance.k3s_server.id
  domain   = "vpc"

  tags = {
    Name = "${var.project_name}-eip"
  }
}
