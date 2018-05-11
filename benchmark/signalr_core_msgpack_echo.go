package benchmark

import(
	"time"
)

var _ Subject = (*SignalrCoreMsgpackEcho)(nil)

type SignalrCoreMsgpackEcho struct {
	SignalrCoreCommon
}

func (s *SignalrCoreMsgpackEcho) LatencyCheckTarget() string {
	return "echo"
}

func (s *SignalrCoreMsgpackEcho) IsJson() bool {
	return false
}

func (s *SignalrCoreMsgpackEcho) IsMsgpack() bool {
	return true
}

func (s *SignalrCoreMsgpackEcho) Name() string {
	return "SignalR Core MessagePack"
}

func (s *SignalrCoreMsgpackEcho) DoJoinGroup(membersPerGroup int) error {
	return nil
}

func (s *SignalrCoreMsgpackEcho) DoEnsureConnection(count int, conPerSec int) error {
	return s.doEnsureConnection(count, conPerSec, func(withSessions *WithSessions) (*Session, error) {
		return s.SignalrCoreMsgPackConnect()
	})
}

func (s *SignalrCoreMsgpackEcho) DoGroupSend(clients int, intervalMillis int) error {
	return nil
}

func (s *SignalrCoreMsgpackEcho) DoSend(clients int, intervalMillis int) error {
	return s.doSend(clients, intervalMillis, &MessagePackMessageGenerator{
		WithInterval: WithInterval{
			interval: time.Millisecond * time.Duration(intervalMillis),
		},
		Target: s.LatencyCheckTarget(),
	})
}
