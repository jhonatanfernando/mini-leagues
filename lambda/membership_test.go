package main

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func membershipItem(leagueID, memberID string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"leagueId": &types.AttributeValueMemberS{Value: leagueID},
		"memberId": &types.AttributeValueMemberS{Value: memberID},
	}
}

func TestGetLeagueRequiresIdentity(t *testing.T) {
	for _, claims := range []map[string]string{nil, {"sub": "friend", "token_use": "id"}} {
		called := false
		app := application{db: fakeDynamo{get: func(context.Context, *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
			called = true
			return nil, errors.New("must not read")
		}}}
		event := events.APIGatewayV2HTTPRequest{PathParameters: map[string]string{"leagueId": "L"}, Headers: map[string]string{"X-User-ID": "owner"}, Body: `{"ownerId":"owner"}`}
		if claims != nil {
			event.RequestContext.Authorizer = jwtAuthorizer(claims)
		}
		resp, err := app.getLeague(context.Background(), event)
		assertResponse(t, resp, err, 401, `{"message":"Unauthorized"}`)
		if called {
			t.Fatal("unauthenticated request reached DynamoDB")
		}
	}
}

func TestGetLeagueMembership(t *testing.T) {
	for _, tc := range []struct {
		name    string
		item    map[string]types.AttributeValue
		failure error
		status  int
	}{
		{"member", membershipItem("L", "friend"), nil, 200},
		{"nonmember", nil, nil, 403},
		{"service failure", nil, errors.New("private service detail"), 500},
		{"decode failure", map[string]types.AttributeValue{"memberId": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{}}}, nil, 500},
		{"missing member key", map[string]types.AttributeValue{"leagueId": &types.AttributeValueMemberS{Value: "L"}}, nil, 500},
		{"wrong member", membershipItem("L", "someone-else"), nil, 500},
		{"wrong league", membershipItem("other", "friend"), nil, 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			membershipReads, leagueReads := 0, 0
			app := application{leaguesTable: "leagues", membersTable: "members", db: fakeDynamo{get: func(got context.Context, in *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
				if got != ctx || !aws.ToBool(in.ConsistentRead) {
					t.Fatal("missing original context or consistent read")
				}
				switch aws.ToString(in.TableName) {
				case "members":
					membershipReads++
					if len(in.Key) != 2 {
						t.Fatalf("unexpected membership key: %+v", in.Key)
					}
					for key, want := range map[string]string{"leagueId": "L", "memberId": "friend"} {
						value, ok := in.Key[key].(*types.AttributeValueMemberS)
						if !ok || value.Value != want {
							t.Fatalf("incorrect %s key", key)
						}
					}
					return &dynamodb.GetItemOutput{Item: tc.item}, tc.failure
				case "leagues":
					leagueReads++
					if membershipReads != 1 || tc.status != 200 {
						t.Fatal("league read before authorization")
					}
					return &dynamodb.GetItemOutput{Item: map[string]types.AttributeValue{
						"leagueId": &types.AttributeValueMemberS{Value: "L"}, "ownerId": &types.AttributeValueMemberS{Value: "owner"},
					}}, nil
				default:
					t.Fatalf("unexpected table %q", aws.ToString(in.TableName))
					return nil, nil
				}
			}}}
			event := events.APIGatewayV2HTTPRequest{RouteKey: "GET /leagues/{leagueId}", PathParameters: map[string]string{"leagueId": " L ", "memberId": "owner"}, Headers: map[string]string{"X-User-ID": "owner"}, Body: `{"ownerId":"owner"}`,
				RequestContext: events.APIGatewayV2HTTPRequestContext{Authorizer: jwtAuthorizer(map[string]string{"token_use": "access", "sub": "friend"})}}
			resp, err := app.route(ctx, event)
			switch tc.status {
			case 200:
				assertResponse(t, resp, err, 200, `{"leagueId":"L","name":"","ownerId":"owner","createdAt":""}`)
			case 403:
				assertResponse(t, resp, err, 403, `{"message":"Forbidden"}`)
			case 500:
				assertResponse(t, resp, err, 500, `{"message":"Unable to retrieve league"}`)
			}
			if membershipReads != 1 {
				t.Fatalf("membership reads = %d", membershipReads)
			}
			if tc.status != 200 && leagueReads != 0 {
				t.Fatal("unauthorized request read league")
			}
			if resp.Headers["Cache-Control"] != "no-store" {
				t.Fatal("protected response must not be cached")
			}
		})
	}
}

func TestGetLeagueMissingMembershipConfiguration(t *testing.T) {
	app := application{}
	event := events.APIGatewayV2HTTPRequest{PathParameters: map[string]string{"leagueId": "L"}, RequestContext: events.APIGatewayV2HTTPRequestContext{Authorizer: jwtAuthorizer(map[string]string{"token_use": "access", "sub": "friend"})}}
	resp, err := app.getLeague(context.Background(), event)
	assertResponse(t, resp, err, 500, `{"message":"Unable to retrieve league"}`)
}
