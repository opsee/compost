package schema

import (
	"errors"
	"fmt"

	opsee_types "github.com/opsee/protobuf/opseeproto/types"
)

var (
	errMissingEmail      = errors.New("missing email.")
	errMissingUserId     = errors.New("missing user id.")
	errMissingCustomerId = errors.New("missing customer id.")
	errUnverified        = errors.New("user's email has not been confirmed.")
	errInactive          = errors.New("user is inactive.")
	errNilUser           = errors.New("user is nil.")
	errNotOpseeAdmin     = opsee_types.NewPermissionsError(opsee_types.OpseeAdmin)
	errNotTeamAdmin      = opsee_types.NewPermissionsError("admin")
	errNoRead            = opsee_types.NewPermissionsError("read target resource")
	errNoModify          = opsee_types.NewPermissionsError("modify target resource")
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

	if user.Active == false {
		return errInactive
	}

	if user.Status == "inactive" {
		return errInactive
	}

	return nil
}

func (user *User) CheckActiveStatus() error {
	if user.Status == "active" {
		return nil
	}
	return errInactive
}

// Returns true if user is a Opsee Admin
func (user *User) IsOpseeAdmin() bool {
	if user.Admin {
		return true
	}
	return false
}

// Returs the value of a single permission t/f for a user
func (user *User) HasPermission(pname string) bool {
	return user.HasPermissions(pname)
}

// Returns true if a user has all of the requested permissions
func (user *User) HasPermissions(pnames ...string) bool {
	if err := user.CheckActiveStatus(); err != nil {
		return false
	}

	if user.IsOpseeAdmin() || len(pnames) == 0 {
		return true
	}
	return user.Perms.TestFlags(pnames...)
}

// Returs the value of a single permission t/f for a user
func (user *User) CheckPermission(pname string) error {
	errs := user.CheckPermissions(pname)
	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

// Checks a number of permissions and returns permission errors
func (user *User) CheckPermissions(pnames ...string) []error {
	err := user.CheckActiveStatus()
	if err != nil {
		return []error{err}
	}

	// Opsee Admins can do anything
	if user.IsOpseeAdmin() {
		return nil
	}

	var errors []error
	for _, pname := range pnames {
		if user.Perms.TestFlag(pname) {
			continue
		} else {
			errors = append(errors, fmt.Errorf("missing permission: %v", pname))
		}
	}
	return errors
}
