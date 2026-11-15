package main

import (
	"errors"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestAuthenticatedUserID(t *testing.T) {
	for _, tc := range []struct {
		name       string
		authorizer *events.APIGatewayV2HTTPRequestContextAuthorizerDescription
		wantID     string
	}{
		{name: "no authorizer"},
		{name: "empty authorizer", authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{}},
		{name: "IAM is not application identity", authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{IAM: &events.APIGatewayV2HTTPRequestContextAuthorizerIAMDescription{UserID: "aws-user"}}},
		{name: "missing claims", authorizer: jwtAuthorizer(nil)},
		{name: "missing subject", authorizer: jwtAuthorizer(map[string]string{"token_use": "access"})},
		{name: "empty subject", authorizer: jwtAuthorizer(map[string]string{"token_use": "access", "sub": ""})},
		{name: "blank subject", authorizer: jwtAuthorizer(map[string]string{"token_use": "access", "sub": " \t\n"})},
		{name: "padded subject", authorizer: jwtAuthorizer(map[string]string{"token_use": "access", "sub": " user-123 "})},
		{name: "ID token", authorizer: jwtAuthorizer(map[string]string{"token_use": "id", "sub": "user-123"})},
		{name: "missing token use", authorizer: jwtAuthorizer(map[string]string{"sub": "user-123"})},
		{name: "invalid token use", authorizer: jwtAuthorizer(map[string]string{"token_use": "ACCESS", "sub": "user-123"})},
		{name: "email is not identity", authorizer: jwtAuthorizer(map[string]string{"token_use": "access", "email": "friend@example.com"})},
		{name: "valid access token", authorizer: jwtAuthorizer(map[string]string{"token_use": "access", "sub": "user-123"}), wantID: "user-123"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			event := events.APIGatewayV2HTTPRequest{RequestContext: events.APIGatewayV2HTTPRequestContext{Authorizer: tc.authorizer}}
			got, err := authenticatedUserID(event)
			if tc.wantID == "" {
				if got != "" || !errors.Is(err, errUnauthenticated) {
					t.Fatalf("identity = %q, error = %v; want no identity and errUnauthenticated", got, err)
				}
				if err.Error() != "Unauthorized" {
					t.Fatalf("authentication failure exposed unexpected detail: %v", err)
				}
			} else if got != tc.wantID || err != nil {
				t.Fatalf("identity = %q, error = %v; want %q", got, err, tc.wantID)
			}
		})
	}
}

func TestAuthenticatedUserIDIgnoresCallerSuppliedIdentity(t *testing.T) {
	for _, trusted := range []bool{false, true} {
		t.Run(map[bool]string{false: "no verified identity", true: "verified identity"}[trusted], func(t *testing.T) {
			event := events.APIGatewayV2HTTPRequest{
				Body:                  `{"sub":"attacker","ownerId":"attacker","memberId":"attacker"}`,
				Headers:               map[string]string{"Authorization": "Bearer unverified-token", "X-User-ID": "attacker", "sub": "attacker"},
				QueryStringParameters: map[string]string{"sub": "attacker"},
				PathParameters:        map[string]string{"memberId": "attacker"},
			}
			if trusted {
				event.RequestContext.Authorizer = jwtAuthorizer(map[string]string{"token_use": "access", "sub": "trusted-member"})
			}
			got, err := authenticatedUserID(event)
			if trusted {
				if got != "trusted-member" || err != nil {
					t.Fatalf("identity = %q, error = %v", got, err)
				}
			} else if got != "" || !errors.Is(err, errUnauthenticated) {
				t.Fatalf("untrusted data supplied identity %q, error = %v", got, err)
			}
		})
	}
}

func jwtAuthorizer(claims map[string]string) *events.APIGatewayV2HTTPRequestContextAuthorizerDescription {
	return &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{Claims: claims}}
}
