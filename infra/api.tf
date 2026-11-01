resource "aws_apigatewayv2_api" "mini_leagues" {
  name          = "joliveira-mini-leagues"
  protocol_type = "HTTP"
}

resource "aws_apigatewayv2_integration" "create_league" {
  api_id = aws_apigatewayv2_api.mini_leagues.id

  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.create_league.invoke_arn
  integration_method     = "POST"
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_route" "create_league" {
  api_id = aws_apigatewayv2_api.mini_leagues.id

  route_key          = "POST /leagues"
  target             = "integrations/${aws_apigatewayv2_integration.create_league.id}"
  authorization_type = "AWS_IAM"
}

resource "aws_apigatewayv2_stage" "dev" {
  api_id      = aws_apigatewayv2_api.mini_leagues.id
  name        = "dev"
  auto_deploy = true

  default_route_settings {
    throttling_burst_limit = 5
    throttling_rate_limit  = 2
  }
}

resource "aws_lambda_permission" "api_gateway" {
  statement_id  = "AllowCreateLeagueFromApiGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.create_league.function_name
  principal     = "apigateway.amazonaws.com"

  source_arn = "${aws_apigatewayv2_api.mini_leagues.execution_arn}/dev/POST/leagues"
}

resource "aws_apigatewayv2_route" "get_league" {
  api_id = aws_apigatewayv2_api.mini_leagues.id

  route_key          = "GET /leagues/{leagueId}"
  target             = "integrations/${aws_apigatewayv2_integration.create_league.id}"
  authorization_type = "AWS_IAM"
}

resource "aws_lambda_permission" "api_gateway_get_league" {
  statement_id  = "AllowGetLeagueFromApiGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.create_league.function_name
  principal     = "apigateway.amazonaws.com"

  source_arn = "${aws_apigatewayv2_api.mini_leagues.execution_arn}/dev/GET/leagues/*"
}