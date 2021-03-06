package benchmark

import (
	"time"
)

var _ Subject = (*SignalrServiceJsonEcho)(nil)

type SignalrServiceJsonEcho struct {
	SignalrCoreCommon
}

func (s *SignalrServiceJsonEcho) LatencyCheckTarget() string {
	return "echo"
}

func (s *SignalrServiceJsonEcho) IsJson() bool {
	return true
}

func (s *SignalrServiceJsonEcho) IsMsgpack() bool {
	return false
}

func (s *SignalrServiceJsonEcho) Name() string {
	return "SignalR Service Echo"
}

func (s *SignalrServiceJsonEcho) DoJoinGroup (membersPerGroup int) error {
	return nil
}

func (s *SignalrServiceJsonEcho) DoEnsureConnection(count int, conPerSec int) error {
	return s.doEnsureConnection(count, conPerSec, func(withSessions *WithSessions) (*Session, error) {
		return s.SignalrServiceJsonConnect()
	})
}

func (s *SignalrServiceJsonEcho) DoGroupSend(clients int, intervalMillis int) error {
	return nil
}

func (s *SignalrServiceJsonEcho) DoSend(clients int, intervalMillis int) error {
	return s.doSend(clients, intervalMillis, &SignalRCoreTextMessageGenerator{
		WithInterval: WithInterval{
			interval: time.Millisecond * time.Duration(intervalMillis),
		},
		Target: s.LatencyCheckTarget(),
	})
}
