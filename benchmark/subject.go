package benchmark

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"aspnet.com/util"
	"github.com/teris-io/shortid"
)

// Config defines the basic configuration for the benchmark.
type Config struct {
	Host     string
	Subject  string
	CmdFile  string
	UseWss   bool
	SendSize int
}

// Subject defines the interface for a test subject.
type Subject interface {
	ProtocolProcessing
	Name() string
	Setup(config *Config, p ProtocolProcessing) error
	Counters() map[string]int64

	DoEnsureConnection(count int, conPerSec int) error
	DoSend(clients int, intervalMillis int) error
	DoGroupSend(clients int, intervalMillis int) error
	DoJoinGroup(membersPerGroup int) error
	DoClear(prefix string) error
}

type WithCounter struct {
	counter *util.Counter
}

func (w *WithCounter) Counter() *util.Counter {
	return w.counter
}

func (w *WithCounter) LogError(errorGroup string, uid string, msg string, err error) {
	log.Printf("[Error][%s] %s due to %s", uid, msg, err)
	if errorGroup != "" {
		w.Counter().Stat(errorGroup, 1)
	}
}

const LatencyStep int64 = 100
const LatencyLength int = 10

func (w *WithCounter) LogLatency(prefix string, latency int64) {
	index := int(latency / LatencyStep)
	if index >= LatencyLength {
		w.Counter().Stat(fmt.Sprintf("%s:ge:%d", prefix, LatencyLength*int(LatencyStep)), 1)
	} else {
		w.Counter().Stat(fmt.Sprintf("%s:lt:%d00", prefix, index+1), 1)
	}
}

func (s *WithCounter) Counters() map[string]int64 {
	return s.Counter().Snapshot()
}

func (s *WithCounter) DoClear(prefix string) error {
	s.counter.Clear(prefix)
	return nil
}

type WithSessions struct {
	host         string
	useWss       bool
	sendSize     int
	sessions     []*Session
	sessionsLock sync.Mutex
	joinGroupWg  sync.WaitGroup

	received chan MessageReceived
}

func (s *WithSessions) doEnsureConnection(count int, conPerSec int, builder func(*WithSessions) (*Session, error)) error {
	if count < 0 {
		return nil
	}

	s.sessionsLock.Lock()
	defer s.sessionsLock.Unlock()

	diff := count - len(s.sessions)
	if diff > 0 {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		var wg sync.WaitGroup
		for _ = range ticker.C {
			nextBatch := diff
			if nextBatch > conPerSec {
				nextBatch = conPerSec
			}
			for i := 0; i < nextBatch; i++ {
				wg.Add(1)
				go func() {
					// randomize the start time of connection
					time.Sleep(time.Millisecond * time.Duration(rand.Int()%1000))
					session, err := builder(s)
					wg.Done()
					if err != nil {
						log.Println("Fail to build connection: ", err)
						return
					}
					s.sessions = append(s.sessions, session)
				}()
			}
			diff -= nextBatch
			if diff <= 0 {
				break
			}
		}
		wg.Wait()
	} else {
		log.Printf("Reduce clients count from %d to %d", len(s.sessions), count)
		extra := s.sessions[count:]
		s.sessions = s.sessions[:count]
		for _, session := range extra {
			session.Close()
		}
	}

	return nil
}

func (s *WithSessions) doSend(clients int, intervalMillis int, gen MessageGenerator) error {
	s.sessionsLock.Lock()
	defer s.sessionsLock.Unlock()

	s.doStopSendUnsafe()

	sessionCount := len(s.sessions)
	bound := sessionCount
	if clients < bound {
		bound = clients
	}

	indices := rand.Perm(sessionCount)
	for i := 0; i < bound; i++ {
		s.sessions[indices[i]].InstallMessageGeneator(gen)
	}

	return nil
}

func (s *WithSessions) doJoinGroup(membersPerGroup int, joinGroup func(string) Message) error {
	s.sessionsLock.Lock()
	defer s.sessionsLock.Unlock()

	s.doStopSendUnsafe()

	sessionCount := len(s.sessions)
	if membersPerGroup > sessionCount {
		membersPerGroup = sessionCount
	}
	indices := rand.Perm(sessionCount)
	bound := sessionCount
	var id string
	for i := 0; i < bound; i++ {
		if i%membersPerGroup == 0 {
			id, _ = shortid.Generate()
		}
		msg := joinGroup(id)
		s.sessions[indices[i]].GroupName = id
		s.sessions[indices[i]].WriteMessage(msg)
	}
	return nil
}

func (s *WithSessions) doLeaveGroup(leaveGroup func(string) Message) error {
	s.sessionsLock.Lock()
	defer s.sessionsLock.Unlock()

	s.doStopSendUnsafe()

	sessionCount := len(s.sessions)
	indices := rand.Perm(sessionCount)
	for i := 0; i < sessionCount; i++ {
		msg := leaveGroup(s.sessions[indices[i]].GroupName)
		s.sessions[indices[i]].WriteMessage(msg)
	}
	return nil
}

func (s *WithSessions) DoStopSend() error {
	s.sessionsLock.Lock()
	defer s.sessionsLock.Unlock()

	return s.doStopSendUnsafe()
}

func (s *WithSessions) doStopSendUnsafe() error {
	for _, session := range s.sessions {
		session.RemoveMessageGenerator()
	}

	return nil
}
