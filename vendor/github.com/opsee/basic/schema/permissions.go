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

	opsee_types.PermissionsRegistry.Register("org", &opsee_types.PermissionsBitmap{
		map[int]string{
			0: "multi-user",
			1: "multibastion",
			2: "on-site support",
		},
		sync.RWMutex{},
	})
}
