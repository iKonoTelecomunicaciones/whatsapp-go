package waid

import (
	"fmt"
	"strings"

	"github.com/iKonoTelecomunicaciones/go/bridgev2/matrix/mxmain"
	"github.com/iKonoTelecomunicaciones/go/bridgev2/networkid"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/types"
)

const LIDPrefix = "lid-"
const BotPrefix = "bot-"

type ParsedMessageID struct {
	Chat   string
	Sender string
	ID     string
}

func MakePortalID(id string) networkid.PortalID {
	return networkid.PortalID(id)
}

func MakePortalKey(id string) networkid.PortalKey {
	return networkid.PortalKey{
		ID:       networkid.PortalID(id + "@s.whatsapp.net"),
		Receiver: networkid.UserLoginID(id),
	}
}

func ParseUserID(user networkid.UserID) string {
	if strings.HasPrefix(string(user), LIDPrefix) {
		return strings.TrimPrefix(string(user), LIDPrefix)
	} else if strings.HasPrefix(string(user), BotPrefix) {
		return strings.TrimPrefix(string(user), BotPrefix)
	}
	return string(user)
}

func MakeUserID(user string) networkid.UserID {
	return networkid.UserID(user)
}

func MakeUserKeyUsingBody(body types.CloudEvent, domain string, brmain mxmain.BridgeMain) types.UserKey {
	name := body.Entry[0].Changes[0].Value.Contacts[0].Profile.Name
	userID := body.Entry[0].Changes[0].Value.Contacts[0].WaID
	userName := brmain.Config.AppService.FormatUsername(userID)
	userKey := MakeUserKey(name, userName, userID, domain)

	return userKey
}

func MakeUserKey(name string, userName string, userID string, domain string) types.UserKey {
	userMXID := fmt.Sprintf("@%s:%s", userName, domain)
	return types.UserKey{
		Name: name,
		ID:   networkid.UserID(userID),
		MXID: userMXID,
	}
}

func ParsePortalID(portal networkid.PortalID) (string, error) {
	return string(portal), nil
}

func MakeUserLoginID(user string) networkid.UserLoginID {
	return networkid.UserLoginID(user)
}

func ParseUserLoginID(user networkid.UserLoginID, deviceID uint16) string {

	return string(user)
}

func ParseMessageID(messageID networkid.MessageID) (*ParsedMessageID, error) {
	parts := strings.SplitN(string(messageID), ":", 3)
	if len(parts) == 3 {
		if parts[0] == "fake" || strings.HasPrefix(parts[2], "FAKE::") {
			return nil, fmt.Errorf("fake message ID")
		}
		chat := parts[0]
		sender := parts[1]
		return &ParsedMessageID{Chat: chat, Sender: sender, ID: parts[2]}, nil
	}
	return nil, fmt.Errorf("invalid message ID format: %s", messageID)
}

func MakeMessageID(chat, sender string, id string) networkid.MessageID {
	return networkid.MessageID(fmt.Sprintf("%s:%s:%s", chat, sender, id))
}
