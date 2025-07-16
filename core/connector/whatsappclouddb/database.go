package whatsappclouddb

import (
	"github.com/iKonoTelecomunicaciones/go/bridgev2/networkid"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/connector/whatsappclouddb/upgrades"
	"github.com/rs/zerolog"
	"go.mau.fi/util/dbutil"
)

type Database struct {
	*dbutil.Database
	CloudRequest *CloudRequestQuery
}

func New(bridgeID networkid.BridgeID, db *dbutil.Database, log zerolog.Logger) *Database {
	db = db.Child("whatsapp_version", upgrades.Table, dbutil.ZeroLogger(log))
	return &Database{
		Database: db,
		CloudRequest: &CloudRequestQuery{
			QueryHelper: dbutil.MakeQueryHelper(db, func(_ *dbutil.QueryHelper[*CloudRequest]) *CloudRequest {
				return &CloudRequest{}
			}),
		},
	}
}
