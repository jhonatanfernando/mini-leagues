package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
)

func (app *application) route(ctx context.Context, event events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	switch event.RouteKey {
	case "POST /leagues":
		return app.createLeague(ctx, event)

	case "GET /leagues/{leagueId}":
		return app.getLeague(ctx, event)

	default:
		return jsonResponse(404, map[string]string{
			"message": "Route not found",
		}), nil
	}
}
