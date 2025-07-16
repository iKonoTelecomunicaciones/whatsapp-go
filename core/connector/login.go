package connector

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/go/bridgev2/database"
	"github.com/iKonoTelecomunicaciones/go/bridgev2/networkid"
	"github.com/iKonoTelecomunicaciones/go/bridgev2/status"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/waid"
	"github.com/rs/zerolog"
	"go.mau.fi/util/exsync"
)

const LoginConnectWait = 15 * time.Second

var (
	_ bridgev2.LoginProcessDisplayAndWait = (*WaCloudLogin)(nil)
	_ bridgev2.LoginProcessUserInput      = (*WaCloudLogin)(nil)
	_ bridgev2.LoginProcessWithOverride   = (*WaCloudLogin)(nil)
)

type WaCloudLogin struct {
	User     *bridgev2.User
	Main     *WhatsappCloudConnector
	Log      zerolog.Logger
	Timezone string

	StartTime     time.Time
	LoginError    error
	LoginComplete *exsync.Event

	Closed         atomic.Bool
	EventHandlerID uint32
}

// TODO: Implement the WaCloudLogin start method to initiate the login process.
func (wl *WaCloudLogin) Start(ctx context.Context) (*bridgev2.LoginStep, error) {
	wl.Log.Info().Msg("Starting ChatBox login process")

	return nil, nil
}

func (wl *WaCloudLogin) Wait(ctx context.Context) (*bridgev2.LoginStep, error) {
	// Here we want to receive the login success event and create a user login from it.
	// But now, we do not connect to WhatsApp yet so we set the newLoginID with the user mxid.
	// Normally, this line should call the chatboxid.MakeUserLoginID(wl.LoginSuccess.ID)
	newLoginID := networkid.UserLoginID(wl.User.MXID)
	ul, err := wl.User.NewLogin(ctx, &database.UserLogin{
		ID:         newLoginID,
		RemoteName: wl.User.MXID.String(),
		RemoteProfile: status.RemoteProfile{
			Name: string(wl.User.BridgeID),
		},
		Metadata: &waid.UserLoginMetadata{
			Timezone: wl.Timezone,

			HistorySyncPortalsNeedCreating: false,
		},
	}, &bridgev2.NewLoginParams{
		DeleteOnConflict: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user login: %w", err)
	}

	ul.Client.Connect(ul.Log.WithContext(context.Background()))

	return &bridgev2.LoginStep{
		Type:         bridgev2.LoginStepTypeComplete,
		Instructions: fmt.Sprintf("Successfully logged in as %s", ul.RemoteName),
		CompleteParams: &bridgev2.LoginCompleteParams{
			UserLoginID: ul.ID,
			UserLogin:   ul,
		},
	}, nil
}

func (wl *WaCloudLogin) Cancel() {
	wl.Closed.Store(true)
}

func (wl *WaCloudLogin) StartWithOverride(
	ctx context.Context, old *bridgev2.UserLogin,
) (*bridgev2.LoginStep, error) {
	step, err := wl.Start(ctx)
	if err == nil && step != nil && old != nil {
		phoneNumber := fmt.Sprintf("+%s", old.ID)
		wl.Log.Debug().
			Str("phone_number", phoneNumber).
			Msg("Auto-submitting phone number for relogin")
		return wl.SubmitUserInput(ctx, map[string]string{
			"phone_number": phoneNumber,
		})
	}
	return step, err
}

func (wl *WaCloudLogin) SubmitUserInput(
	ctx context.Context, input map[string]string,
) (*bridgev2.LoginStep, error) {
	ctx, cancel := context.WithTimeout(ctx, LoginConnectWait)
	defer cancel()

	return nil, nil
}
