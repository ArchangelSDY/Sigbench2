package benchmark


var _ Subject = (*SignalrServiceMsgpackBroadcast)(nil)

type SignalrServiceMsgpackBroadcast struct {
	SignalrServiceMsgpackEcho
}

func (s *SignalrServiceMsgpackBroadcast) LatencyCheckTarget() string {
	return "broadcastMessage"
}

func (s *SignalrServiceMsgpackBroadcast) IsJson() bool {
	return false
}

func (s *SignalrServiceMsgpackBroadcast) IsMsgpack() bool {
	return true
}

func (s *SignalrServiceMsgpackBroadcast) Name() string {
	return "SignalR Service MsgPack Broadcast"
}

/*
func (s *SignalrServiceMsgpackBroadcast) DoSend(clients int, intervalMillis int) error {
	return s.doSend(clients, intervalMillis, &MessagePackMessageGenerator{
		WithInterval: WithInterval{
			interval: time.Millisecond * time.Duration(intervalMillis),
		},
		Target: s.LatencyCheckTarget(),
	})
}
*/
