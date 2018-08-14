package util

import (
	"log"
	"strings"
	"time"
)

type countRecord struct {
	name  string
	value int64
}

type controlRecord struct {
	cmd    string
	params string
}

// Counter is a thread safe multiple producer single consumer counter.
// It is thread safe only when there is one consumer draining the results.
type Counter struct {
	stats          map[string]int64
	recordChannel  chan countRecord
	controlChannel chan controlRecord
	resultChannel  chan map[string]int64
	stopped        bool
}

func NewCounter() *Counter {
	counter := new(Counter)
	counter.stats = make(map[string]int64)
	counter.recordChannel = make(chan countRecord)
	counter.controlChannel = make(chan controlRecord)
	counter.resultChannel = make(chan map[string]int64)

	counter.start()
	return counter
}

// Stat adds a new count record to the counter.
// This is the only method that can be called by the producers from multiple threads / routines.
func (c *Counter) Stat(name string, value int64) {
	if !c.stopped {
		c.recordChannel <- countRecord{name, value}
	}
}

// Snapshot taks a new snapshot of the current counter result.
func (c *Counter) Snapshot() map[string]int64 {
	if !c.stopped {
		c.controlChannel <- controlRecord{"snapshot", ""}
		return <-c.resultChannel
	}
	return nil
}

// Clear reset all counts that start with prefix.
func (c *Counter) Clear(prefix string) {
	if !c.stopped {
		c.controlChannel <- controlRecord{"clear", prefix}
	}
}

// Stop closes the counter and it will not accumulate future records.
func (c *Counter) Stop() {
	c.controlChannel <- controlRecord{"stop", ""}
}

func (c *Counter) start() {
	go func() {
		defer close(c.recordChannel)
		defer close(c.controlChannel)
		defer close(c.resultChannel)

		for {
			select {
			case record := <-c.controlChannel:
				switch record.cmd {
				case "clear":
					original := c.stats
					c.stats = make(map[string]int64)
					if record.params != "" {
						for k, v := range original {
							if !strings.HasPrefix(k, record.params) {
								c.stats[k] = v
							}
						}
					}
				case "stop":
					return
				case "snapshot":
					snapshot := make(map[string]int64)
					for k, v := range c.stats {
						snapshot[k] = v
					}
					c.resultChannel <- snapshot
				default:
					log.Println("Unknown control command: ", record.cmd)
				}
				continue
			default:
				// No control command, move forward to check record channel
			}

			select {
			case record := <-c.recordChannel:
				c.stats[record.name] += record.value
			case <-time.After(100 * time.Millisecond):
				// Wait at most 100ms for data record,
				// otherwise go to next loop to check control record
				continue
			}
		}
	}()
}
