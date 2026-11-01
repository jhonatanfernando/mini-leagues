package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type League struct {
	LeagueID  string `json:"leagueId" dynamodbav:"leagueId"`
	Name      string `json:"name" dynamodbav:"name"`
	OwnerID   string `json:"ownerId" dynamodbav:"ownerId"`
	CreatedAt string `json:"createdAt" dynamodbav:"createdAt"`
}

type CreateLeagueRequest struct {
	LeagueID string `json:"leagueId"`
	Name     string `json:"name"`
	OwnerID  string `json:"ownerId"`
}

type CreateLeagueResponse struct {
	LeagueID string `json:"leagueId"`
}

func (app *application) createLeague(ctx context.Context, event events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	body := []byte(event.Body)

	if event.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(event.Body)
		if err != nil {
			return jsonResponse(400, map[string]string{
				"message": "Invalid request body",
			}), nil
		}
		body = decoded
	}

	var request CreateLeagueRequest
	if err := json.Unmarshal(body, &request); err != nil {
		return jsonResponse(400, map[string]string{
			"message": "Invalid JSON",
		}), nil
	}

	request.LeagueID = strings.TrimSpace(request.LeagueID)
	request.Name = strings.TrimSpace(request.Name)
	request.OwnerID = strings.TrimSpace(request.OwnerID)

	if request.LeagueID == "" || request.Name == "" || request.OwnerID == "" {
		return jsonResponse(400, map[string]string{
			"message": "leagueId, name and ownerId are required",
		}), nil
	}

	league := League{
		LeagueID:  request.LeagueID,
		Name:      request.Name,
		OwnerID:   request.OwnerID,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	item, err := attributevalue.MarshalMap(league)
	if err != nil {
		log.Printf("encode league failed: %v", err)
		return jsonResponse(500, map[string]string{
			"message": "Unable to create league",
		}), nil
	}

	_, err = app.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(app.leaguesTable),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(leagueId)"),
	})

	if err != nil {
		var conflict *types.ConditionalCheckFailedException
		if errors.As(err, &conflict) {
			return jsonResponse(409, map[string]string{
				"message": "League already exists",
			}), nil
		}

		log.Printf("create league failed: %v", err)
		return jsonResponse(500, map[string]string{
			"message": "Unable to create league",
		}), nil
	}

	return jsonResponse(201, CreateLeagueResponse{
		LeagueID: league.LeagueID,
	}), nil
}

func (app *application) getLeague(ctx context.Context, event events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	leagueID := strings.TrimSpace(event.PathParameters["leagueId"])

	if leagueID == "" {
		return jsonResponse(400, map[string]string{
			"message": "leagueId is required",
		}), nil
	}

	result, err := app.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(app.leaguesTable),
		Key: map[string]types.AttributeValue{
			"leagueId": &types.AttributeValueMemberS{
				Value: leagueID,
			},
		},
		ConsistentRead: aws.Bool(true),
	})

	if err != nil {
		log.Printf("get league failed: %v", err)
		return jsonResponse(500, map[string]string{
			"message": "Unable to retrieve league",
		}), nil
	}

	if len(result.Item) == 0 {
		return jsonResponse(404, map[string]string{
			"message": "League not found",
		}), nil
	}

	var league League
	if err := attributevalue.UnmarshalMap(result.Item, &league); err != nil {
		log.Printf("decode league failed: %v", err)
		return jsonResponse(500, map[string]string{
			"message": "Unable to retrieve league",
		}), nil
	}

	return jsonResponse(200, league), nil
}
