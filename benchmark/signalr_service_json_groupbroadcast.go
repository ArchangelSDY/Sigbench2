package benchmark

import (
	"time"
)

var _ Subject = (*SignalrServiceJsonGroupBroadcast)(nil)

type SignalrServiceJsonGroupBroadcast struct {
	SignalrCoreCommon
}

func (s *SignalrServiceJsonGroupBroadcast) LatencyCheckTarget() string {
	return "SendToGroup"
}

func (s *SignalrServiceJsonGroupBroadcast) IsJson() bool {
	return true
}

func (s *SignalrServiceJsonGroupBroadcast) IsMsgpack() bool {
	return false
}

func (s *SignalrServiceJsonGroupBroadcast) JoinGroupTarget() string {
	return "JoinGroup"
}

func (s *SignalrServiceJsonGroupBroadcast) LeaveGroupTarget() string {
	return "LeaveGroup"
}

func (s *SignalrServiceJsonGroupBroadcast) Name() string {
	return "SignalR Service Group Broadcast"
}

func (s *SignalrServiceJsonGroupBroadcast) DoEnsureConnection(count int, conPerSec int) error {
	return s.doEnsureConnection(count, conPerSec, func(withSessions *WithSessions) (*Session, error) {
		return s.SignalrServiceJsonConnect()
	})
}

func (s *SignalrServiceJsonGroupBroadcast) DoSend(clients int, intervalMillis int) error {
	return nil
}

func (s *SignalrServiceJsonGroupBroadcast) DoGroupSend(clients int, intervalMillis int) error {
	return s.doSend(clients, intervalMillis, &JsonGroupSendMessageGenerator{
		WithInterval: WithInterval{
			interval: time.Millisecond * time.Duration(intervalMillis),
		},
		Target: s.LatencyCheckTarget(),
	})
}

func (s *SignalrServiceJsonGroupBroadcast) DoJoinGroup(membersPerGroup int) error {
	return s.doJoinGroup(membersPerGroup, func(groupName string) Message {
		arguments := []string{groupName, "perf"}
		return GenerateJsonRequest(s.JoinGroupTarget(), arguments)
	})
}

func (s *SignalrServiceJsonGroupBroadcast) DoLeaveGroup() error {
	return s.doLeaveGroup(func(groupName string) Message {
		arguments := []string{groupName, "perf"}
		return GenerateJsonRequest(s.LeaveGroupTarget(), arguments)
	})
}
