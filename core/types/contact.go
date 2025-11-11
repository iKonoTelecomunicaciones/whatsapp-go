package types

import "github.com/iKonoTelecomunicaciones/go/bridgev2/networkid"

type ContactInfo struct {
	Found bool

	FirstName    string
	FullName     string
	PushName     string
	BusinessName string
}

type UserKey struct {
	Name string
	MXID string
	ID   networkid.UserID
}
