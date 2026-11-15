package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Bound input before decoding or writing it. IDs use bytes; display names use
// Unicode code points so non-ASCII names have the same character allowance.
const (
	maxLeagueBodyBytes      = 16 * 1024
	maxLeagueIDBytes        = 128
	maxLeagueNameCharacters = 200
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
	// Bound the encoded representation too, before allocating a decoded buffer.
	bodyLimit := maxLeagueBodyBytes
	if event.IsBase64Encoded {
		bodyLimit = base64.StdEncoding.EncodedLen(maxLeagueBodyBytes)
	}
	if len(event.Body) > bodyLimit {
		return jsonResponse(413, map[string]string{"message": "Request body is too large"}), nil
	}
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

	if len(body) > maxLeagueBodyBytes {
		return jsonResponse(413, map[string]string{"message": "Request body is too large"}), nil
	}

	// A pointer distinguishes a JSON null from an object. A second decode rejects
	// trailing values; DisallowUnknownFields catches misspelled request fields.
	var request *CreateLeagueRequest
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&request)
	if err != nil || request == nil || !utf8.Valid(body) {
		return jsonResponse(400, map[string]string{
			"message": "Invalid JSON",
		}), nil
	}

	if err := decoder.Decode(new(any)); err != io.EOF {
		return jsonResponse(400, map[string]string{"message": "Invalid JSON"}), nil
	}

	request.LeagueID = strings.TrimSpace(request.LeagueID)
	request.Name = strings.TrimSpace(request.Name)
	request.OwnerID = strings.TrimSpace(request.OwnerID)

	if request.LeagueID == "" || request.Name == "" || request.OwnerID == "" {
		return jsonResponse(400, map[string]string{
			"message": "leagueId, name and ownerId are required",
		}), nil
	}

	if len(request.LeagueID) > maxLeagueIDBytes || len(request.OwnerID) > maxLeagueIDBytes {
		return jsonResponse(400, map[string]string{"message": "leagueId and ownerId must be at most 128 bytes"}), nil
	}
	if utf8.RuneCountInString(request.Name) > maxLeagueNameCharacters {
		return jsonResponse(400, map[string]string{"message": "name must be at most 200 characters"}), nil
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

	if len(leagueID) > maxLeagueIDBytes {
		return jsonResponse(400, map[string]string{"message": "leagueId must be at most 128 bytes"}), nil
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
