package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type fakeDynamo struct {
	put func(context.Context, *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
	get func(context.Context, *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
}

func (f fakeDynamo) PutItem(ctx context.Context, in *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return f.put(ctx, in)
}
func (f fakeDynamo) GetItem(ctx context.Context, in *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return f.get(ctx, in)
}

func TestCreateLeagueValidation(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		encoded    bool
		message    string
	}{
		{"json", "{", false, "Invalid JSON"},
		{"base64", "!", true, "Invalid request body"},
		{"missing", `{}`, false, "leagueId, name and ownerId are required"},
		{"blank id", `{"leagueId":" ","name":"N","ownerId":"O"}`, false, "leagueId, name and ownerId are required"},
		{"blank name", `{"leagueId":"L","name":" ","ownerId":"O"}`, false, "leagueId, name and ownerId are required"},
		{"blank owner", `{"leagueId":"L","name":"N","ownerId":" "}`, false, "leagueId, name and ownerId are required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := application{}
			resp, err := app.createLeague(context.Background(), events.APIGatewayV2HTTPRequest{Body: tc.body, IsBase64Encoded: tc.encoded})
			assertResponse(t, resp, err, 400, `{"message":"`+tc.message+`"}`)
		})
	}
}

func TestCreateLeague(t *testing.T) {
	for _, encoded := range []bool{false, true} {
		t.Run(map[bool]string{false: "json", true: "base64"}[encoded], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			called := false
			app := application{leaguesTable: "leagues", db: fakeDynamo{put: func(got context.Context, in *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
				called = true
				if got != ctx || aws.ToString(in.TableName) != "leagues" || aws.ToString(in.ConditionExpression) != "attribute_not_exists(leagueId)" {
					t.Fatalf("incorrect write context/options: %+v", in)
				}
				for key, want := range map[string]string{"leagueId": "L", "name": "Name", "ownerId": "O"} {
					if value, ok := in.Item[key].(*types.AttributeValueMemberS); !ok || value.Value != want {
						t.Fatalf("incorrect %s: %v", key, in.Item[key])
					}
				}
				timestamp, ok := in.Item["createdAt"].(*types.AttributeValueMemberS)
				if !ok {
					t.Fatal("missing timestamp")
				}
				if _, err := time.Parse(time.RFC3339, timestamp.Value); err != nil {
					t.Fatal(err)
				}
				return &dynamodb.PutItemOutput{}, nil
			}}}
			body := `{"leagueId":" L ","name":" Name ","ownerId":" O "}`
			if encoded {
				body = base64.StdEncoding.EncodeToString([]byte(body))
			}
			resp, err := app.route(ctx, events.APIGatewayV2HTTPRequest{RouteKey: "POST /leagues", Body: body, IsBase64Encoded: encoded})
			assertResponse(t, resp, err, 201, `{"leagueId":"L"}`)
			if !called {
				t.Fatal("write not called")
			}
		})
	}
	for _, tc := range []struct {
		name    string
		failure error
		status  int
		message string
	}{
		{"conflict", &types.ConditionalCheckFailedException{}, 409, "League already exists"},
		{"failure", errors.New("private service detail"), 500, "Unable to create league"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := application{db: fakeDynamo{put: func(context.Context, *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) { return nil, tc.failure }}}
			resp, err := app.createLeague(context.Background(), events.APIGatewayV2HTTPRequest{Body: `{"leagueId":"L","name":"N","ownerId":"O"}`})
			assertResponse(t, resp, err, tc.status, `{"message":"`+tc.message+`"}`)
		})
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
			app := application{leaguesTable: "leagues", db: fakeDynamo{get: func(got context.Context, in *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
				called = true
				if got != ctx || aws.ToString(in.TableName) != "leagues" || !aws.ToBool(in.ConsistentRead) {
					t.Fatalf("incorrect read options: %+v", in)
				}
				if key, ok := in.Key["leagueId"].(*types.AttributeValueMemberS); !ok || key.Value != "L" {
					t.Fatal("incorrect key")
				}
				return &dynamodb.GetItemOutput{Item: tc.item}, tc.failure
			}}}
			resp, err := app.route(ctx, events.APIGatewayV2HTTPRequest{RouteKey: "GET /leagues/{leagueId}", PathParameters: map[string]string{"leagueId": " L "}})
			assertResponse(t, resp, err, tc.status, tc.body)
			if !called {
				t.Fatal("read not called")
			}
		})
	}
	for _, id := range []string{"", "  "} {
		app := application{}
		resp, err := app.getLeague(context.Background(), events.APIGatewayV2HTTPRequest{PathParameters: map[string]string{"leagueId": id}})
		assertResponse(t, resp, err, 400, `{"message":"leagueId is required"}`)
	}
}

func assertResponse(t *testing.T, resp events.APIGatewayV2HTTPResponse, err error, status int, body string) {
	t.Helper()
	if err != nil || resp.StatusCode != status || resp.Body != body || resp.Headers["Content-Type"] != "application/json" || !json.Valid([]byte(resp.Body)) {
		t.Fatalf("response = %+v, error = %v; want %d %s", resp, err, status, body)
	}
}
