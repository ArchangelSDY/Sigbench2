package benchmark

var _ Subject = (*Dummy)(nil)

type Dummy struct {
}

func (s *Dummy) Setup(config *Config, p ProtocolProcessing) error {
	return nil
}

func (s *Dummy) Counters() map[string]int64 {
	return map[string]int64{
		"counter1": 100,
		"counter2": 50,
	}
}

func (s *Dummy) LatencyCheckTarget() string {
	return "echo"
}

func (s *Dummy) JoinGroupTarget() string {
	return ""
}

func (s *Dummy) LeaveGroupTarget() string {
	return ""
}

func (s *Dummy) IsJson() bool {
	return true
}

func (s *Dummy) IsMsgpack() bool {
	return false
}

func (s *Dummy) Name() string {
	return "Dummy"
}

func (s *Dummy) DoJoinGroup(membersPerGroup int) error {
	return nil
}

func (s *Dummy) DoClear(prefix string) error {
	return nil
}

func (s *Dummy) DoEnsureConnection(count int, conPerSec int) error {
	return nil
}

func (s *Dummy) DoDisconnect(count int) error {
	return nil
}

func (s *Dummy) DoGroupSend(clients int, intervalMillis int) error {
	return nil
}

func (s *Dummy) DoSend(clients int, intervalMillis int) error {
	return nil
}
