package cloudhandle

import (
	"time"
)

// MessageID is the internal ID of a WhatsApp message.
type MessageID = string

// MessageServerID is the server ID of a WhatsApp newsletter message.
type MessageServerID = int
type EditAttribute string

type MessageSource struct {
	Chat           string // The chat where the message was sent.
	Sender         string // The user who sent the message.
	IsFromMe       bool   // Whether the message was sent by the current user instead of someone else.
	IsGroup        bool   // Whether the chat is a group chat or broadcast list.
	AddressingMode string
}

type CloudMessageInfo struct {
	MessageSource
	ID        MessageID
	ServerID  MessageServerID
	Sender    string
	Type      string
	PushName  string
	Timestamp time.Time
	Category  string
	Edit      EditAttribute
}
