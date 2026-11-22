mock_provider "aws" {
  override_during = plan
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

}
