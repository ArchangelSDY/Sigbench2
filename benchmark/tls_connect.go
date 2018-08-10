package benchmark

import (
	"crypto/tls"
	"log"
	"math/rand"
	"sync"
	"time"

	"aspnet.com/util"
)

var _ Subject = (*TlsConnect)(nil)

type TlsConnect struct {
	WithCounter
	WithSessions
	host string
}

func (s *TlsConnect) Setup(config *Config, p ProtocolProcessing) error {
	s.host = config.Host
	s.counter = util.NewCounter()
	return nil
}

func (s *TlsConnect) LatencyCheckTarget() string {
	return "echo"
}

func (s *TlsConnect) JoinGroupTarget() string {
	return ""
}

func (s *TlsConnect) LeaveGroupTarget() string {
	return ""
}

func (s *TlsConnect) IsJson() bool {
	return false
}

func (s *TlsConnect) IsMsgpack() bool {
	return false
}

func (s *TlsConnect) Name() string {
	return "TlsConnect"
}

func (s *TlsConnect) DoJoinGroup(membersPerGroup int) error {
	return nil
}

func (s *TlsConnect) DoClear(prefix string) error {
	return nil
}

func (s *TlsConnect) DoEnsureConnection(count int, conPerSec int) error {
	if count < 0 {
		return nil
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var wg sync.WaitGroup
	for _ = range ticker.C {
		nextBatch := count
		if nextBatch > conPerSec {
			nextBatch = conPerSec
		}
		for i := 0; i < nextBatch; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// randomize the start time of connection
				time.Sleep(time.Millisecond * time.Duration(rand.Int()%1000))

				s.Counter().Stat("tls:inprogress", 1)
				t := time.Now()

				_, err := tls.Dial("tcp", s.host, &tls.Config{})
				if err != nil {
					s.Counter().Stat("tls:inprogress", -1)
					s.Counter().Stat("tls:error", 1)
					log.Println("Fail to build connection: ", err)
					return
				}
				s.Counter().Stat("tls:inprogress", -1)
				s.Counter().Stat("tls:connected", 1)
				s.LogLatency("tls:dial", int64(time.Now().Sub(t)/time.Millisecond))
			}()
		}
		count -= nextBatch
		if count <= 0 {
			break
		}
	}
	wg.Wait()

	return nil
}

func (s *TlsConnect) DoGroupSend(clients int, intervalMillis int) error {
	return nil
}

func (s *TlsConnect) DoSend(clients int, intervalMillis int) error {
	return nil
}
