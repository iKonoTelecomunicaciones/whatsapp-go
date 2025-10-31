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

package waid

import (
	"encoding/json"

	"go.mau.fi/util/jsontime"
)

type UserLoginMetadata struct {
	WabaID          string `json:"waba_id"`
	BusinessPhoneID string `json:"business_phone_id"`
	PageAccessToken string `json:"page_access_token"`
}

type PushKeys struct {
	P256DH  []byte `json:"p256dh"`
	Auth    []byte `json:"auth"`
	Private []byte `json:"private"`
}

type MessageErrorType string

const (
	MsgNoError             MessageErrorType = ""
	MsgErrDecryptionFailed MessageErrorType = "decryption_failed"
	MsgErrMediaNotFound    MessageErrorType = "media_not_found"
)

type GroupInviteMeta struct {
	Code       string `json:"code"`
	Expiration int64  `json:"expiration,string"`
}

type MessageMetadata struct {
	SenderDeviceID  uint16           `json:"sender_device_id,omitempty"`
	Error           MessageErrorType `json:"error,omitempty"`
	GroupInvite     *GroupInviteMeta `json:"group_invite,omitempty"`
	FailedMediaMeta json.RawMessage  `json:"media_meta,omitempty"`
	DirectMediaMeta json.RawMessage  `json:"direct_media_meta,omitempty"`
	IsMatrixPoll    bool             `json:"is_matrix_poll,omitempty"`
}

func (mm *MessageMetadata) CopyFrom(other any) {
	otherMM := other.(*MessageMetadata)
	mm.SenderDeviceID = otherMM.SenderDeviceID
	mm.Error = otherMM.Error
	if otherMM.FailedMediaMeta != nil {
		mm.FailedMediaMeta = otherMM.FailedMediaMeta
	}
	if otherMM.DirectMediaMeta != nil {
		mm.DirectMediaMeta = otherMM.DirectMediaMeta
	}
	if otherMM.GroupInvite != nil {
		mm.GroupInvite = otherMM.GroupInvite
	}
	mm.IsMatrixPoll = mm.IsMatrixPoll || otherMM.IsMatrixPoll
}

type ReactionMetadata struct {
	SenderDeviceID uint16 `json:"sender_device_id,omitempty"`
}

type PortalMetadata struct {
	DisappearingTimerSetAt     int64         `json:"disappearing_timer_set_at,omitempty"`
	LastSync                   jsontime.Unix `json:"last_sync,omitempty"`
	CommunityAnnouncementGroup bool          `json:"is_cag,omitempty"`
}

type GhostMetadata struct {
	LastSync jsontime.Unix `json:"last_sync,omitempty"`
}
