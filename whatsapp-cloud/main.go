package main

import (
	"context"
	"net/http"

	"github.com/iKonoTelecomunicaciones/go/bridgev2/bridgeconfig"
	"github.com/iKonoTelecomunicaciones/go/bridgev2/matrix/mxmain"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/connector"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/connector/whatsappclouddb/upgrades"
)

// Information to find out exactly which commit the bridge was built from.
// These are filled at build time with the -X linker flag.
var (
	Tag       = "unknown"
	Commit    = "unknown"
	BuildTime = "unknown"
)

var whatsappConnector = &connector.WhatsappCloudConnector{}
var brmain = mxmain.BridgeMain{
	Name:        "whatsapp-cloud",
	URL:         "https://github.com/iKonoTelecomunicaciones/whatsapp-go",
	Description: "A WhatsApp Cloud puppeting bridge.",
	Version:     "v0.2.14",
	Connector:   whatsappConnector,
}

func main() {
	bridgeconfig.HackyMigrateLegacyNetworkConfig = migrateLegacyConfig
	brmain.PostInit = func() {
		brmain.CheckLegacyDB(
			2,
			"v0.2.13",
			"v0.2.14",
			brmain.LegacyMigrateWithAnotherUpgrader(
				legacyMigrateRenameTables,
				legacyMigrateCopyData,
				22,
				upgrades.Table,
				"whatsapp_version",
				1,
			),
			true,
		)
	}
	brmain.PostStart = func() {
		ctx := brmain.Log.WithContext(context.Background())
		_, err := brmain.DB.Exec(ctx, legacyMigratePostCopyData)

		if err != nil {
			// If the post copy data migration fails, it may be because it was already run.
			// This is not a critical error, so we log it as an error but continue running.
			// This is useful for development and testing, where the migration may have already been run.
			// In production, this should not happen unless the database schema has changed.
			// If the migration fails for another reason, it will be caught by the bridge's
			// error handling and logged appropriately.
			brmain.Log.Err(err).Msg(
				"Error trying to run legacy migration post copy data, perhaps it was already run...",
			)
		}

		if brmain.Matrix.Provisioning != nil {
			brmain.Matrix.Provisioning.Router.HandleFunc(
				"/v1/receive", legacyProvReceive,
			).Methods(http.MethodGet)
			brmain.Matrix.Provisioning.Router.HandleFunc(
				"/v1/receive", legacyProvVerifyConnection,
			).Methods(http.MethodPost)
		}
	}
	brmain.InitVersion(Tag, Commit, BuildTime)
	brmain.Run()
}
