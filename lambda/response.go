package main

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
)

func jsonResponse(status int, body any) events.APIGatewayV2HTTPResponse {
	data, err := json.Marshal(body)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       `{"message":"Unable to encode response"}`,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: status,
		Body:       string(data),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}
