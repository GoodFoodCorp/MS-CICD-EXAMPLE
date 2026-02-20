output "vpc_id" {
  description = "ID du VPC"
  value       = aws_vpc.main.id
}

output "public_subnet_id" {
  description = "ID du subnet public"
  value       = aws_subnet.public.id
}

output "private_subnet_ids" {
  description = "IDs des subnets privés"
  value       = aws_subnet.private[*].id
}

# ═══════════════════════════════════════════════════════════
# Base de données
# ═══════════════════════════════════════════════════════════

output "rds_endpoint" {
  description = "Endpoint de la base de données RDS"
  value       = aws_db_instance.postgres.endpoint
}

output "rds_database_name" {
  description = "Nom de la base de données"
  value       = aws_db_instance.postgres.db_name
}

output "rds_username" {
  description = "Username de la base de données"
  value       = aws_db_instance.postgres.username
  sensitive   = true
}

# ═══════════════════════════════════════════════════════════
# Serveur K3s
# ═══════════════════════════════════════════════════════════

output "k3s_server_id" {
  description = "ID de l'instance EC2 K3s"
  value       = aws_instance.k3s_server.id
}

output "k3s_public_ip" {
  description = "IP publique du serveur K3s"
  value       = aws_eip.k3s.public_ip
}

output "k3s_private_ip" {
  description = "IP privée du serveur K3s"
  value       = aws_instance.k3s_server.private_ip
}

# ═══════════════════════════════════════════════════════════
# URLs d'accès
# ═══════════════════════════════════════════════════════════

output "application_url" {
  description = "URL de l'application"
  value       = "http://${aws_eip.k3s.public_ip}:${var.app_port}"
}

output "ssh_command" {
  description = "Commande SSH pour se connecter au serveur"
  value       = "ssh -i ~/.ssh/id_rsa ec2-user@${aws_eip.k3s.public_ip}"
}

# ═══════════════════════════════════════════════════════════
# Commandes Kubernetes
# ═══════════════════════════════════════════════════════════

output "kubectl_config_command" {
  description = "Commande pour récupérer la config kubectl"
  value       = "scp -i ~/.ssh/id_rsa ec2-user@${aws_eip.k3s.public_ip}:/home/ec2-user/.kube/config ~/.kube/config-auth-service"
}

output "kubectl_check_command" {
  description = "Commande pour vérifier le déploiement"
  value       = "kubectl --kubeconfig ~/.kube/config-auth-service get pods -n production"
}

# ═══════════════════════════════════════════════════════════
# Informations de connexion (pour GitHub Secrets)
# ═══════════════════════════════════════════════════════════

output "github_secrets_info" {
  description = "Informations pour configurer les GitHub Secrets"
  value = {
    AWS_REGION              = var.aws_region
    DB_HOST                 = aws_db_instance.postgres.endpoint
    DB_NAME                 = aws_db_instance.postgres.db_name
    K3S_SERVER_IP           = aws_eip.k3s.public_ip
    APPLICATION_URL         = "http://${aws_eip.k3s.public_ip}:${var.app_port}"
  }
}

output "kubeconfig_export_command" {
  description = "Commande pour exporter le kubeconfig en base64 (pour GitHub Secrets)"
  value       = "ssh -i ~/.ssh/id_rsa ec2-user@${aws_eip.k3s.public_ip} 'cat /home/ec2-user/.kube/config' | base64"
}
