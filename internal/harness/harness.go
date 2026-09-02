package harness

import (
	"errors"

	"github.com/anonyxhappie/keystone/v2/internal/domain"
)

type Adapter interface {
	Discover() error
	Capabilities() []string
	Start(domain.WorkPacket) (string, error)
	Send(string) error
	Observe() ([]domain.Observation, error)
	Interrupt() error
	Resume(domain.Checkpoint) error
	Result() (domain.Status, error)
}

type HarnessIdentity interface {
	HarnessID() string
}

// SessionIdentity exposes the provider's durable conversation/session id when
// the provider has one. It is intentionally optional so existing adapters do
// not need to implement provider-specific lifecycle details.
type SessionIdentity interface {
	SessionID() string
}

type Versioned interface {
	Version() string
}

// PacketResumer lets a provider resume its own conversation while giving it a
// freshly compiled packet. Adapters without provider-native resume continue
// to use the generic Adapter.Resume contract.
type PacketResumer interface {
	ResumePacket(domain.Checkpoint, domain.WorkPacket) (string, error)
}

type Stopper interface {
	Stop() error
}

type Metadata struct {
	Provider      string   `json:"provider"`
	Version       string   `json:"version,omitempty"`
	Control       []string `json:"control,omitempty"`
	Observability []string `json:"observability,omitempty"`
	Evidence      []string `json:"evidence,omitempty"`
	Limitations   []string `json:"limitations,omitempty"`
}

type MetadataProvider interface {
	Metadata() Metadata
}

var ErrUnsupported = errors.New("harness capability is unsupported")

type Manual struct{}

func (Manual) Discover() error                           { return nil }
func (Manual) Capabilities() []string                    { return []string{"prompt-export", "result-import"} }
func (Manual) HarnessID() string                         { return "manual" }
func (Manual) Start(p domain.WorkPacket) (string, error) { return Render(p), nil }
func (Manual) Send(string) error                         { return nil }
func (Manual) Observe() ([]domain.Observation, error)    { return nil, nil }
func (Manual) Interrupt() error                          { return nil }
func (Manual) Resume(domain.Checkpoint) error            { return nil }
func (Manual) Result() (domain.Status, error)            { return domain.StatusUnknown, nil }
