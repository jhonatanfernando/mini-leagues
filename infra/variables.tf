variable "account_id" {
  description = "Expected AWS account ID; independently guards provider access."
  type        = string

  validation {
    condition     = can(regex("^[0-9]{12}$", var.account_id))
    error_message = "account_id must contain exactly 12 digits."
  }
}

variable "region" {
  description = "Application AWS region. Changing it requires a separate migration review."
  type        = string
  default     = "us-east-2"

  validation {
    condition     = length(trimspace(var.region)) > 0
    error_message = "region must not be blank."
  }
}

variable "environment" {
  description = "This root currently manages only dev; additional environments need separate state and resource names."
  type        = string
  default     = "dev"

  validation {
    condition     = var.environment == "dev"
    error_message = "Only dev is supported by this root's existing resource names."
  }
}

variable "owner" {
  description = "Owner tag for application resources."
  type        = string
  default     = "jhonatan.oliveira"

  validation {
    condition     = length(trimspace(var.owner)) > 0
    error_message = "owner must not be blank."
  }
}

variable "log_retention_days" {
  description = "CloudWatch Lambda log retention in days; explicit supported retention period."
  type        = number
  default     = 7

  validation {
    condition     = contains([1, 3, 5, 7, 14, 30, 60, 90, 120, 150, 180, 365, 400, 545, 731, 1096, 1827, 2192, 2557, 2922, 3288, 3653], var.log_retention_days)
    error_message = "Choose a supported finite CloudWatch log retention period."
  }
}

variable "lambda_memory_mb" {
  description = "Lambda memory in MB; benchmark changes separately."
  type        = number
  default     = 128

  validation {
    condition     = var.lambda_memory_mb >= 128 && var.lambda_memory_mb <= 10240 && floor(var.lambda_memory_mb) == var.lambda_memory_mb
    error_message = "lambda_memory_mb must be an integer from 128 to 10240."
  }
}

variable "lambda_timeout_seconds" {
  description = "Lambda timeout in seconds."
  type        = number
  default     = 10

  validation {
    condition     = var.lambda_timeout_seconds >= 1 && var.lambda_timeout_seconds <= 900 && floor(var.lambda_timeout_seconds) == var.lambda_timeout_seconds
    error_message = "lambda_timeout_seconds must be an integer from 1 to 900."
  }
}

variable "api_throttling_burst_limit" {
  description = "API Gateway burst request target; throttling is not a hard spending cap."
  type        = number
  default     = 5

  validation {
    condition     = var.api_throttling_burst_limit >= 1 && floor(var.api_throttling_burst_limit) == var.api_throttling_burst_limit
    error_message = "api_throttling_burst_limit must be a positive integer."
  }
}

variable "api_throttling_rate_limit" {
  description = "API Gateway steady-state requests per second target."
  type        = number
  default     = 2

  validation {
    condition     = var.api_throttling_rate_limit > 0
    error_message = "api_throttling_rate_limit must be positive."
  }
}
