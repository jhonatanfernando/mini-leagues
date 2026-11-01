resource "aws_dynamodb_table" "mini_leagues" {
  name         = "joliveira-mini-leagues"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "leagueId"

  attribute {
    name = "leagueId"
    type = "S"
  }

  tags = {
    Owner   = "jhonatan.oliveira"
    Purpose = "Terraform learning"
  }
}

resource "aws_dynamodb_table" "mini_league_members" {
  name         = "joliveira-mini-league-members"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "leagueId"
  range_key    = "memberId"

  attribute {
    name = "leagueId"
    type = "S"
  }

  attribute {
    name = "memberId"
    type = "S"
  }

  tags = {
    Owner   = "jhonatan.oliveira"
    Purpose = "Terraform learning"
  }
}