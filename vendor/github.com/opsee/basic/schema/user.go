package schema

import (
	"errors"

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

	return nil
}

// Can user get target resource
func (user *User) CanRead(target interface{}, requiredPerms ...string) bool {
	//requesting user is opsee admin or has ability to modify
	if user.IsOpseeAdmin() {
		return true
	}

	// check extra, required permissions
	if !user.HasPermissions(requiredPerms...) {
		return false
	}

	switch t := target.(type) {
	case *User:
		// requesting user is target user or is on same team
		if user.Id == t.Id || user.CustomerId == t.CustomerId {
			return true
		}
	case *Team:
		// requesting user is on target team
		if user.CustomerId == t.Id {
			return true
		}
	}
	return false
}

// Can user update or delete target resource
func (user *User) CanModify(target interface{}, requiredPerms ...string) bool {
	//requesting user is opsee admin
	if user.IsOpseeAdmin() {
		return true
	}

	// check extra, required permissions
	if !user.HasPermissions(requiredPerms...) {
		return false
	}

	switch t := target.(type) {
	case *User:
		// requesting user is target user
		if user.Id == t.Id {
			return true
		}
		// requesting user is on same team and is team admin
		if user.CustomerId == t.CustomerId && user.HasPermission("admin") {
			return true
		}
	case *Team:
		// requesting user is on same team and is team admin
		if user.CustomerId == t.Id && user.HasPermission("admin") {
			return true
		}
	}
	return false
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
	if user.IsOpseeAdmin() || len(pnames) == 0 {
		return true
	}

	hasPermissions := true
	for pname, err := range user.Perms.CheckPermissions(pnames...) {
		switch pname {
		case opsee_types.OpseeAdmin:
			// if opsee_types.OpseeAdmin is specified, nothing else matters
			return user.IsOpseeAdmin()
		default:
			if err != nil {
				hasPermissions = false
				break
			}
		}
	}

	return hasPermissions
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
	// Opsee Admins can do anything
	if user.IsOpseeAdmin() {
		return nil
	}

	var errors []error
	for pname, err := range user.Perms.CheckPermissions(pnames...) {
		switch pname {
		case opsee_types.OpseeAdmin:
			if !user.IsOpseeAdmin() {
				errors = append(errors, errNotOpseeAdmin)
			}
		default:
			if err != nil {
				errors = append(errors, err)
			}
		}
	}

	return errors
}
