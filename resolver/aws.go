package resolver

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/opsee/basic/schema"
	opsee "github.com/opsee/basic/service"
	"github.com/opsee/spanx/spanxcreds"
	"golang.org/x/net/context"
)

func (c *Client) awsSession(ctx context.Context, user *schema.User, region string) (*session.Session, error) {
	if err := user.Validate(); err != nil {
		return nil, fmt.Errorf("user invalid: %#v", user)
	}

	sess := session.New(&aws.Config{
		Region:      aws.String(region),
		Credentials: spanxcreds.NewSpanxCredentials(user, c.Spanx),
	})

	return sess, nil
}
