// Package resolver provides a unified interface to query backend services.
package resolver

import (
	"crypto/tls"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	etcd "github.com/coreos/etcd/client"
	"github.com/opsee/basic/clients/bartnet"
	"github.com/opsee/basic/clients/beavis"
	"github.com/opsee/basic/clients/hugs"
	opsee "github.com/opsee/basic/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type ClientConfig struct {
	SkipVerify bool
	Bartnet    string
	Beavis     string
	Spanx      string
	Vape       string
	Keelhaul   string
	Bezos      string
	Hugs       string
	Etcd       string
}

type Client struct {
	Bartnet  bartnet.Client
	Beavis   beavis.Client
	Spanx    opsee.SpanxClient
	Vape     opsee.VapeClient
	Keelhaul opsee.KeelhaulClient
	Hugs     hugs.Client
	Bezos    opsee.BezosClient
	Dynamo   *dynamodb.DynamoDB
	EtcdKeys etcd.KeysAPI
}

func NewClient(config ClientConfig) (*Client, error) {
	spanxConn, err := grpcConn(config.Spanx, config.SkipVerify)
	if err != nil {
		return nil, err
	}

	vapeConn, err := grpcConn(config.Vape, config.SkipVerify)
	if err != nil {
		return nil, err
	}

	keelhaulConn, err := grpcConn(config.Keelhaul, config.SkipVerify)
	if err != nil {
		return nil, err
	}

	bezosConn, err := grpcConn(config.Bezos, config.SkipVerify)
	if err != nil {
		return nil, err
	}

	etcdClient, err := etcd.New(etcd.Config{
		Endpoints:               []string{config.Etcd},
		Transport:               etcd.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		Bartnet:  bartnet.New(config.Bartnet),
		Beavis:   beavis.New(config.Beavis),
		Spanx:    opsee.NewSpanxClient(spanxConn),
		Vape:     opsee.NewVapeClient(vapeConn),
		Keelhaul: opsee.NewKeelhaulClient(keelhaulConn),
		Hugs:     hugs.New(config.Hugs),
		Bezos:    opsee.NewBezosClient(bezosConn),
		Dynamo:   dynamodb.New(session.New(aws.NewConfig().WithRegion("us-west-2"))),
		EtcdKeys: etcd.NewKeysAPI(etcdClient),
	}, nil
}

func grpcConn(addr string, skipVerify bool) (*grpc.ClientConn, error) {
	return grpc.Dial(
		addr,
		grpc.WithTransportCredentials(
			credentials.NewTLS(&tls.Config{
				InsecureSkipVerify: skipVerify,
			}),
		),
	)
}
