package benchmark


var _ Subject = (*SignalrServiceJsonBroadcast)(nil)

type SignalrServiceJsonBroadcast struct {
	SignalrServiceJsonEcho
}

func (s *SignalrServiceJsonBroadcast) LatencyCheckTarget() string {
	return "broadcastMessage"
}

func (s *SignalrServiceJsonBroadcast) IsJson() bool {
	return true
}

func (s *SignalrServiceJsonBroadcast) IsMsgpack() bool {
	return false
}

func (s *SignalrServiceJsonBroadcast) Name() string {
	return "SignalR Service Broadcast"
}
/*
func (s *SignalrServiceJsonBroadcast) DoSend(clients int, intervalMillis int) error {
	return s.doSend(clients, intervalMillis, &SignalRCoreTextMessageGenerator{
		WithInterval: WithInterval{
			interval: time.Millisecond * time.Duration(intervalMillis),
		},
		Target: s.LatencyCheckTarget(),
	})
}
*/
