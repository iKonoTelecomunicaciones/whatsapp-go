package connector

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/connector/whatsappclouddb"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/msgconv"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/waid"
	"go.mau.fi/util/exsync"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	wbLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

const EditMaxAge = 15 * time.Minute

type WhatsappCloudConnector struct {
	Bridge      *bridgev2.Bridge
	Config      Config
	DeviceStore *sqlstore.Container
	MsgConv     *msgconv.MessageConverter
	DB          *whatsappclouddb.Database

	firstClientConnectOnce sync.Once

	mediaEditCache         MediaEditCache
	mediaEditCacheLock     sync.RWMutex
	stopMediaEditCacheLoop atomic.Pointer[context.CancelFunc]
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
		DefaultPort:          29342,
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
	whatsappConnector.DeviceStore = sqlstore.NewWithDB(
		bridge.DB.RawDB,
		bridge.DB.Dialect.String(),
		wbLog.Zerolog(bridge.Log.With().Str("db_section", "whatsmeow").Logger()),
	)
	whatsappConnector.MsgConv.DB = whatsappConnector.DB
	store.DeviceProps.Os = proto.String(whatsappConnector.Config.OSName)
}

func (whatsappConnector *WhatsappCloudConnector) Start(ctx context.Context) error {
	err := whatsappConnector.DeviceStore.Upgrade(ctx)
	if err != nil {
		return bridgev2.DBUpgradeError{Err: err, Section: "whatsmeow"}
	}
	err = whatsappConnector.DB.Upgrade(ctx)
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
	return []bridgev2.LoginFlow{}
}

func (whatsappConnector *WhatsappCloudConnector) onFirstClientConnect() {
	ctx := context.Background()
	httpClient := &http.Client{}
	ver, err := whatsmeow.GetLatestVersion(ctx, httpClient)
	if err != nil {
		whatsappConnector.Bridge.Log.Err(err).Msg("Failed to get latest WhatsApp web version number")
	} else {
		whatsappConnector.Bridge.Log.Debug().
			Stringer("hardcoded_version", store.GetWAVersion()).
			Stringer("latest_version", *ver).
			Msg("Got latest WhatsApp web version number")
		store.SetWAVersion(*ver)
	}
	meclCtx, cancel := context.WithCancel(context.Background())
	whatsappConnector.stopMediaEditCacheLoop.Store(&cancel)
	go whatsappConnector.mediaEditCacheExpireLoop(meclCtx)
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
	}, nil
}

func (whatsappConnector *WhatsappCloudConnector) LoadUserLogin(
	_ context.Context, login *bridgev2.UserLogin,
) error {
	client := &WhatsappCloudClient{
		Main:      whatsappConnector,
		UserLogin: login,
	}
	login.Client = client

	var err error
	client.JID = waid.ParseUserLoginID(login.ID, 0)
	if err != nil {
		return err
	}

	if client.Device != nil {
		log := client.UserLogin.Log.With().Str("component", "whatsmeow").Logger()
		client.Client = whatsmeow.NewClient(client.Device, wbLog.Zerolog(log))
	} else {
		client.UserLogin.Log.Warn().Stringer(
			"jid",
			client.JID,
		).Msg("No device found for user in whatsmeow store")
	}

	return nil
}
