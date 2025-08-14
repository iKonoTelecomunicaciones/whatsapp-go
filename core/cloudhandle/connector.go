package cloudhandle

import (
	"context"
	"sync"
	"time"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/connector/whatsappclouddb"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/types"
	"go.mau.fi/util/exsync"
)

const EditMaxAge = 15 * time.Minute

type WhatsappCloudConnector struct {
	Bridge  *bridgev2.Bridge
	Config  Config
	MsgConv *MessageConverter
	DB      *whatsappclouddb.Database

	firstClientConnectOnce sync.Once
}

// SetMaxFileSize sets the maximum file size for media uploads.
func (whatsappConnector *WhatsappCloudConnector) SetMaxFileSize(maxSize int64) {
	whatsappConnector.MsgConv.MaxFileSize = maxSize
}

// GetName returns the static identifying information for this bridge.
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

// Init initializes the connector with the main bridge instance and sets up
// the message converter and database connection.
func (whatsappConnector *WhatsappCloudConnector) Init(bridge *bridgev2.Bridge) {
	whatsappConnector.Bridge = bridge
	whatsappConnector.MsgConv = NewMessageConverter(bridge)
	whatsappConnector.MsgConv.OldMediaSuffix = "Requesting old media is not enabled on this bridge."

	whatsappConnector.DB = whatsappclouddb.New(
		bridge.ID,
		bridge.DB.Database,
		bridge.Log.With().Str("db_section", "whatsappcloud").Logger(),
	)
	whatsappConnector.MsgConv.DB = whatsappConnector.DB
}

// Start begins the connector's operation, which includes performing database schema upgrades.
func (whatsappConnector *WhatsappCloudConnector) Start(ctx context.Context) error {
	err := whatsappConnector.DB.Upgrade(ctx)
	if err != nil {
		return bridgev2.DBUpgradeError{Err: err, Section: "whatsappcloud"}
	}

	return nil
}

// GetBridgeInfoVersion returns the version of the bridge's information and capabilities.
func (whatsappConnector *WhatsappCloudConnector) GetBridgeInfoVersion() (info, capabilities int) {
	return 1, 1
}

// GetCapabilities returns the general network capabilities of the bridge,
// such as support for disappearing messages.
func (whatsappConnector *WhatsappCloudConnector) GetCapabilities() *bridgev2.NetworkGeneralCapabilities {
	return &bridgev2.NetworkGeneralCapabilities{
		DisappearingMessages: true,
		AggressiveUpdateInfo: true,
	}
}

// GetLoginFlows returns the available login flows that this connector supports.
func (whatsappConnector *WhatsappCloudConnector) GetLoginFlows() []bridgev2.LoginFlow {
	return []bridgev2.LoginFlow{
		{
			Name:        "Whatsapp Cloud Login",
			Description: "Login to WhatsApp Cloud using META's login flow.",
			ID:          "whatsapp-cloud-login",
		},
	}
}

// CreateLogin creates a new login process for a user based on a specific flow ID.
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

// CreateAppLogin creates a new login process for an application,
// using the provided application details.
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

// LoadUserLogin loads an existing user login session and initializes the
// corresponding WhatsApp Cloud client.
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
