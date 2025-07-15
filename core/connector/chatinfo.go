package connector

import (
	"context"
	"time"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"go.mau.fi/mautrix-whatsapp/pkg/waid"
	"go.mau.fi/util/jsontime"
)

const (
	StatusBroadcastTopic = "WhatsApp status updates from your contacts"
	PrivateChatTopic     = "WhatsApp private chat"
	BotChatTopic         = "WhatsApp chat with a bot"
	StatusBroadcastName  = "WhatsApp Status Broadcast"
	nobodyPL             = 99
	superAdminPL         = 75
	adminPL              = 50
	defaultPL            = 0
)

func updatePortalLastSyncAt(_ context.Context, portal *bridgev2.Portal) bool {
	meta := portal.Metadata.(*waid.PortalMetadata)
	forceSave := time.Since(meta.LastSync.Time) > 24*time.Hour
	meta.LastSync = jsontime.UnixNow()
	return forceSave
}
