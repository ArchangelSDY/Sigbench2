package benchmark

import (
	"time"
)

var _ Subject = (*SignalrServiceMsgpackGroupBroadcast)(nil)

type SignalrServiceMsgpackGroupBroadcast struct {
	SignalrCoreCommon
}

func (s *SignalrServiceMsgpackGroupBroadcast) LatencyCheckTarget() string {
	return "SendToGroup"
}

func (s *SignalrServiceMsgpackGroupBroadcast) IsJson() bool {
	return false
}

func (s *SignalrServiceMsgpackGroupBroadcast) IsMsgpack() bool {
	return true
}

func (s *SignalrServiceMsgpackGroupBroadcast) Name() string {
	return "SignalR Service MsgPack Echo"
}

func (s *SignalrServiceMsgpackGroupBroadcast) JoinGroupTarget() string {
	return "JoinGroup"
}

func (s *SignalrServiceMsgpackGroupBroadcast) LeaveGroupTarget() string {
	return "LeaveGroup"
}

func (s *SignalrServiceMsgpackGroupBroadcast) DoJoinGroup(membersPerGroup int) error {
	return s.doJoinGroup(membersPerGroup, func(uid string) Message {
		arguments := []string{uid, "perf"}
		return GenerateMessagePackRequest(s.JoinGroupTarget(), arguments)
	})
}

func (s *SignalrServiceMsgpackGroupBroadcast) DoLeaveGroup() error {
	return s.doLeaveGroup(func(groupName string) Message {
		arguments := []string{groupName, "perf"}
                return GenerateMessagePackRequest(s.JoinGroupTarget(), arguments)
	})
}

func (s *SignalrServiceMsgpackGroupBroadcast) DoEnsureConnection(count int, conPerSec int) error {
	return s.doEnsureConnection(count, conPerSec, func(withSessions *WithSessions) (*Session, error) {
		return s.SignalrServiceMsgPackConnect()
	})
}

func (s *SignalrServiceMsgpackGroupBroadcast) DoSend(clients int, intervalMillis int) error {
	return nil
}

func (s *SignalrServiceMsgpackGroupBroadcast) DoGroupSend(clients int, intervalMillis int) error {
	return s.doSend(clients, intervalMillis, &MessagePackGroupSendMessageGenerator{
		WithInterval: WithInterval{
			interval: time.Millisecond * time.Duration(intervalMillis),
		},
		Target: s.LatencyCheckTarget(),
	})
}
