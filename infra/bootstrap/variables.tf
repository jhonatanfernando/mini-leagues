variable "account_id" {
  description = "Expected AWS account ID; prevents deployment to a different account."
  type        = string

  validation {
    condition     = can(regex("^[0-9]{12}$", var.account_id))
    error_message = "account_id must contain exactly 12 digits."
  }
}

variable "region" {
  description = "AWS region for the state bucket."
  type        = string
  default     = "us-east-2"
}
