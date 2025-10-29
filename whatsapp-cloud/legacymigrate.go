package main

import (
	_ "embed"

	"github.com/iKonoTelecomunicaciones/go/bridgev2/bridgeconfig"
	"go.mau.fi/util/configupgrade"
)

const legacyMigrateRenameTables = `
ALTER TABLE portal RENAME TO portal_old;
ALTER TABLE message RENAME TO message_old;
ALTER TABLE matrix_user RENAME TO matrix_user_old;
ALTER TABLE puppet RENAME TO puppet_old;
ALTER TABLE mx_version RENAME TO mx_version_old;
ALTER TABLE reaction RENAME TO reaction_old;
ALTER TABLE mx_user_profile RENAME TO mx_user_profile_old;
ALTER TABLE mx_room_state RENAME TO mx_room_state_old;
`

//go:embed legacymigrate.sql
var legacyMigrateCopyData string

//go:embed legacymigratepost.sql
var legacyMigratePostCopyData string

func migrateLegacyConfig(helper configupgrade.Helper) {
	helper.Set(
		configupgrade.Str,
		"github.com:iKonoTelecomunicaciones/whatsapp-go",
		"encryption",
		"pickle_key",
	)
	bridgeconfig.CopyToOtherLocation(
		helper,
		configupgrade.Str,
		[]string{"whatsapp-cloud", "os_name"},
		[]string{"network", "os_name"},
	)
	bridgeconfig.CopyToOtherLocation(
		helper,
		configupgrade.Str,
		[]string{"chatbox", "user_powerlevel"},
		[]string{"network", "user_powerlevel"},
	)
	bridgeconfig.CopyToOtherLocation(
		helper,
		configupgrade.Bool,
		[]string{"bridge", "disable_status_broadcast_send"},
		[]string{"network", "disable_status_broadcast_send"},
	)
}
