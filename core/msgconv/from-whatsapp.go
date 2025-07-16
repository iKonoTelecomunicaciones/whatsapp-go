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
	"context"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/go/event"
	_ "golang.org/x/image/webp"
)

type contextKey int

const (
	contextKeyClient contextKey = iota
	contextKeyIntent
	contextKeyPortal
)

func getPortal(ctx context.Context) *bridgev2.Portal {
	return ctx.Value(contextKeyPortal).(*bridgev2.Portal)
}

var failedCommentPart = &bridgev2.ConvertedMessagePart{
	Type: event.EventMessage,
	Content: &event.MessageEventContent{
		Body:    "Failed to decrypt comment",
		MsgType: event.MsgNotice,
	},
}
