package schema

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

func (user *User) Validate() error {
	if user.Id == 0 {
		return errMissingUserId
	}

	if user.CustomerId == "" {
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
