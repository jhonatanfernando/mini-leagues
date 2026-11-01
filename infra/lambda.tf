resource "aws_iam_role" "create_league" {
  name = "joliveira-create-league"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Service = "lambda.amazonaws.com"
      }
      Action = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy" "create_league" {
  name = "create-league-permissions"
  role = aws_iam_role.create_league.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",
          "dynamodb:GetItem"
        ]
        Resource = aws_dynamodb_table.mini_leagues.arn
      },
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "${aws_cloudwatch_log_group.create_league.arn}:*"
      }
    ]
  })
}

resource "aws_cloudwatch_log_group" "create_league" {
  name              = "/aws/lambda/joliveira-create-league"
  retention_in_days = 7
}

resource "aws_lambda_function" "create_league" {
  function_name = "joliveira-create-league"
  role          = aws_iam_role.create_league.arn

  runtime       = "provided.al2023"
  handler       = "bootstrap"
  architectures = ["x86_64"]

  filename         = "${path.module}/artifacts/function.zip"
  source_code_hash = filebase64sha256("${path.module}/artifacts/function.zip")

  memory_size = 128
  timeout     = 10

  environment {
    variables = {
      LEAGUES_TABLE_NAME = aws_dynamodb_table.mini_leagues.name
    }
  }

  depends_on = [
    aws_iam_role_policy.create_league,
    aws_cloudwatch_log_group.create_league
  ]
}