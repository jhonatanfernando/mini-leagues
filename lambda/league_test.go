package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type fakeDynamo struct {
	transact func(context.Context, *dynamodb.TransactWriteItemsInput) (*dynamodb.TransactWriteItemsOutput, error)
	put      func(context.Context, *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
	get      func(context.Context, *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
}

func (f fakeDynamo) TransactWriteItems(ctx context.Context, in *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	return f.transact(ctx, in)
}
func (f fakeDynamo) PutItem(ctx context.Context, in *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return f.put(ctx, in)
}
func (f fakeDynamo) GetItem(ctx context.Context, in *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return f.get(ctx, in)
}

func TestCreateLeagueOrganizer(t *testing.T) {
	event := events.APIGatewayV2HTTPRequest{Body: `{"leagueId":"L","name":"Name","displayName":" Organizer "}`, RequestContext: events.APIGatewayV2HTTPRequestContext{Authorizer: jwtAuthorizer(map[string]string{"token_use": "access", "sub": "organizer"})}}
	ctx := context.Background()
	called := false
	app := application{leaguesTable: "leagues", membersTable: "members", initialOrganizerID: "organizer", db: fakeDynamo{transact: func(got context.Context, in *dynamodb.TransactWriteItemsInput) (*dynamodb.TransactWriteItemsOutput, error) {
		called = true
		if got != ctx || len(in.TransactItems) != 2 {
			t.Fatal("incorrect transaction")
		}
		league, member := in.TransactItems[0].Put, in.TransactItems[1].Put
		if aws.ToString(league.TableName) != "leagues" || aws.ToString(member.TableName) != "members" || league.Item["ownerId"].(*types.AttributeValueMemberS).Value != "organizer" || member.Item["memberId"].(*types.AttributeValueMemberS).Value != "organizer" || member.Item["displayName"].(*types.AttributeValueMemberS).Value != "Organizer" || league.ConditionExpression == nil || member.ConditionExpression == nil {
			t.Fatal("incorrect transaction items")
		}
		return &dynamodb.TransactWriteItemsOutput{}, nil
	}}}
	resp, err := app.createLeague(ctx, event)
	assertResponse(t, resp, err, 201, `{"leagueId":"L"}`)
	if !called {
		t.Fatal("transaction not called")
	}
	for _, tc := range []struct {
		name   string
		mutate func(*application, *events.APIGatewayV2HTTPRequest)
		status int
	}{
		{"missing identity", func(a *application, e *events.APIGatewayV2HTTPRequest) { e.RequestContext.Authorizer = nil }, 401},
		{"wrong organizer", func(a *application, e *events.APIGatewayV2HTTPRequest) { a.initialOrganizerID = "other" }, 403},
		{"missing config", func(a *application, e *events.APIGatewayV2HTTPRequest) { a.initialOrganizerID = "" }, 500},
		{"owner supplied", func(a *application, e *events.APIGatewayV2HTTPRequest) {
			e.Body = `{"leagueId":"L","name":"N","displayName":"D","ownerId":"spoof"}`
		}, 400},
		{"invalid id", func(a *application, e *events.APIGatewayV2HTTPRequest) {
			e.Body = `{"leagueId":"bad id","name":"N","displayName":"D"}`
		}, 400},
		{"blank display", func(a *application, e *events.APIGatewayV2HTTPRequest) {
			e.Body = `{"leagueId":"L","name":"N","displayName":" "}`
		}, 400},
		{"long display", func(a *application, e *events.APIGatewayV2HTTPRequest) {
			e.Body = `{"leagueId":"L","name":"N","displayName":"` + strings.Repeat("界", 81) + `"}`
		}, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, e := app, event
			tc.mutate(&a, &e)
			called = false
			r, _ := a.createLeague(ctx, e)
			if r.StatusCode != tc.status || called {
				t.Fatalf("status %d called %v", r.StatusCode, called)
			}
		})
	}
	for _, tc := range []struct {
		failure error
		status  int
	}{{&types.TransactionCanceledException{CancellationReasons: []types.CancellationReason{{Code: aws.String("ConditionalCheckFailed")}}}, 409}, {errors.New("private"), 500}} {
		a := app
		a.db = fakeDynamo{transact: func(context.Context, *dynamodb.TransactWriteItemsInput) (*dynamodb.TransactWriteItemsOutput, error) {
			return nil, tc.failure
		}}
		r, _ := a.createLeague(ctx, event)
		if r.StatusCode != tc.status {
			t.Fatalf("status %d", r.StatusCode)
		}
	}
}

func TestGetLeague(t *testing.T) {
	for _, tc := range []struct {
		name    string
		item    map[string]types.AttributeValue
		failure error
		status  int
		body    string
	}{
		{"success", map[string]types.AttributeValue{"leagueId": &types.AttributeValueMemberS{Value: "L"}, "name": &types.AttributeValueMemberS{Value: "N"}, "ownerId": &types.AttributeValueMemberS{Value: "O"}, "createdAt": &types.AttributeValueMemberS{Value: "2026-11-15T00:00:00Z"}}, nil, 200, `{"leagueId":"L","name":"N","ownerId":"O","createdAt":"2026-11-15T00:00:00Z"}`},
		{"missing", nil, nil, 404, `{"message":"League not found"}`},
		{"service failure", nil, errors.New("private detail"), 500, `{"message":"Unable to retrieve league"}`},
		{"decode failure", map[string]types.AttributeValue{"name": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{}}}, nil, 500, `{"message":"Unable to retrieve league"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			called := false
			app := application{leaguesTable: "leagues", membersTable: "members", db: fakeDynamo{get: func(got context.Context, in *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
				if aws.ToString(in.TableName) == "members" {
					return &dynamodb.GetItemOutput{Item: membershipItem("L", "friend")}, nil
				}
				called = true
				if got != ctx || aws.ToString(in.TableName) != "leagues" || !aws.ToBool(in.ConsistentRead) {
					t.Fatalf("incorrect read options: %+v", in)
				}
				if key, ok := in.Key["leagueId"].(*types.AttributeValueMemberS); !ok || key.Value != "L" {
					t.Fatal("incorrect key")
				}
				return &dynamodb.GetItemOutput{Item: tc.item}, tc.failure
			}}}
			resp, err := app.route(ctx, events.APIGatewayV2HTTPRequest{RouteKey: "GET /leagues/{leagueId}", RequestContext: events.APIGatewayV2HTTPRequestContext{Authorizer: jwtAuthorizer(map[string]string{"token_use": "access", "sub": "friend"})}, PathParameters: map[string]string{"leagueId": " L "}})
			assertResponse(t, resp, err, tc.status, tc.body)
			if !called {
				t.Fatal("read not called")
			}
		})
	}
	for _, id := range []string{"", "  "} {
		app := application{}
		resp, err := app.getLeague(context.Background(), events.APIGatewayV2HTTPRequest{RequestContext: events.APIGatewayV2HTTPRequestContext{Authorizer: jwtAuthorizer(map[string]string{"token_use": "access", "sub": "friend"})}, PathParameters: map[string]string{"leagueId": id}})
		assertResponse(t, resp, err, 400, `{"message":"leagueId is required"}`)
	}
}

func assertResponse(t *testing.T, resp events.APIGatewayV2HTTPResponse, err error, status int, body string) {
	t.Helper()
	if err != nil || resp.StatusCode != status || resp.Body != body || resp.Headers["Content-Type"] != "application/json" || resp.Headers["Cache-Control"] != "no-store" || !json.Valid([]byte(resp.Body)) {
		t.Fatalf("response = %+v, error = %v; want %d %s", resp, err, status, body)
	}
}

func TestGetLeagueIdentifierLimit(t *testing.T) {
	app := application{}
	resp, err := app.getLeague(context.Background(), events.APIGatewayV2HTTPRequest{RequestContext: events.APIGatewayV2HTTPRequestContext{Authorizer: jwtAuthorizer(map[string]string{"token_use": "access", "sub": "friend"})}, PathParameters: map[string]string{"leagueId": strings.Repeat("a", 129)}})
	assertResponse(t, resp, err, 400, `{"message":"leagueId must be at most 128 bytes"}`)
}

func TestCreateLeagueBodyValidation(t *testing.T) {
	valid := `{"leagueId":"L","name":"N","displayName":"D"}`
	for _, tc := range []struct {
		name, body string
		encoded    bool
		status     int
	}{
		{"malformed", "{", false, 400}, {"base64", "!", true, 400},
		{"null", "null", false, 400}, {"array", "[]", false, 400},
		{"trailing", valid + ` {}`, false, 400}, {"invalid UTF-8", "{\"leagueId\":\"\xff\"}", false, 400},
		{"oversized", valid + strings.Repeat(" ", 16385-len(valid)), false, 413},
		{"long id", `{"leagueId":"` + strings.Repeat("a", 129) + `","name":"N","displayName":"D"}`, false, 400},
		{"long name", `{"leagueId":"L","name":"` + strings.Repeat("界", 201) + `","displayName":"D"}`, false, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := application{leaguesTable: "leagues", membersTable: "members", initialOrganizerID: "organizer", db: fakeDynamo{transact: func(context.Context, *dynamodb.TransactWriteItemsInput) (*dynamodb.TransactWriteItemsOutput, error) {
				t.Fatal("invalid request reached database")
				return nil, nil
			}}}
			event := events.APIGatewayV2HTTPRequest{Body: tc.body, IsBase64Encoded: tc.encoded, RequestContext: events.APIGatewayV2HTTPRequestContext{Authorizer: jwtAuthorizer(map[string]string{"token_use": "access", "sub": "organizer"})}}
			resp, err := app.createLeague(context.Background(), event)
			if err != nil || resp.StatusCode != tc.status {
				t.Fatalf("response %+v, error %v", resp, err)
			}
		})
	}
}
