package connector

import (
	"context"
	"fmt"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/go/bridgev2/database"
	"github.com/iKonoTelecomunicaciones/go/bridgev2/networkid"
	"github.com/iKonoTelecomunicaciones/go/event"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/waid"
	"github.com/rs/zerolog"
	"go.mau.fi/util/ptr"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
)

type WhatsappCloudClient struct {
	Main      *WhatsappCloudConnector
	UserLogin *bridgev2.UserLogin
	Client    *whatsmeow.Client
	Device    *store.Device
	JID       types.JID
}

func (whatsappClient *WhatsappCloudClient) Connect(ctx context.Context) {
	// Method to establish a connection to the whatsappClient.
	// For now, we'll return nil to satisfy the interface.
	return
}

func (whatsappClient *WhatsappCloudClient) Disconnect() {
	// Method to disconnect the whatsappClient.
	// For now, we'll return nil to satisfy the interface.
	return
}

func (whatsappClient *WhatsappCloudClient) GetCapabilities(
	ctx context.Context,
	portal *bridgev2.Portal,
) *event.RoomFeatures {
	// Method to get the capabilities of the whatsappClient.
	return whatsappCloudCaps
}

func (whatsappClient *WhatsappCloudClient) GetChatInfo(
	ctx context.Context, portal *bridgev2.Portal) (*bridgev2.ChatInfo, error,
) {
	portalJID, err := waid.ParsePortalID(portal.ID)
	if err != nil {
		return nil, err
	}
	return whatsappClient.getChatInfo(ctx, portalJID)
}

func (whatsappClient *WhatsappCloudClient) makeEventSender(id types.JID) bridgev2.EventSender {
	if id.Server == types.NewsletterServer {
		// Send as bot
		return bridgev2.EventSender{}
	}
	return bridgev2.EventSender{
		IsFromMe:    waid.MakeUserLoginID(id) == whatsappClient.UserLogin.ID,
		Sender:      waid.MakeUserID(id),
		SenderLogin: waid.MakeUserLoginID(id),
	}
}

func (whatsappClient *WhatsappCloudClient) wrapDMInfo(jid types.JID) *bridgev2.ChatInfo {
	info := &bridgev2.ChatInfo{
		Topic: ptr.Ptr(PrivateChatTopic),
		Members: &bridgev2.ChatMemberList{
			IsFull:           true,
			TotalMemberCount: 2,
			OtherUserID:      waid.MakeUserID(jid),
			MemberMap: map[networkid.UserID]bridgev2.ChatMember{
				waid.MakeUserID(jid): {
					EventSender: whatsappClient.makeEventSender(jid),
				},
				waid.MakeUserID(whatsappClient.JID): {
					EventSender: whatsappClient.makeEventSender(whatsappClient.JID),
				},
			},
			PowerLevels: nil,
		},
		Type: ptr.Ptr(database.RoomTypeDM),
	}
	if jid.Server == types.BotServer {
		info.Topic = ptr.Ptr(BotChatTopic)
	}
	if jid == whatsappClient.JID.ToNonAD() {
		// For chats with self, force-split the members so the user's own ghost is always in the room.
		info.Members.MemberMap = map[networkid.UserID]bridgev2.ChatMember{
			waid.MakeUserID(jid): {EventSender: bridgev2.EventSender{Sender: waid.MakeUserID(jid)}},
			"":                   {EventSender: bridgev2.EventSender{IsFromMe: true}},
		}
	}
	return info
}

func (whatsappClient *WhatsappCloudClient) wrapStatusBroadcastInfo() *bridgev2.ChatInfo {
	userLocal := &bridgev2.UserLocalPortalInfo{}

	return &bridgev2.ChatInfo{
		Name:  ptr.Ptr(StatusBroadcastName),
		Topic: ptr.Ptr(StatusBroadcastTopic),
		Members: &bridgev2.ChatMemberList{
			IsFull: false,
			MemberMap: map[networkid.UserID]bridgev2.ChatMember{
				waid.MakeUserID(whatsappClient.JID): {
					EventSender: whatsappClient.makeEventSender(whatsappClient.JID),
				},
			},
		},
		Type:        ptr.Ptr(database.RoomTypeDefault),
		UserLocal:   userLocal,
		CanBackfill: false,
	}
}

