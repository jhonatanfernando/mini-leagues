package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type application struct {
	db           *dynamodb.Client
	leaguesTable string
}

func main() {
	tableName := os.Getenv("LEAGUES_TABLE_NAME")
	if tableName == "" {
		log.Fatal("LEAGUES_TABLE_NAME is required")
	}

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("load AWS configuration: %v", err)
	}

	app := &application{
		db:           dynamodb.NewFromConfig(cfg),
		leaguesTable: tableName,
	}

	lambda.Start(app.route)
}
