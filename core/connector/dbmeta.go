package connector

import (
	"github.com/iKonoTelecomunicaciones/go/bridgev2/database"

	"go.mau.fi/mautrix-whatsapp/pkg/waid"
)

func (wa *WhatsappCloudConnector) GetDBMetaTypes() database.MetaTypes {
	return database.MetaTypes{
		Ghost: func() any {
			return &waid.GhostMetadata{}
		},
		Message: func() any {
			return &waid.MessageMetadata{}
		},
		Reaction: func() any {
			return &waid.ReactionMetadata{}
		},
		Portal: func() any {
			return &waid.PortalMetadata{}
		},
		UserLogin: func() any {
			return &waid.UserLoginMetadata{}
		},
	}
}
