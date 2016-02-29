// Package resolver provides a unified interface to query backend services.
package resolver

import (
	"crypto/tls"
	"github.com/opsee/basic/clients/bartnet"
	"github.com/opsee/basic/clients/beavis"
	"github.com/opsee/basic/schema"
	opsee "github.com/opsee/basic/service"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Client interface {
	ListChecks(context.Context, *schema.User) ([]*schema.Check, error)
	GetCredentials(context.Context, string) (*opsee.GetCredentialsResponse, error)
	ListCustomers(context.Context, *opsee.ListUsersRequest) (*opsee.ListCustomersResponse, error)
	GetUser(context.Context, *opsee.GetUserRequest) (*opsee.GetUserResponse, error)
}

type ClientConfig struct {
	Bartnet  string
	Beavis   string
	Spanx    string
	Vape     string
	Keelhaul string
}

type client struct {
	Bartnet  bartnet.Client
	Beavis   beavis.Client
	Spanx    opsee.SpanxClient
	Vape     opsee.VapeClient
	Keelhaul opsee.KeelhaulClient
}

func NewClient(config ClientConfig) (Client, error) {
	spanxConn, err := grpcConn(config.Spanx)
	if err != nil {
		return nil, err
	}

	vapeConn, err := grpcConn(config.Vape)
	if err != nil {
		return nil, err
	}

	keelhaulConn, err := grpcConn(config.Keelhaul)
	if err != nil {
		return nil, err
	}

	return &client{
		Bartnet:  bartnet.New(config.Bartnet),
		Beavis:   beavis.New(config.Beavis),
		Spanx:    opsee.NewSpanxClient(spanxConn),
		Vape:     opsee.NewVapeClient(vapeConn),
		Keelhaul: opsee.NewKeelhaulClient(keelhaulConn),
	}, nil
}

func grpcConn(addr string) (*grpc.ClientConn, error) {
	return grpc.Dial(
		addr,
		grpc.WithTransportCredentials(
			credentials.NewTLS(&tls.Config{}),
		),
	)
}
