package benchmark

import (
)

var _ Subject = (*SignalrCoreJsonBroadcast)(nil)

type SignalrCoreJsonBroadcast struct {
	SignalrCoreJsonEcho
}

func (s *SignalrCoreJsonBroadcast) LatencyCheckTarget() string {
	return "broadcastMessage"
}

func (s *SignalrCoreJsonBroadcast) IsJson() bool {
	return true
}

func (s *SignalrCoreJsonBroadcast) IsMsgpack() bool {
	return false
}

func (s *SignalrCoreJsonBroadcast) Name() string {
	return "SignalR Core Connection"
}

/*
func (s *SignalrCoreJsonBroadcast) DoSend(clients int, intervalMillis int) error {
	return s.doSend(clients, intervalMillis, &SignalRCoreTextMessageGenerator{
		WithInterval: WithInterval{
			interval: time.Millisecond * time.Duration(intervalMillis),
		},
		Target: s.LatencyCheckTarget(),
	})
}
*/
