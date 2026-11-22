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
	LeagueID    string `json:"leagueId"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type CreateLeagueResponse struct {
	LeagueID string `json:"leagueId"`
}

func (app *application) createLeague(ctx context.Context, event events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	memberID, err := authenticatedUserID(event)
	if err != nil {
		return jsonResponse(401, map[string]string{"message": "Unauthorized"}), nil
	}
	if app.initialOrganizerID == "" || app.membersTable == "" || app.leaguesTable == "" {
		return jsonResponse(500, map[string]string{"message": "Unable to create league"}), nil
	}
	if memberID != app.initialOrganizerID {
		return jsonResponse(403, map[string]string{"message": "Forbidden"}), nil
	}
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
	err = decoder.Decode(&request)
	if err != nil || request == nil || !utf8.Valid(body) {
		return jsonResponse(400, map[string]string{
			"message": "Invalid JSON",
		}), nil
	}

	if err := decoder.Decode(new(any)); err != io.EOF {
		return jsonResponse(400, map[string]string{"message": "Invalid JSON"}), nil
	}

	request.Name = strings.TrimSpace(request.Name)
	request.DisplayName = strings.TrimSpace(request.DisplayName)
	validID := len(request.LeagueID) > 0 && len(request.LeagueID) <= maxLeagueIDBytes
	for _, c := range request.LeagueID {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			validID = false
		}
	}
	if !validID || request.Name == "" || utf8.RuneCountInString(request.Name) > maxLeagueNameCharacters || request.DisplayName == "" || utf8.RuneCountInString(request.DisplayName) > 80 {
		return jsonResponse(400, map[string]string{"message": "Invalid league details"}), nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	league := League{LeagueID: request.LeagueID, Name: request.Name, OwnerID: memberID, CreatedAt: now}
	member := Membership{LeagueID: request.LeagueID, MemberID: memberID, DisplayName: request.DisplayName, JoinedAt: now}
	item, err := attributevalue.MarshalMap(league)
	if err != nil {
		return jsonResponse(500, map[string]string{"message": "Unable to create league"}), nil
	}
	memberItem, err := attributevalue.MarshalMap(member)
	if err != nil {
		return jsonResponse(500, map[string]string{"message": "Unable to create league"}), nil
	}
	_, err = app.db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: []types.TransactWriteItem{
		{Put: &types.Put{TableName: aws.String(app.leaguesTable), Item: item, ConditionExpression: aws.String("attribute_not_exists(leagueId)")}},
		{Put: &types.Put{TableName: aws.String(app.membersTable), Item: memberItem, ConditionExpression: aws.String("attribute_not_exists(leagueId) AND attribute_not_exists(memberId)")}},
	}})
	if err != nil {
		var canceled *types.TransactionCanceledException
		if errors.As(err, &canceled) {
			for _, reason := range canceled.CancellationReasons {
				if aws.ToString(reason.Code) == "ConditionalCheckFailed" {
					return jsonResponse(409, map[string]string{"message": "League already exists"}), nil
				}
			}
		}
		log.Printf("create league transaction failed: %v", err)
		return jsonResponse(500, map[string]string{"message": "Unable to create league"}), nil
	}

	return jsonResponse(201, CreateLeagueResponse{
		LeagueID: league.LeagueID,
	}), nil
}

func (app *application) getLeague(ctx context.Context, event events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	memberID, err := authenticatedUserID(event)
	if err != nil {
		return jsonResponse(401, map[string]string{"message": "Unauthorized"}), nil
	}

	leagueID := strings.TrimSpace(event.PathParameters["leagueId"])

	if leagueID == "" {
		return jsonResponse(400, map[string]string{
			"message": "leagueId is required",
		}), nil
	}

	if len(leagueID) > maxLeagueIDBytes {
		return jsonResponse(400, map[string]string{"message": "leagueId must be at most 128 bytes"}), nil
	}

	if err := app.requireMembership(ctx, leagueID, memberID); err != nil {
		if errors.Is(err, errForbidden) {
			return jsonResponse(403, map[string]string{"message": "Forbidden"}), nil
		}
		log.Printf("authorize league read failed: %v", err)
		return jsonResponse(500, map[string]string{"message": "Unable to retrieve league"}), nil
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
