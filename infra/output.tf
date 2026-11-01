output "mini_leagues_table_name" {
  value = aws_dynamodb_table.mini_leagues.name
}

output "mini_league_members_table_name" {
  value = aws_dynamodb_table.mini_league_members.name
}

output "create_league_url" {
  value = "${aws_apigatewayv2_stage.dev.invoke_url}/leagues"
}