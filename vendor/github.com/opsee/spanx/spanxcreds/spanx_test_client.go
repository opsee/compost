package spanxcreds

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/opsee/basic/schema/aws/credentials"
	"github.com/opsee/basic/service"
	opsee_types "github.com/opsee/protobuf/opseeproto/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type SpanxTestClient struct {
	ExpiryString       string
	FailGetCredentials bool
}

// Stub
func (t *SpanxTestClient) EnhancedCombatMode(ctx context.Context, in *service.EnhancedCombatModeRequest, opts ...grpc.CallOption) (*service.EnhancedCombatModeResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// Stub
func (t *SpanxTestClient) PutRole(ctx context.Context, in *service.PutRoleRequest, opts ...grpc.CallOption) (*service.PutRoleResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// Return Test Credentials
func (t *SpanxTestClient) GetCredentials(ctx context.Context, in *service.GetCredentialsRequest, opts ...grpc.CallOption) (*service.GetCredentialsResponse, error) {
	if t.FailGetCredentials {
		return nil, fmt.Errorf("Couldn't get credentials.")
	}

	expiryTime, err := time.Parse("2006-01-02T15:04:05Z", t.ExpiryString)
	if err != nil {
		return nil, err
	}

	expiryTimestamp := &opsee_types.Timestamp{}
	err = expiryTimestamp.Scan(expiryTime)
	if err != nil {
		return nil, fmt.Errorf("Couldn't create expiry timestamp")
	}

	return &service.GetCredentialsResponse{
		Credentials: &credentials.Value{
			AccessKeyID:     aws.String("accessKey"),
			SecretAccessKey: aws.String("secret"),
			SessionToken:    aws.String("token"),
		},
		Expires: expiryTimestamp,
	}, nil
}