func (whatsappClient *WhatsappCloudClient) getChatInfo(
	ctx context.Context, portalJID types.JID) (wrapped *bridgev2.ChatInfo, err error,
) {
	switch portalJID.Server {
	case types.DefaultUserServer, types.BotServer:
		wrapped = whatsappClient.wrapDMInfo(portalJID)
	case types.BroadcastServer:
		if portalJID == types.StatusBroadcastJID {
			wrapped = whatsappClient.wrapStatusBroadcastInfo()
		} else {
			return nil, fmt.Errorf("broadcast list bridging is currently not supported")
		}
	case types.GroupServer:
		info, err := whatsappClient.Client.GetGroupInfo(portalJID)
		if err != nil {
			return nil, err
		}
		wrapped = whatsappClient.wrapGroupInfo(info)
		wrapped.ExtraUpdates = bridgev2.MergeExtraUpdaters(
			wrapped.ExtraUpdates, updatePortalLastSyncAt,
		)
	case types.NewsletterServer:
		info, err := whatsappClient.Client.GetNewsletterInfo(portalJID)
		if err != nil {
			return nil, err
		}
		wrapped = whatsappClient.wrapNewsletterInfo(info)
	default:
		return nil, fmt.Errorf("unsupported server %s", portalJID.Server)
	}

	return wrapped, nil
}

func (whatsappClient *WhatsappCloudClient) HandleMatrixMessage(
	ctx context.Context,
	msg *bridgev2.MatrixMessage,
) (*bridgev2.MatrixMessageResponse, error) {
	chatboxMsg, req, err := whatsappClient.Main.MsgConv.ToWhatsApp(
		ctx,
		whatsappClient.Client,
		msg.Event,
		msg.Content,
		msg.ReplyTo,
		msg.ThreadRoot,
		msg.Portal,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to convert message: %w", err)
	}
	return whatsappClient.handleConvertedMatrixMessage(ctx, msg, chatboxMsg, req)
}

func (whatsappClient *WhatsappCloudClient) GetStore() *store.Device {
	if cli := whatsappClient.Client; cli != nil {
		if currentStore := cli.Store; currentStore != nil {
			return currentStore
		}
	}
	whatsappClient.UserLogin.Log.Warn().Caller(1).Msg("Returning noop device in GetStore")
	return store.NoopDevice
}

func (whatsappClient *WhatsappCloudClient) IsLoggedIn() bool {
	return whatsappClient.Client != nil && whatsappClient.Client.IsLoggedIn()
}

func (whatsappClient *WhatsappCloudClient) IsThisUser(
	_ context.Context, userID networkid.UserID,
) bool {
	return userID == waid.MakeUserID(whatsappClient.JID)
}

func (whatsappClient *WhatsappCloudClient) LogoutRemote(ctx context.Context) {
	if cli := whatsappClient.Client; cli != nil {
		err := cli.Logout(ctx)

		if err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("Failed to log out")
		}
	}

	whatsappClient.Client = nil
}

func (whatsappClient *WhatsappCloudClient) wrapGroupInfo(info *types.GroupInfo) *bridgev2.ChatInfo {
	// Convert the group info to a ChatInfo structure
	// For now, we'll return nil to satisfy the interface.

	return nil
}

func (whatsappClient *WhatsappCloudClient) wrapNewsletterInfo(
	info *types.NewsletterMetadata,
) *bridgev2.ChatInfo {
	// Convert the group info to a ChatInfo structure
	// For now, we'll return nil to satisfy the interface.

	return nil
}

func (whatsappClient *WhatsappCloudClient) GetUserInfo(
	ctx context.Context, ghost *bridgev2.Ghost,
) (*bridgev2.UserInfo, error) {
	// Method to get user information from the whatsappClient.
	// For now, we'll return an empty UserInfo to satisfy the interface.

	return &bridgev2.UserInfo{}, nil
}
