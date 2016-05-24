package schema

import (
	"sync"

	opsee_types "github.com/opsee/protobuf/opseeproto/types"
)

func init() {
	opsee_types.PermissionsRegistry.Register("user", &opsee_types.PermissionsBitmap{
		map[int]string{
			0: "admin",
			1: "edit",
			2: "billing",
		},
		sync.RWMutex{},
	})

	opsee_types.PermissionsRegistry.Register("team", &opsee_types.PermissionsBitmap{
		map[int]string{
			0: "multi-user",
			1: "multibastion",
			2: "on-site support",
		},
		sync.RWMutex{},
	})
}

// Wrapper for check if requesting user can modify target
func CheckRead(requestor *User, target interface{}, requiredPerms ...string) error {
	if requestor == nil || !requestor.CanRead(target, requiredPerms...) {
		return errNoRead
	}
	return nil
}

// Wrapper for check if requesting user can modify target
func CheckModify(requestor *User, target interface{}, requiredPerms ...string) error {
	if requestor == nil || !requestor.CanModify(target, requiredPerms...) {
		return errNoModify
	}
	return nil
}

// Wrapper for Opsee Admin permissions check that does null check
func IsOpseeAdmin(user *User) bool {
	if user == nil || !user.IsOpseeAdmin() {
		return false
	}
	return true
}

// Wrapper for Admin permissions check that does null check
func CheckOpseeAdmin(user *User) error {
	if user == nil || !user.IsOpseeAdmin() {
		return errNotOpseeAdmin
	}
	return nil
}

// Wrapper for user.HasPermissions check that does null check
func HasPermissions(user *User, perms ...string) bool {
	if user == nil {
		return false
	}
	return user.HasPermissions(perms...)
}

// Wrapper for user.CheckPermission check that does null check
func CheckPermission(user *User, perm string) error {
	if user == nil {
		return errNilUser
	}
	return user.CheckPermission(perm)
}

// Wrapper for user.HasPermissions check that does null check and returns a permissions error
func CheckPermissions(user *User, perms ...string) []error {
	if user == nil {
		return []error{errNilUser}
	}
	return user.CheckPermissions(perms...)
}
