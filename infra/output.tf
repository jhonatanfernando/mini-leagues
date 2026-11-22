output "mini_leagues_table_name" {
  value = aws_dynamodb_table.mini_leagues.name
}

output "mini_league_members_table_name" {
  value = aws_dynamodb_table.mini_league_members.name
}

output "create_league_url" {
  value = "${aws_apigatewayv2_stage.dev.invoke_url}/leagues"
}
output "web_auth" {
  description = "Public browser authentication settings. No credentials or client secret."
  value = {
    user_pool_id  = aws_cognito_user_pool.podium_picks.id
    client_id     = aws_cognito_user_pool_client.web.id
    issuer        = "https://${aws_cognito_user_pool.podium_picks.endpoint}"
    domain        = "https://${aws_cognito_user_pool_domain.podium_picks.domain}.auth.${var.region}.amazoncognito.com"
    scopes        = ["openid", "email", "${aws_cognito_resource_server.podium_picks.identifier}/play"]
    callback_urls = distinct(concat(var.auth_callback_urls, ["${local.web_origin}/auth/callback"]))
    logout_urls   = distinct(concat(var.auth_logout_urls, ["${local.web_origin}/"]))
  }
}

output "web_url" {
  description = "Public HTTPS website URL."
  value       = local.web_origin
}
