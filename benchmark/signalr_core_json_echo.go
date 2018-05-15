package benchmark

import (
	"time"
)

var _ Subject = (*SignalrCoreJsonEcho)(nil)

type SignalrCoreJsonEcho struct {
	SignalrCoreCommon
}

func (s *SignalrCoreJsonEcho) LatencyCheckTarget() string {
	return "echo"
}

func (s *SignalrCoreJsonEcho) IsJson() bool {
	return true
}

func (s *SignalrCoreJsonEcho) IsMsgpack() bool {
	return false
}

func (s *SignalrCoreJsonEcho) Name() string {
	return "SignalR Core Connection"
}

func (s *SignalrCoreJsonEcho) DoEnsureConnection(count int, conPerSec int) error {
	return s.doEnsureConnection(count, conPerSec, func(withSessions *WithSessions) (*Session, error) {
		return s.SignalrCoreJsonConnect()
	})
}

func (s *SignalrCoreJsonEcho) DoGroupSend(clients int, intervalMillis int) error {
	return nil
}

func (s *SignalrCoreJsonEcho) DoSend(clients int, intervalMillis int) error {
	return s.doSend(clients, intervalMillis, &SignalRCoreTextMessageGenerator{
		WithInterval: WithInterval{
			interval: time.Millisecond * time.Duration(intervalMillis),
		},
		Target: s.LatencyCheckTarget(),
	})
}

func (s *SignalrCoreJsonEcho) DoJoinGroup(membersPerGroup int) error {
	return nil
}
