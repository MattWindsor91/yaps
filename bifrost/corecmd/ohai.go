package corecmd

// File corecmd/ohai.go describes parsing and emitting routines for the OHAI core request.

import (
	"fmt"
	"github.com/UniversityRadioYork/baps3d/bifrost/msgproto"
)

const (
	// RsOhai is the Bifrost response word OHAI.
	RsOhai = "OHAI"

	// ThisProtocolVer represents the Bifrost protocol version this library represents.
	ThisProtocolVer = "bifrost-0.0.0"
)

// OhaiResponse represents the information contained within an OHAI response.
type OhaiResponse struct {
	// ProtocolVer is the semantic-version identifier for the Bifrost protocol.
	ProtocolVer string
	// ProtocolVer is the semantic-version identifier for the server itself.
	ServerVer   string
}

// Message converts an Ohai into an OHAI message with tag tag.
func (o *OhaiResponse) Message(tag string) *msgproto.Message {
	return msgproto.NewMessage(tag, RsOhai).AddArg(o.ProtocolVer).AddArg(o.ServerVer)
}

// ParseOhaiResponse tries to parse an arbitrary message as an OHAI response.
func ParseOhaiResponse(m msgproto.Message) (resp *OhaiResponse, err error) {
	if m.Word() != RsOhai {
		return nil, fmt.Errorf("expected word %s, got %s", RsOhai, m.Word())
	}

	args := m.Args()
	if len(args) != 2 {
		return nil, fmt.Errorf("bad arity: expected 2, got %d", len(args))
	}

	r := OhaiResponse{
		ProtocolVer: args[0],
		ServerVer:   args[1],
	}
	return &r, nil
}
