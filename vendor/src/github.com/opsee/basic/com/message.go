package com

import (
	"errors"
)

type Message struct {
	Command    string                 `json:"command"`
	State      string                 `json:"state"`
	Message    string                 `json:"message"`
	Attributes map[string]interface{} `json:"attributes"`
	CustomerID string                 `json:"customer_id"`
	BastionID  string                 `json:"bastion_id"`
}

var (
	errMissingCommand = errors.New("missing command.")
)

func (msg *Message) Validate() error {
	if msg.CustomerID == "" {
		return errMissingCustomerId
	}

	if msg.Command == "" {
		return errMissingCommand
	}

	return nil
}
