package benchmark


var _ Subject = (*SignalrCoreMsgpackBroadcast)(nil)

type SignalrCoreMsgpackBroadcast struct {
	SignalrCoreMsgpackEcho
}

func (s *SignalrCoreMsgpackBroadcast) LatencyCheckTarget() string {
	return "broadcastMessage"
}

func (s *SignalrCoreMsgpackBroadcast) IsJson() bool {
	return false
}

func (s *SignalrCoreMsgpackBroadcast) IsMsgpack() bool {
	return true
}

func (s *SignalrCoreMsgpackBroadcast) Name() string {
	return "SignalR Core MessagePack"
}

