# Cognito authentication foundation and JWT authorizer for the league routes.
resource "aws_cognito_user_pool" "podium_picks" {
  name                     = "podium-picks-${var.environment}"
  user_pool_tier           = "LITE"
  username_attributes      = ["email"]
  auto_verified_attributes = ["email"]
  deletion_protection      = "ACTIVE"

  # Development milestone: no SMS or MFA enrollment UI. Review mandatory TOTP
  # before broader use; do not enable paid threat protection implicitly.
  mfa_configuration = "OFF"

  username_configuration {
    case_sensitive = false
  }

  password_policy {
    minimum_length                   = 12
    require_lowercase                = true
    require_uppercase                = true
    require_numbers                  = true
    require_symbols                  = true
    temporary_password_validity_days = 3
  }

  admin_create_user_config {
    allow_admin_create_user_only = false
  }

  verification_message_template {
    default_email_option = "CONFIRM_WITH_CODE"
  }

  email_configuration {
    email_sending_account = "COGNITO_DEFAULT"
  }

  account_recovery_setting {
    recovery_mechanism {
      name     = "verified_email"
      priority = 1
    }
  }

  user_attribute_update_settings {
    attributes_require_verification_before_update = ["email"]
  }
}

resource "aws_cognito_resource_server" "podium_picks" {
  identifier   = "podium-picks"
  name         = "Podium Picks API"
  user_pool_id = aws_cognito_user_pool.podium_picks.id

  scope {
    scope_name        = "play"
    scope_description = "Access the Podium Picks application; league permissions are checked by the backend."
  }
}

resource "aws_cognito_user_pool_client" "web" {
  name         = "podium-picks-web-${var.environment}"
  user_pool_id = aws_cognito_user_pool.podium_picks.id

  generate_secret                               = false
  supported_identity_providers                  = ["COGNITO"]
  allowed_oauth_flows_user_pool_client          = true
  allowed_oauth_flows                           = ["code"]
  allowed_oauth_scopes                          = ["openid", "email", "${aws_cognito_resource_server.podium_picks.identifier}/play"]
  callback_urls                                 = distinct(concat(var.auth_callback_urls, ["${local.web_origin}/auth/callback"]))
  logout_urls                                   = distinct(concat(var.auth_logout_urls, ["${local.web_origin}/"]))
  prevent_user_existence_errors                 = "ENABLED"
  enable_token_revocation                       = true
  enable_propagate_additional_user_context_data = false

  # Browser login will use OAuth code + S256 PKCE; refresh-token rotation
  # is unavailable with the selected Lite tier.
  explicit_auth_flows = ["ALLOW_USER_SRP_AUTH"]
  read_attributes     = ["email", "email_verified", "sub"]
  write_attributes    = ["email"]

  access_token_validity  = 15
  id_token_validity      = 15
  refresh_token_validity = 1

  token_validity_units {
    access_token  = "minutes"
    id_token      = "minutes"
    refresh_token = "days"
  }

}

resource "aws_cognito_user_pool_domain" "podium_picks" {
  domain                = "podium-picks-${var.account_id}-${var.region}-${var.environment}"
  user_pool_id          = aws_cognito_user_pool.podium_picks.id
  managed_login_version = 1 # Classic hosted UI is available with Lite.
}

# Both league routes use this authorizer and require the application play scope.
resource "aws_apigatewayv2_authorizer" "podium_picks" {
  api_id           = aws_apigatewayv2_api.mini_leagues.id
  name             = "podium-picks-users"
  authorizer_type  = "JWT"
  identity_sources = ["$request.header.Authorization"]

  jwt_configuration {
    issuer   = "https://${aws_cognito_user_pool.podium_picks.endpoint}"
    audience = [aws_cognito_user_pool_client.web.id]
  }
}

# Classic hosted UI customization is scoped to this public app client.
resource "aws_cognito_user_pool_ui_customization" "web" {
  user_pool_id = aws_cognito_user_pool_domain.podium_picks.user_pool_id
  client_id    = aws_cognito_user_pool_client.web.id
  css          = file("${path.module}/branding/cognito.css")
  image_file   = filebase64("${path.module}/branding/podium-picks.png")
}
