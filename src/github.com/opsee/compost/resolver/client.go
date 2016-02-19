// Package resolver provides a unified interface to query backend services.
package resolver

import (
	"github.com/opsee/basic/clients/bartnet"
	"github.com/opsee/basic/clients/beavis"
	"github.com/opsee/basic/schema"
)

type Client interface {
	ResolveChecks(user *schema.User) ([]*schema.Check, error)
}

type ClientConfig struct {
	Bartnet string
	Beavis  string
}

type client struct {
	Bartnet bartnet.Client
	Beavis  beavis.Client
}

func NewClient(config ClientConfig) Client {
	return &client{
		Bartnet: bartnet.New(config.Bartnet),
		Beavis:  beavis.New(config.Beavis),
	}
}
