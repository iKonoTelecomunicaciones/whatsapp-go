package main

import (
	"net/http"

	"github.com/iKonoTelecomunicaciones/go/bridgev2/networkid"
	"github.com/iKonoTelecomunicaciones/go/id"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/connector"

	"go.mau.fi/util/exhttp"
)

type Response struct {
	Detail map[string]interface{} `json:"detail"`
}

type CreateBody struct {
	DisplayName string `json:"display_name"`
	UserName    string `json:"user_name"`
	Password    string `json:"password"`
}

type RespResolveIdentifier struct {
	ID          networkid.UserID    `json:"id"`
	Name        string              `json:"name,omitempty"`
	AvatarURL   id.ContentURIString `json:"avatar_url,omitempty"`
	Identifiers []string            `json:"identifiers,omitempty"`
	MXID        id.UserID           `json:"mxid,omitempty"`
	DMRoomID    id.RoomID           `json:"dm_room_mxid,omitempty"`
}

type ConnInfo struct {
	IsConnected bool `json:"is_connected"`
	IsLoggedIn  bool `json:"is_logged_in"`
}

type ConnectionInfo struct {
	HasSession     bool      `json:"has_session"`
	ManagementRoom id.RoomID `json:"management_room"`
	Conn           ConnInfo  `json:"conn"`
	JID            string    `json:"jid"`
	Phone          string    `json:"phone"`
	Platform       string    `json:"platform"`
}

type PingInfo struct {
	WhatsappConnectionInfo ConnectionInfo `json:"whatsapp"`
	Mxid                   id.UserID      `json:"mxid"`
}

func jsonResponse(w http.ResponseWriter, status int, detail map[string]interface{}) {
	exhttp.WriteJSONResponse(w, status, Response{Detail: detail})
}

func legacyProvReceive(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message": "This endpoint is not implemented yet",
	})
}

func legacyProvVerifyConnection(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message": "This endpoint is not implemented yet",
	})
}

func legacyProvPing(w http.ResponseWriter, r *http.Request) {
	userLogin := brmain.Matrix.Provisioning.GetLoginForRequest(w, r)

	if userLogin == nil {
		return
	}

	whatsappClient := userLogin.Client.(*connector.WhatsappCloudClient)
	managementRoom, err := userLogin.User.GetManagementRoom(r.Context())

	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"message": "Error while fetching management room, failed getting the room",
		})
		return
	}

	whatsappConnectionInfo := ConnectionInfo{
		HasSession:     whatsappClient.IsLoggedIn(),
		ManagementRoom: managementRoom,
	}

	if !whatsappClient.JID.IsEmpty() {
		whatsappConnectionInfo.JID = whatsappClient.JID.String()
		whatsappConnectionInfo.Phone = "+" + whatsappClient.JID.User
		if whatsappClient.Device != nil && whatsappClient.Device.Platform != "" {
			whatsappConnectionInfo.Platform = whatsappClient.Device.Platform
		}
	}

	if whatsappClient.Client != nil {
		whatsappConnectionInfo.Conn = ConnInfo{
			IsConnected: whatsappClient.Client.IsConnected(),
			IsLoggedIn:  whatsappClient.Client.IsLoggedIn(),
		}
	}

	resp := PingInfo{
		WhatsappConnectionInfo: whatsappConnectionInfo,
		Mxid:                   whatsappClient.UserLogin.User.MXID,
	}

	w.Header().Set("Content-Type", "application/json")
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Error while fetching management room, failed getting the room",
	})
	exhttp.WriteJSONResponse(w, http.StatusOK, resp)
}
