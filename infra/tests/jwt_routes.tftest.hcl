mock_provider "aws" {
  override_during = plan
  mock_resource "aws_cognito_user_pool" {
    defaults = { id = "us-east-2_testpool" }
  }
  mock_resource "aws_cognito_user_pool_client" {
    defaults = { id = "testclient" }
  }
  mock_resource "aws_cloudfront_distribution" {
    defaults = { domain_name = "web.example.cloudfront.net" }
  }
  mock_resource "aws_dynamodb_table" {
    defaults = { arn = "arn:aws:dynamodb:us-east-2:123456789012:table/test" }
  }
  mock_resource "aws_cloudwatch_log_group" {
    defaults = { arn = "arn:aws:logs:us-east-2:123456789012:log-group:test" }
  }
  mock_resource "aws_apigatewayv2_authorizer" {
    defaults = { id = "test-authorizer" }
  }
}

override_resource {
  target          = aws_dynamodb_table.mini_league_members
  values          = { arn = "arn:aws:dynamodb:us-east-2:123456789012:table/members" }
  override_during = plan
}

variables {
  account_id            = "123456789012"
  initial_organizer_sub = "organizer-sub"
}

run "league_authorization" {
  command = plan

  assert {
    condition = alltrue([
      aws_apigatewayv2_route.create_league.authorization_type == "JWT",
      aws_apigatewayv2_route.get_league.authorization_type == "JWT",
      aws_apigatewayv2_route.create_league.authorizer_id == aws_apigatewayv2_authorizer.podium_picks.id,
      aws_apigatewayv2_route.get_league.authorizer_id == aws_apigatewayv2_authorizer.podium_picks.id,
      length(aws_apigatewayv2_route.create_league.authorization_scopes) == 1 && contains(aws_apigatewayv2_route.create_league.authorization_scopes, "podium-picks/play"),
      length(aws_apigatewayv2_route.get_league.authorization_scopes) == 1 && contains(aws_apigatewayv2_route.get_league.authorization_scopes, "podium-picks/play"),
    ])
    error_message = "Both league routes must require the Cognito authorizer and play scope."
  }

  assert {
    condition = alltrue([
      aws_lambda_function.create_league.environment[0].variables.MEMBERS_TABLE_NAME == aws_dynamodb_table.mini_league_members.name,
      aws_lambda_function.create_league.environment[0].variables.INITIAL_ORGANIZER_SUB == "organizer-sub",
    ])
    error_message = "Lambda must receive the membership table and configured organizer sub."
  }

  assert {
    condition = anytrue([
      for statement in jsondecode(aws_iam_role_policy.create_league.policy).Statement :
      statement.Resource == aws_dynamodb_table.mini_league_members.arn &&
      toset(statement.Action) == toset(["dynamodb:GetItem", "dynamodb:PutItem"])
      if can(toset(statement.Action))
    ])
    error_message = "Membership permissions must be limited to GetItem and PutItem on the membership table."
  }


  assert {
    condition = alltrue([
      toset(aws_apigatewayv2_api.mini_leagues.cors_configuration[0].allow_origins) == toset(["http://localhost:5173", "https://web.example.cloudfront.net"]),
      toset(aws_apigatewayv2_api.mini_leagues.cors_configuration[0].allow_methods) == toset(["GET", "POST", "OPTIONS"]),
      toset(aws_apigatewayv2_api.mini_leagues.cors_configuration[0].allow_headers) == toset(["authorization", "content-type"]),
    ])
    error_message = "CORS must allow localhost and the website origin with only the required methods and headers."
  }


  assert {
    condition = alltrue([
      aws_s3_bucket_public_access_block.web.block_public_acls,
      aws_s3_bucket_public_access_block.web.block_public_policy,
      aws_s3_bucket_public_access_block.web.ignore_public_acls,
      aws_s3_bucket_public_access_block.web.restrict_public_buckets,
      aws_cloudfront_origin_access_control.web.signing_behavior == "always",
      aws_cloudfront_distribution.web.default_cache_behavior[0].viewer_protocol_policy == "redirect-to-https",
      contains(aws_cognito_user_pool_client.web.callback_urls, "https://web.example.cloudfront.net/auth/callback"),
      contains(aws_cognito_user_pool_client.web.logout_urls, "https://web.example.cloudfront.net/"),
    ])
    error_message = "The website must use a private S3 origin, HTTPS, and matching Cognito URLs."
  }


  assert {
    condition = alltrue([
      aws_cognito_user_pool_ui_customization.web.client_id == aws_cognito_user_pool_client.web.id,
      aws_cognito_user_pool_ui_customization.web.user_pool_id == aws_cognito_user_pool.podium_picks.id,
      aws_cognito_user_pool_ui_customization.web.css == file("${path.module}/branding/cognito.css"),
      aws_cognito_user_pool_ui_customization.web.image_file == filebase64("${path.module}/branding/podium-picks.png"),
    ])
    error_message = "Classic branding must target only the web client and use the reviewed CSS and logo."
  }

}
