package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Membership uses the existing table's leagueId/memberId key. MemberID is the
// authenticated Cognito sub; a league owner must have a membership row too.
type Membership struct {
	LeagueID    string `json:"leagueId" dynamodbav:"leagueId"`
	MemberID    string `json:"memberId" dynamodbav:"memberId"`
	DisplayName string `json:"displayName" dynamodbav:"displayName"`
	JoinedAt    string `json:"joinedAt" dynamodbav:"joinedAt"`
}

var errForbidden = errors.New("Forbidden")

func (app *application) requireMembership(ctx context.Context, leagueID, memberID string) error {
	if app.membersTable == "" {
		return errors.New("MEMBERS_TABLE_NAME is required for membership checks")
	}
	result, err := app.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(app.membersTable),
		Key: map[string]types.AttributeValue{
			"leagueId": &types.AttributeValueMemberS{Value: leagueID},
			"memberId": &types.AttributeValueMemberS{Value: memberID},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("read membership: %w", err)
	}
	if result == nil {
		return errors.New("missing membership response")
	}
	if len(result.Item) == 0 {
		return errForbidden
	}
	var member Membership
	if err := attributevalue.UnmarshalMap(result.Item, &member); err != nil {
		return fmt.Errorf("decode membership: %w", err)
	}
	// Malformed data must never grant access. No email/ownerId fallback exists.
	if member.LeagueID != leagueID || member.MemberID != memberID {
		return errors.New("membership keys do not match request")
	}
	return nil
}
