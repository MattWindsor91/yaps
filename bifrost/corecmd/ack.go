package corecmd

import (
	"github.com/UniversityRadioYork/baps3d/bifrost/msgproto"
)

// File corecmd/ack.go describes parsing and emitting routines for the ACK core request.

const (
	// RsAck is the Bifrost response word ACK.
	RsAck = "ACK"

	// OkDescription is the standard description for an OK response.
	OkDescription = "success"
)

// AckResponse represents the information contained within an ACK response.
type AckResponse struct {
	// Status is the status code of this acknowledgement.
	Status Status

	// Description is the parseable, but human-readable, string describing the acknowledgement.
	Description string
}

// AckOk is the standard OK response.
var AckOk = AckResponse{
	Status:      StatusOk,
	Description: OkDescription,
}

// Message converts an AckResponse into an ACK message with tag tag.
func (a *AckResponse) Message(tag string) *msgproto.Message {
	return msgproto.NewMessage(tag, RsAck).AddArgs(a.Status.String(), a.Description)
}

// ErrorAck converts an error err into an AckResponse.
// If the error is nil, the response is OK.
// Otherwise, its status is that appropriate for the error's blame, and the description is its Error().
func ErrorAck(err error) *AckResponse {
	if err == nil {
		return &AckOk
	}
	return &AckResponse{
		Status:      ErrorStatus(err),
		Description: err.Error(),
	}
}

// ParseAckResponse tries to parse an arbitrary message as an ACK response.
func ParseAckResponse(m *msgproto.Message) (*AckResponse, error) {
	var err error
	if err = CheckWord(RsAck, m); err != nil {
		return nil, err
	}

	var sstr, desc string
	if sstr, desc, err = TwoArgs(m); err != nil {
		return nil, err
	}

	var s Status
	if s, err = ParseStatus(sstr); err != nil {
		return nil, err
	}

	r := AckResponse{Status: s, Description: desc}
	return &r, nil
}
