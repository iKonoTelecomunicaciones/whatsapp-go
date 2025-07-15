package waid

import (
	"fmt"
	"strings"

	"github.com/iKonoTelecomunicaciones/go/bridgev2/networkid"
	"go.mau.fi/whatsmeow/types"
)

const LIDPrefix = "lid-"
const BotPrefix = "bot-"

type ParsedMessageID struct {
	Chat   types.JID
	Sender types.JID
	ID     types.MessageID
}

func MakePortalID(jid types.JID) networkid.PortalID {
	return networkid.PortalID(jid.ToNonAD().String())
}

func ParseUserID(user networkid.UserID) types.JID {
	if strings.HasPrefix(string(user), LIDPrefix) {
		return types.NewJID(strings.TrimPrefix(string(user), LIDPrefix), types.HiddenUserServer)
	} else if strings.HasPrefix(string(user), BotPrefix) {
		return types.NewJID(strings.TrimPrefix(string(user), BotPrefix), types.BotServer)
	}
	return types.NewJID(string(user), types.DefaultUserServer)
}

func MakeUserID(user types.JID) networkid.UserID {
	switch user.Server {
	case types.DefaultUserServer:
		return networkid.UserID(user.User)
	case types.BotServer:
		return networkid.UserID(BotPrefix + user.User)
	case types.HiddenUserServer:
		return networkid.UserID(LIDPrefix + user.User)
	default:
		return ""
	}
}

func ParsePortalID(portal networkid.PortalID) (types.JID, error) {
	parsed, err := types.ParseJID(string(portal))
	if err != nil {
		return types.EmptyJID, fmt.Errorf("invalid portal ID: %w", err)
	}
	return parsed, nil
}

func MakeUserLoginID(user types.JID) networkid.UserLoginID {
	if user.Server != types.DefaultUserServer {
		return ""
	}
	return networkid.UserLoginID(user.User)
}

func ParseUserLoginID(user networkid.UserLoginID, deviceID uint16) types.JID {
	if user == "" {
		return types.EmptyJID
	}
	return types.JID{
		Server: types.DefaultUserServer,
		User:   string(user),
		Device: deviceID,
	}
}

func ParseMessageID(messageID networkid.MessageID) (*ParsedMessageID, error) {
	parts := strings.SplitN(string(messageID), ":", 3)
	if len(parts) == 3 {
		if parts[0] == "fake" || strings.HasPrefix(parts[2], "FAKE::") {
			return nil, fmt.Errorf("fake message ID")
		}
		chat, err := types.ParseJID(parts[0])
		if err != nil {
			return nil, err
		}
		sender, err := types.ParseJID(parts[1])
		if err != nil {
			return nil, err
		}
		return &ParsedMessageID{Chat: chat, Sender: sender, ID: parts[2]}, nil
	} else {
		return nil, fmt.Errorf("invalid message ID")
	}
}

func MakeMessageID(chat, sender types.JID, id types.MessageID) networkid.MessageID {
	return networkid.MessageID(fmt.Sprintf("%s:%s:%s", chat.ToNonAD().String(), sender.ToNonAD().String(), id))
}
