package harness

import "github.com/anonyxhappie/keystone/internal/domain"

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

type Manual struct{}
func (Manual) Discover() error { return nil }
func (Manual) Capabilities() []string { return []string{"prompt-export", "result-import"} }
func (Manual) Start(p domain.WorkPacket) (string,error) { return Render(p),nil }
func (Manual) Send(string) error { return nil }
func (Manual) Observe() ([]domain.Observation,error) { return nil,nil }
func (Manual) Interrupt() error { return nil }
func (Manual) Resume(domain.Checkpoint) error { return nil }
func (Manual) Result() (domain.Status,error) { return domain.StatusUnknown,nil }
