data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

resource "aws_s3_bucket" "learning" {
  bucket = format("terraform-learning-%s-%s-dev", data.aws_caller_identity.current.account_id, data.aws_region.current.region)

  tags = {
    Environment = "Development"
    ManagedBy   = "Terraform"
  }
}
