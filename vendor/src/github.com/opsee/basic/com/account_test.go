package com

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAccountRoleARN(t *testing.T) {
	account := &Account{
		ID:         666666666666,
		CustomerID: "deadbeef",
	}
	assert.Equal(t, "arn:aws:iam::666666666666:role/opsee-role-deadbeef", account.RoleARN())

	account = &Account{
		ID:         666666,
		CustomerID: "deadbeef",
	}
	assert.Equal(t, "arn:aws:iam::000000666666:role/opsee-role-deadbeef", account.RoleARN())
}
