package connector

import (
	"time"

	"github.com/iKonoTelecomunicaciones/go/event"
	"go.mau.fi/util/ffmpeg"
	"go.mau.fi/util/jsontime"
	"go.mau.fi/util/ptr"
)

const MaxTextLength = 65536

func capID() string {
	base := "fi.mau.whatsapp.capabilities.2025_01_10"
	if ffmpeg.Supported() {
		return base + "+ffmpeg"
	}
	return base
}

var whatsappCloudCaps = &event.RoomFeatures{
	ID: capID(),

	Formatting: map[event.FormattingFeature]event.CapabilitySupportLevel{
		event.FmtBold:          event.CapLevelFullySupported,
		event.FmtItalic:        event.CapLevelFullySupported,
		event.FmtStrikethrough: event.CapLevelFullySupported,
		event.FmtInlineCode:    event.CapLevelFullySupported,
		event.FmtCodeBlock:     event.CapLevelFullySupported,
		event.FmtUserLink:      event.CapLevelFullySupported,
		event.FmtUnorderedList: event.CapLevelFullySupported,
		event.FmtOrderedList:   event.CapLevelFullySupported,
		event.FmtListStart:     event.CapLevelFullySupported,
		event.FmtBlockquote:    event.CapLevelFullySupported,

		event.FmtInlineLink: event.CapLevelPartialSupport,
		event.FmtHeaders:    event.CapLevelPartialSupport,
	},
	// TODO: Implement file features
	//File:                map[event.CapabilityMsgType]*event.FileFeatures{},
	MaxTextLength:       MaxTextLength,
	LocationMessage:     event.CapLevelFullySupported,
	Poll:                event.CapLevelFullySupported,
	Reply:               event.CapLevelFullySupported,
	Edit:                event.CapLevelFullySupported,
	EditMaxCount:        10,
	EditMaxAge:          ptr.Ptr(jsontime.S(EditMaxAge)),
	Delete:              event.CapLevelFullySupported,
	DeleteForMe:         false,
	DeleteMaxAge:        ptr.Ptr(jsontime.S(2 * 24 * time.Hour)),
	Reaction:            event.CapLevelFullySupported,
	ReactionCount:       1,
	ReadReceipts:        true,
	TypingNotifications: true,
}
