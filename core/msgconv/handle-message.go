// mautrix-whatsapp - A Matrix-WhatsApp puppeting bridge.
// Copyright (C) 2024 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package msgconv

import (
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/waid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

func MessageIDToInfo(client *whatsmeow.Client, parsedMsgID *waid.ParsedMessageID) *types.MessageInfo {
	isFromMe := parsedMsgID.Sender.User == client.Store.GetLID().User || parsedMsgID.Sender.User == client.Store.GetJID().User
	return &types.MessageInfo{
		MessageSource: types.MessageSource{
			Chat:     parsedMsgID.Chat,
			Sender:   parsedMsgID.Sender,
			IsFromMe: isFromMe,
			IsGroup:  parsedMsgID.Chat.Server == types.GroupServer,
		},
		ID: parsedMsgID.ID,
	}
}
