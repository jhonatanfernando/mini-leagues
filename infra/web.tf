# The site bucket is private. CloudFront is the only public entry point and
# signs requests to the S3 REST origin using origin access control.
resource "aws_s3_bucket" "web" {
  bucket = "podium-picks-web-${var.account_id}-${var.region}-${var.environment}"
}

resource "aws_s3_bucket_public_access_block" "web" {
  bucket                  = aws_s3_bucket.web.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_ownership_controls" "web" {
  bucket = aws_s3_bucket.web.id
  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_cloudfront_origin_access_control" "web" {
  name                              = "podium-picks-web-${var.environment}"
  description                       = "Private S3 access for Podium Picks"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

# Only the OAuth callback is rewritten; missing assets retain normal errors.
resource "aws_cloudfront_function" "web_callback" {
  name    = "podium-picks-callback-${var.environment}"
  runtime = "cloudfront-js-2.0"
  comment = "Serve the application shell for the OAuth callback"
  publish = true
  code    = file("${path.module}/web_callback.js")
}

resource "aws_cloudfront_response_headers_policy" "web" {
  name = "podium-picks-security-${var.environment}"
  security_headers_config {
    content_type_options { override = true }
    frame_options {
      frame_option = "DENY"
      override     = true
    }
    referrer_policy {
      referrer_policy = "strict-origin-when-cross-origin"
      override        = true
    }
    strict_transport_security {
      access_control_max_age_sec = 31536000
      include_subdomains         = false
      override                   = true
    }
    xss_protection {
      protection = true
      mode_block = true
      override   = true
    }
  }
}

resource "aws_cloudfront_distribution" "web" {
  enabled             = true
  is_ipv6_enabled     = true
  default_root_object = "index.html"
  comment             = "Podium Picks web ${var.environment}"

  origin {
    domain_name              = aws_s3_bucket.web.bucket_regional_domain_name
    origin_id                = "private-web-bucket"
    origin_access_control_id = aws_cloudfront_origin_access_control.web.id
  }

  default_cache_behavior {
    target_origin_id       = "private-web-bucket"
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD"]
    cached_methods         = ["GET", "HEAD"]
    compress               = true
    # AWS managed CachingDisabled: fresh HTML and no cached OAuth callback.
    cache_policy_id            = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"
    response_headers_policy_id = aws_cloudfront_response_headers_policy.web.id
    function_association {
      event_type   = "viewer-request"
      function_arn = aws_cloudfront_function.web_callback.arn
    }
  }

  restrictions {
    geo_restriction { restriction_type = "none" }
  }
  viewer_certificate { cloudfront_default_certificate = true }

  lifecycle {
    precondition {
      condition     = fileexists("${path.module}/../web/dist/index.html")
      error_message = "Build web/dist before planning or applying so existing website objects cannot be removed by an empty build."
    }
  }
}

# OAC is constrained to this distribution and read access to this bucket only.
resource "aws_s3_bucket_policy" "web" {
  bucket = aws_s3_bucket.web.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid       = "AllowCloudFrontRead"
      Effect    = "Allow"
      Principal = { Service = "cloudfront.amazonaws.com" }
      Action    = "s3:GetObject"
      Resource  = "${aws_s3_bucket.web.arn}/*"
      Condition = { StringEquals = { "AWS:SourceArn" = aws_cloudfront_distribution.web.arn } }
    }]
  })
}

locals {
  web_origin = "https://${aws_cloudfront_distribution.web.domain_name}"
  web_mime_types = {
    html = "text/html; charset=utf-8"
    js   = "text/javascript; charset=utf-8"
    css  = "text/css; charset=utf-8"
    svg  = "image/svg+xml"
    png  = "image/png"
    ico  = "image/x-icon"
    json = "application/json"
  }
}

# Build once before planning and apply that same saved plan and build.
resource "aws_s3_object" "web" {
  for_each      = fileset("${path.module}/../web/dist", "**")
  bucket        = aws_s3_bucket.web.id
  key           = each.value
  source        = "${path.module}/../web/dist/${each.value}"
  source_hash   = filemd5("${path.module}/../web/dist/${each.value}")
  content_type  = lookup(local.web_mime_types, lower(element(reverse(split(".", each.value)), 0)), "application/octet-stream")
  cache_control = "no-store"
}
