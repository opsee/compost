package schema

import opsee_types "github.com/opsee/protobuf/opseeproto/types"

func init() {
	opsee_types.PermissionsBitmap.Register(0, "billing")
	opsee_types.PermissionsBitmap.Register(1, "management")
	opsee_types.PermissionsBitmap.Register(2, "editing")
}
