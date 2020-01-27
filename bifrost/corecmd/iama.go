package corecmd

import "github.com/UniversityRadioYork/baps3d/bifrost/msgproto"

// File corecmd/iama.go describes parsing and emitting routines for the IAMA core request.

const (
	// RsIama is the Bifrost response word IAMA.
	RsIama = "IAMA"
)

// IamaResponse announces a server or controller's Bifrost role.
type IamaResponse struct {
	// Role contains the announced role.
	Role string
}

// Message converts an AckResponse into an IAMA message with tag tag.
func (a *IamaResponse) Message(tag string) *msgproto.Message {
	return msgproto.NewMessage(tag, RsIama).AddArgs(a.Role)
}

// ParseIamaResponse tries to parse an arbitrary message as an IAMA response.
func ParseIamaResponse(m *msgproto.Message) (*IamaResponse, error) {
	var err error
	if err = CheckWord(RsIama, m); err != nil {
		return nil, err
	}

	var role string
	if role, err = OneArg(m); err != nil {
		return nil, err
	}

	r := IamaResponse{role}
	return &r, nil
}
