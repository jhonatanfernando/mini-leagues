locals {
  common_tags = {
    Project     = "mini-leagues"
    Environment = var.environment
    Owner       = var.owner
    ManagedBy   = "Terraform"
  }
}
