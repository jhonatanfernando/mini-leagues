data "aws_caller_identity" "inspector_setup" {}

data "aws_region" "inspector_setup" {}

data "aws_iam_role" "inspector_source" {
  name = "AccountFullAccessRole"
}

resource "aws_iam_role" "mini_leagues_inspector" {
  name                 = "mini-leagues-inspector"
  max_session_duration = 3600

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        AWS = "arn:aws:iam::${data.aws_caller_identity.inspector_setup.account_id}:root"
      }
      Action = "sts:AssumeRole"
      Condition = {
        ArnEquals = {
          "aws:PrincipalArn" = data.aws_iam_role.inspector_source.arn
        }
      }
    }]
  })
}

resource "aws_iam_role_policy" "mini_leagues_inspector" {
  name = "inspect-mini-leagues"
  role = aws_iam_role.mini_leagues_inspector.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "ReadLambdaConfiguration"
        Effect   = "Allow"
        Action   = ["lambda:GetFunctionConfiguration"]
        Resource = aws_lambda_function.create_league.arn
      },
      {
        Sid    = "DescribeProjectTables"
        Effect = "Allow"
        Action = ["dynamodb:DescribeTable"]
        Resource = [
          aws_dynamodb_table.mini_leagues.arn,
          aws_dynamodb_table.mini_league_members.arn
        ]
      },
      {
        Sid    = "ReadProjectLogs"
        Effect = "Allow"
        Action = [
          "logs:DescribeLogStreams",
          "logs:FilterLogEvents",
          "logs:GetLogEvents"
        ]
        Resource = [
          aws_cloudwatch_log_group.create_league.arn,
          "${aws_cloudwatch_log_group.create_league.arn}:*"
        ]
      },
      {
        Sid    = "ReadProjectHttpApi"
        Effect = "Allow"
        Action = ["apigateway:GET"]
        Resource = [
          "arn:aws:apigateway:${data.aws_region.inspector_setup.region}::/apis/${aws_apigatewayv2_api.mini_leagues.id}",
          "arn:aws:apigateway:${data.aws_region.inspector_setup.region}::/apis/${aws_apigatewayv2_api.mini_leagues.id}/*"
        ]
      }
    ]
  })
}

output "inspector_role_arn" {
  value = aws_iam_role.mini_leagues_inspector.arn
}