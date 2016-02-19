package com

import (
	"errors"
)

var (
	errMissingEmail      = errors.New("missing email.")
	errMissingUserId     = errors.New("missing user id.")
	errMissingCustomerId = errors.New("missing customer id.")
	errUnverified        = errors.New("user's email has not been confirmed.")
	errInactive          = errors.New("user is inactive.")
)

type User struct {
	ID         int    `json:"id" token:"id"`
	CustomerID string `json:"customer_id" token:"customer_id"`
	Email      string `json:"email" token:"email"`
	Name       string `json:"name" token:"name"`
	Verified   bool   `json:"verified" token:"verified"`
	Admin      bool   `json:"admin" token:"admin"`
	Active     bool   `json:"active" token:"active"`
}

func (user *User) Validate() error {
	if user.ID == 0 {
		return errMissingUserId
	}

	if user.CustomerID == "" {
		return errMissingCustomerId
	}

	if user.Email == "" {
		return errMissingEmail
	}

	if user.Verified == false {
		return errUnverified
	}

	if user.Active == false {
		return errInactive
	}

	return nil
}
