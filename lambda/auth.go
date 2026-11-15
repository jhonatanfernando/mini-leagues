package main

import (
	"errors"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// Handlers using this helper must map this error to HTTP 401.
var errUnauthenticated = errors.New("Unauthorized")

// authenticatedUserID reads only API Gateway's verified JWT request context.
// It relies on the route's JWT authorizer to validate signature, issuer,
// audience, expiry, and the required application scope. It does not parse or
// trust raw Authorization headers, request bodies, or IAM identities.
// Do not use it behind an unauthenticated integration that permits callers to
// supply the Lambda event directly.
func authenticatedUserID(event events.APIGatewayV2HTTPRequest) (string, error) {
	authorizer := event.RequestContext.Authorizer
	if authorizer == nil || authorizer.JWT == nil {
		return "", errUnauthenticated
	}

	claims := authorizer.JWT.Claims
	if claims["token_use"] != "access" {
		return "", errUnauthenticated
	}

	subject := claims["sub"]
	// Reject malformed subjects rather than normalizing them into another ID.
	if subject == "" || strings.TrimSpace(subject) != subject {
		return "", errUnauthenticated
	}
	return subject, nil
}
