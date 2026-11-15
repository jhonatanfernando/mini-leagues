package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"testing"
)

func TestUnknownRoute(t *testing.T) {
	app := application{}
	resp, err := app.route(context.Background(), events.APIGatewayV2HTTPRequest{RouteKey: "DELETE /leagues/{leagueId}"})
	assertResponse(t, resp, err, 404, `{"message":"Route not found"}`)
}
