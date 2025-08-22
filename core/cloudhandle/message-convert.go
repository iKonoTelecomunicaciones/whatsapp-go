package cloudhandle

import (
	"context"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/go/format"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/connector/whatsappclouddb"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/waid"
)

type AnimatedStickerConfig struct {
	Target string `yaml:"target"`
	Args   struct {
		Width  int `yaml:"width"`
		Height int `yaml:"height"`
		FPS    int `yaml:"fps"`
	} `yaml:"args"`
}

type MessageConverter struct {
	Bridge                *bridgev2.Bridge
	DB                    *whatsappclouddb.Database
	MaxFileSize           int64
	HTMLParser            *format.HTMLParser
	AnimatedStickerConfig AnimatedStickerConfig
	FetchURLPreviews      bool
	ExtEvPolls            bool
	DisableViewOnce       bool
	DirectMedia           bool
	OldMediaSuffix        string
}

type WhatsappCloudClient struct {
	Main      *WhatsappCloudConnector
	UserLogin *bridgev2.UserLogin
}

func (whatsappClient *WhatsappCloudClient) GetMetaData(
	ctx context.Context,
) *waid.UserLoginMetadata {
	return whatsappClient.UserLogin.Metadata.(*waid.UserLoginMetadata)
}
