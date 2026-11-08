output "state_bucket_name" {
  description = "Bucket name to use in the reviewed backend configuration files."
  value       = aws_s3_bucket.terraform_state.id
}
