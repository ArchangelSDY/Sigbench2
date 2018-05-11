package benchmark

import (
	"time"
)

var _ Subject = (*SignalrServiceMsgpackEcho)(nil)

type SignalrServiceMsgpackEcho struct {
	SignalrCoreCommon
}

func (s *SignalrServiceMsgpackEcho) LatencyCheckTarget() string {
	return "echo"
}

func (s *SignalrServiceMsgpackEcho) IsJson() bool {
	return false
}

func (s *SignalrServiceMsgpackEcho) IsMsgpack() bool {
	return true
}

func (s *SignalrServiceMsgpackEcho) Name() string {
	return "SignalR Service MsgPack Echo"
}

func (s *SignalrServiceMsgpackEcho) DoJoinGroup(membersPerGroup int) error {
	return nil
}

func (s *SignalrServiceMsgpackEcho) DoEnsureConnection(count int, conPerSec int) error {
	return s.doEnsureConnection(count, conPerSec, func(withSessions *WithSessions) (*Session, error) {
		return s.SignalrServiceMsgPackConnect()
	})
}

func (s *SignalrServiceMsgpackEcho) DoSend(clients int, intervalMillis int) error {
	return s.doSend(clients, intervalMillis, &MessagePackMessageGenerator{
		WithInterval: WithInterval{
			interval: time.Millisecond * time.Duration(intervalMillis),
		},
		Target: s.LatencyCheckTarget(),
	})
}
