package resolver

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/opsee/basic/schema"
	opsee "github.com/opsee/basic/service"
	"golang.org/x/net/context"
	"fmt"
)

func (c *Client) awsSession(ctx context.Context, user *schema.User, region string) (*session.Session, error) {
	if err := user.Validate(); err != nil {
		return nil, fmt.Errorf("user invalid: %#v", user)
	}

	creds, err := c.Spanx.GetCredentials(ctx, &opsee.GetCredentialsRequest{User: user})
	if err != nil {
		return nil, err
	}

	sess := session.New(&aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			creds.Credentials.GetAccessKeyID(),
			creds.Credentials.GetSecretAccessKey(),
			creds.Credentials.GetSessionToken(),
		),
	})

	return sess, nil
}
