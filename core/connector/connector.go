package connector

import (
	"context"
	"sync"
	"time"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/connector/whatsappclouddb"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/msgconv"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/types"
	"go.mau.fi/util/exsync"
)

const EditMaxAge = 15 * time.Minute

type WhatsappCloudConnector struct {
	Bridge  *bridgev2.Bridge
	Config  Config
	MsgConv *msgconv.MessageConverter
	DB      *whatsappclouddb.Database

	firstClientConnectOnce sync.Once
}

func (whatsappConnector *WhatsappCloudConnector) SetMaxFileSize(maxSize int64) {
	whatsappConnector.MsgConv.MaxFileSize = maxSize
}

func (whatsappConnector *WhatsappCloudConnector) GetName() bridgev2.BridgeName {
	return bridgev2.BridgeName{
		DisplayName:          "WhatsApp Cloud",
		NetworkURL:           "https://whatsappcloud.com",
		NetworkIcon:          "mxc://maunium.net/NeXNQarUbrlYBiPCpprYsRqr",
		NetworkID:            "whatsapp-cloud",
		BeeperBridgeType:     "whatsapp-cloud",
		DefaultPort:          29340,
		DefaultCommandPrefix: "!wb",
	}
}

func (whatsappConnector *WhatsappCloudConnector) Init(bridge *bridgev2.Bridge) {
	whatsappConnector.Bridge = bridge
	whatsappConnector.MsgConv = msgconv.New(bridge)
	whatsappConnector.MsgConv.OldMediaSuffix = "Requesting old media is not enabled on this bridge."

	whatsappConnector.DB = whatsappclouddb.New(
		bridge.ID,
		bridge.DB.Database,
		bridge.Log.With().Str("db_section", "whatsappcloud").Logger(),
	)
	whatsappConnector.MsgConv = msgconv.New(bridge)
	whatsappConnector.MsgConv.DB = whatsappConnector.DB
}

func (whatsappConnector *WhatsappCloudConnector) Start(ctx context.Context) error {
	err := whatsappConnector.DB.Upgrade(ctx)
	if err != nil {
		return bridgev2.DBUpgradeError{Err: err, Section: "whatsappcloud"}
	}

	return nil
}

func (whatsappConnector *WhatsappCloudConnector) GetBridgeInfoVersion() (info, capabilities int) {
	return 1, 1
}

func (whatsappConnector *WhatsappCloudConnector) GetCapabilities() *bridgev2.NetworkGeneralCapabilities {
	return &bridgev2.NetworkGeneralCapabilities{
		DisappearingMessages: true,
		AggressiveUpdateInfo: true,
	}
}

// GetLoginFlows implements the required method for the NetworkConnector interface.
func (whatsappConnector *WhatsappCloudConnector) GetLoginFlows() []bridgev2.LoginFlow {
	return []bridgev2.LoginFlow{
		{
			Name:        "Whatsapp Cloud Login",
			Description: "Login to WhatsApp Cloud using META's login flow.",
			ID:          "whatsapp-cloud-login",
		},
	}
}

func (whatsappConnector *WhatsappCloudConnector) CreateLogin(
	_ context.Context,
	user *bridgev2.User,
	flowID string,
) (bridgev2.LoginProcess, error) {
	return &WaCloudLogin{
		User: user,
		Main: whatsappConnector,
		Log: user.Log.With().
			Str("action", "login").
			Logger(),

		LoginComplete: exsync.NewEvent(),
		Received515:   exsync.NewEvent(),
	}, nil
}

func (whatsappConnector *WhatsappCloudConnector) CreateAppLogin(
	_ context.Context,
	user *bridgev2.User,
	body types.CloudRegisterAppRequest,
) (bridgev2.LoginProcess, error) {
	return &WaCloudLogin{
		User: user,
		Main: whatsappConnector,
		Log: user.Log.With().
			Str("action", "login").
			Logger(),

		LoginComplete:   exsync.NewEvent(),
		Received515:     exsync.NewEvent(),
		WabaID:          body.WabaID,
		BusinessPhoneID: body.AppPhoneID,
		PageAccessToken: body.AccessToken,
		AppName:         body.AppName,
	}, nil
}

func (whatsappConnector *WhatsappCloudConnector) LoadUserLogin(
	_ context.Context, login *bridgev2.UserLogin,
) error {
	wClient := &WhatsappCloudClient{
		Main:      whatsappConnector,
		UserLogin: login,
	}

	log := wClient.UserLogin.Log.With().Str("component", "WhatsAppCloudClient").Logger()
	log.Info().Msg("Loading WhatsApp Cloud client for user")

	login.Client = wClient

	return nil
}
