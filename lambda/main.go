package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type dynamoDB interface {
	TransactWriteItems(context.Context, *dynamodb.TransactWriteItemsInput, ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
}

type application struct {
	db                 dynamoDB
	leaguesTable       string
	membersTable       string
	initialOrganizerID string
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
		db:                 dynamodb.NewFromConfig(cfg),
		leaguesTable:       tableName,
		membersTable:       os.Getenv("MEMBERS_TABLE_NAME"),
		initialOrganizerID: os.Getenv("INITIAL_ORGANIZER_SUB"),
	}

	lambda.Start(app.route)
}
