package master

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"net/rpc"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"aspnet.com/agent"
	"aspnet.com/benchmark"
	"aspnet.com/metrics"
)

const (
	AgentRoleClient = "client"
)

type AgentProxy struct {
	Name    string
	Address string
	Client  *rpc.Client
}

func NewAgentProxy(address string) (*AgentProxy, error) {
	parts := strings.Split(address, ":")
	name := parts[0]

	client, err := rpc.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	proxy := &AgentProxy{
		Name:    name,
		Address: address,
		Client:  client,
	}
	return proxy, nil
}

type agentMetrics struct {
	Metrics   metrics.AgentMetrics
	Agent     string
	AgentRole string
}

type SnapshotWriter interface {
	WriteCounters(now time.Time, counters map[string]int64) error
	WriteMetrics(now time.Time, metrics []agentMetrics) error
}

// Controller stands for a master and manages all the agents.
type Controller struct {
	SnapshotWriters  []SnapshotWriter
	Agents           []*AgentProxy
	AgentRoles       map[string]string
	CollectProcesses []string
}

func (c *Controller) clientAgents() []*AgentProxy {
	clients := make([]*AgentProxy, 0, len(c.Agents))
	for _, agent := range c.Agents {
		if c.AgentRoles[agent.Name] == AgentRoleClient {
			clients = append(clients, agent)
		}
	}
	return clients
}

func (c *Controller) RegisterAgent(address string, role string) error {
	proxy, err := NewAgentProxy(address)
	if err != nil {
		return err
	}
	c.Agents = append(c.Agents, proxy)
	c.AgentRoles[proxy.Name] = role
	return nil
}

func (c *Controller) setupAgents(config *benchmark.Config) error {
	var wg sync.WaitGroup
	for _, agent := range c.Agents {
		wg.Add(1)
		go func(agent *AgentProxy) {
			if err := agent.Client.Call("Agent.Setup", config, &struct{}{}); err != nil {
				log.Fatalln(err)
			}
			wg.Done()
		}(agent)
	}

	wg.Wait()
	return nil
}

func (c *Controller) collectCounters() map[string]int64 {
	resultsChan := make(chan map[string]int64, len(c.Agents))
	for _, agent := range c.Agents {
		go func(agent *AgentProxy) {
			result := make(map[string]int64)
			if err := agent.Client.Call("Agent.CollectCounters", &struct{}{}, &result); err != nil {
				log.Println("ERROR: Failed to list counters from agent: ", agent.Address, err)
			}
			resultsChan <- result
		}(agent)
	}
	counters := make(map[string]int64)
	for i := 0; i < len(c.Agents); i++ {
		result := <-resultsChan
		for k, v := range result {
			counters[k] += v
		}
	}
	return counters
}

func (c *Controller) printCounters(counters map[string]int64) {
	table := make([][2]string, 0, len(counters))
	for k, v := range counters {
		table = append(table, [2]string{k, strconv.FormatInt(v, 10)})
	}

	sort.Slice(table, func(i, j int) bool {
		return table[i][0] < table[j][0]
	})

	log.Println("Counters:")
	for _, row := range table {
		log.Println("    ", row[0], ": ", row[1])
	}
}

func (c *Controller) collectMetrics() []agentMetrics {
	results := make([]agentMetrics, 0, len(c.Agents))
	resultsChan := make(chan agentMetrics, len(c.Agents))
	for _, agentProxy := range c.Agents {
		go func(agentProxy *AgentProxy) {
			args := &agent.CollectMetricsArgs{
				CollectProcesses: c.CollectProcesses,
			}
			result := metrics.AgentMetrics{}
			if err := agentProxy.Client.Call("Agent.CollectMetrics", args, &result); err != nil {
				log.Println("ERROR: Failed to list metrics from agent: ", agentProxy.Address, err)
			}
			resultsChan <- agentMetrics{
				Metrics:   result,
				Agent:     agentProxy.Name,
				AgentRole: c.AgentRoles[agentProxy.Name],
			}
		}(agentProxy)
	}
	for i := 0; i < len(c.Agents); i++ {
		results = append(results, <-resultsChan)
	}

	return results
}

func (c *Controller) printMetrics(data []agentMetrics) {
	log.Println("Metrics:")
	for _, row := range data {
		log.Printf("    %s: %+v\n", row.Agent, row.Metrics)
	}
}

func (c *Controller) SplitNumber(total, index int) int {
	agentCount := len(c.clientAgents())
	base := total / agentCount
	if index < total%agentCount {
		base++
	}
	return base
}

var csvHeader string
var counterFields = []string{
	"connection:inprogress",
	"connection:established",
	"connection:closed",
	"connection:error",
	"connection:groupjoin",
	"message:lt:100",
	"message:lt:200",
	"message:lt:300",
	"message:lt:400",
	"message:lt:500",
	"message:lt:600",
	"message:lt:700",
	"message:lt:800",
	"message:lt:900",
	"message:lt:1000",
	"message:ge:1000",
	"message:sent",
	"message:received",
	"message:send_error",
	"message:receive_error",
	"message:decode_error",
}

var globalChannels []chan struct{}

func registerStopChannels(ch chan struct{}) {
	globalChannels = append(globalChannels, ch)
}

func NewController(snapshotWriters []SnapshotWriter) *Controller {
	return &Controller{
		SnapshotWriters: snapshotWriters,
		AgentRoles:      make(map[string]string),
	}
}

func (c *Controller) clearAllTask() {
	err := c.stopSending()
	if err != nil {
		fmt.Printf("Fail to stop sending %s\n", err)
	}
	err = c.closeConnection()
	if err != nil {
		fmt.Printf("Fail to close connections %s\n", err)
	}
	c.closeAllChan()
}

// Handle Ctrl C
func (c *Controller) handleSigterm() {
	fmt.Println("Handle Ctrl C")
	c.clearAllTask()
}

func (c *Controller) closeAllChan() {
	for _, ch := range globalChannels {
		close(ch)
	}
}

func (c *Controller) stopSending() error {
	var values []string
	values = append(values, "s")
	values = append(values, "0")
	fmt.Printf("Stop sending: %s\n", values)
	return c.send(values)
}

func (c *Controller) closeConnection() error {
	var values []string
	values = append(values, "c")
	values = append(values, "0")
	fmt.Printf("Close connections: %s\n", values)
	return c.connect(values)
}

func formatCSVRecord(counters map[string]int64) string {
	values := make([]string, len(counterFields))
	for i, field := range counterFields {
		values[i] = strconv.FormatInt(counters[field], 10)
	}
	return strings.Join(values, ",")
}

func (c *Controller) doInvoke(command string, arguments ...string) error {
	for _, agentProxy := range c.Agents {
		err := agentProxy.Client.Call("Agent.Invoke", &agent.Invocation{
			Command:   command,
			Arguments: arguments,
		}, nil)
		if err != nil {
			fmt.Printf("ERROR[%s] %s(%s): %v\n", agentProxy.Address, command, strings.Join(arguments, ","), err)
		}
	}

	return nil
}

func (c *Controller) watchCounters(config *benchmark.Config) {
	stopWatchCounterChan := make(chan struct{})
	registerStopChannels(stopWatchCounterChan)
	go c.watchCountersInternal(stopWatchCounterChan, func(counters map[string]int64) error {
		for _, writer := range c.SnapshotWriters {
			if err := writer.WriteCounters(time.Now(), counters); err != nil {
				log.Println("Error: fail to write counter snapshot: ", err)
				return err
			}
		}
		return nil
	})
}

func (c *Controller) watchCountersInternal(stopChan chan struct{}, snapshotWriter func(map[string]int64) error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			counters := c.collectCounters()
			snapshotWriter(counters)
			c.printCounters(counters)
		case <-stopChan:
			return
		}
	}
}

func (c *Controller) watchMetrics(config *benchmark.Config) {
	stopWatchMetricsChan := make(chan struct{})
	registerStopChannels(stopWatchMetricsChan)
	go c.watchMetricsInternal(stopWatchMetricsChan, func(data []agentMetrics) error {
		for _, writer := range c.SnapshotWriters {
			if err := writer.WriteMetrics(time.Now(), data); err != nil {
				log.Println("Error: fail to write metrics snapshot: ", err)
				return err
			}
		}
		return nil
	})
}

func (c *Controller) watchMetricsInternal(stopChan chan struct{}, snapshotWriter func([]agentMetrics) error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			data := c.collectMetrics()
			snapshotWriter(data)
		case <-stopChan:
			return
		}
	}
}

func (c *Controller) waitTimeoutOrComplete(parts []string, stop bool) error {
	partsLen := len(parts)
	if partsLen < 2 || partsLen > 3 {
		return fmt.Errorf("SYNTAX: w <wait_time_seconds>")
	}
	timeoutSec, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("ERROR: ", err)
	}
	if timeoutSec < 0 {
		return fmt.Errorf("ERROR: connection number is negative")
	}

	waitChannel := make(chan struct{})
	registerStopChannels(waitChannel)

	ticker := time.NewTicker(time.Second * time.Duration(timeoutSec))
	defer ticker.Stop()

	select {
	case <-ticker.C:
		fmt.Printf("--- Finished after %d sec ---\n", timeoutSec)
	case <-waitChannel:
		log.Println("--- Stopped ---")
	}
	if stop {
		c.clearAllTask()
	}
	return nil
}

func (c *Controller) batchRun(config *benchmark.Config) error {
	cmdFile := config.CmdFile
	file, err := os.Open(cmdFile)

	if err != nil {
		return fmt.Errorf("Fail to open %s\n", cmdFile)
	}
	defer func() {
		cerr := file.Close()
		if cerr != nil {
			fmt.Println("Error occurs when close '%s'\n", cmdFile)
		}
	}()

	re := regexp.MustCompile("\\s+")
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}

		parts := re.Split(text, -1)
		if len(parts) < 1 {
			continue
		}
		switch parts[0] {
		case "r", "result":
			c.printCounters(c.collectCounters())
		case "cm", "ClearMessage":
			c.doInvoke("Clear", "message")
		case "wr", "WatchResult":
			c.watchCounters(config)
		case "wm", "WatchMetrics":
			c.watchMetrics(config)
		case "c", "EnsureConnection":
			err = c.connect(parts)
			if err != nil {
				fmt.Println(err)
				return err
			}
		case "gs", "GroupSend":
			err = c.groupSend(parts)
			if err != nil {
				fmt.Println(err)
				break
			}
		case "s", "Send":
			err = c.send(parts)
			if err != nil {
				fmt.Println(err)
				return err
			}
		case "wc", "WaitAndContinue":
			err = c.waitTimeoutOrComplete(parts, false)
			if err != nil {
				fmt.Println(err)
				return err
			}
		case "w", "Wait":
			err = c.waitTimeoutOrComplete(parts, true)
			if err != nil {
				fmt.Println(err)
				return err
			}
		case "jg", "JoinGroup":
			err = c.joinGroup(parts)
			if err != nil {
				fmt.Println(err)
				return err
			}
		case "lg", "LeaveGroup":
			err = c.leaveGroup()
			if err != nil {
				fmt.Println(err)
				return err
			}
		default:
			fmt.Printf("Illegal command!")
			return fmt.Errorf("Illegal command!")
		}
	}
	return nil
}

func (c *Controller) interactiveRun() error {
	reader := bufio.NewReader(os.Stdin)

	re := regexp.MustCompile("\\s+")
	for {
		fmt.Print("> ")
		text, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		parts := re.Split(text, -1)
		if len(parts) < 1 {
			continue
		}
		switch parts[0] {
		case "r", "result":
			c.printCounters(c.collectCounters())
		case "m", "metrics":
			c.printMetrics(c.collectMetrics())
		case "v":
			c.clearAndWaitAndDump(10)
		case "c", "EnsureConnection":
			err = c.connect(parts)
			if err != nil {
				fmt.Println(err)
				break
			}
		case "gs", "GroupSend":
			err = c.groupSend(parts)
			if err != nil {
				fmt.Println(err)
				break
			}
		case "s", "Send":
			err = c.send(parts)
			if err != nil {
				fmt.Println(err)
				break
			}
		case "jg", "JoinGroup":
			err = c.joinGroup(parts)
			if err != nil {
				fmt.Println(err)
				break
			}
		case "lg", "LeaveGroup":
			err = c.leaveGroup()
			if err != nil {
				fmt.Println(err)
				break
			}
		default:
			for _, agentProxy := range c.Agents {
				err := agentProxy.Client.Call("Agent.Invoke", &agent.Invocation{
					Command:   parts[0],
					Arguments: parts[1:],
				}, nil)
				if err != nil {
					fmt.Printf("ERROR[%s]: %v\n", agentProxy.Address, err)
				}
			}
		}
	}

	return nil
}

func (c *Controller) clearAndWaitAndDump(secWait int) {
	c.doInvoke("Clear", "message")
	time.Sleep(time.Duration(secWait) * time.Second)
	fmt.Println(csvHeader)
	counters := c.collectCounters()
	fmt.Println(formatCSVRecord(counters))
}

func (c *Controller) connect(parts []string) error {
	partsLen := len(parts)
	if partsLen < 2 || partsLen > 3 {
		return fmt.Errorf("SYNTAX: c <connection_count> [connection_per_second]")
	}
	connection, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("ERROR: ", err)
	}
	if connection < 0 {
		return fmt.Errorf("ERROR: connection number is negative")
	}
	connPerSecond := math.MaxInt32
	if len(parts) == 3 {
		connPerSecond, err = strconv.Atoi(parts[2])
		if err != nil {
			return fmt.Errorf("ERROR: ", err)
		}
	}
	if connPerSecond < 0 {
		return fmt.Errorf("ERROR: connection per second is negative")
	}

	var wg sync.WaitGroup
	for i, agentProxy := range c.clientAgents() {
		agentConnection := c.SplitNumber(connection, i)
		agentConnPerSec := c.SplitNumber(connPerSecond, i)
		wg.Add(1)
		go func(aProxy *AgentProxy) {
			err := aProxy.Client.Call("Agent.Invoke", &agent.Invocation{
				Command:   "EnsureConnection",
				Arguments: []string{strconv.Itoa(agentConnection), strconv.Itoa(agentConnPerSec)},
			}, nil)
			if err != nil {
				fmt.Errorf("ERROR[%s]: %v\n", aProxy.Address, err)
			}
			wg.Done()
		}(agentProxy)
	}
	wg.Wait()
	return nil
}

func (c *Controller) joinGroup(parts []string) error {
	partsLen := len(parts)
	if partsLen < 2 {
		return fmt.Errorf("SYNTAX: jg <members_of_group>")
	}
	members, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("ERROR: ", err)
	}
	for _, agentProxy := range c.clientAgents() {
		err := agentProxy.Client.Call("Agent.Invoke", &agent.Invocation{
			Command:   "JoinGroup",
			Arguments: []string{strconv.Itoa(members)},
		}, nil)
		if err != nil {
			return fmt.Errorf("ERROR[%s]: %v\n", agentProxy.Address, err)
		}
	}
	return nil
}

func (c *Controller) leaveGroup() error {
	for _, agentProxy := range c.clientAgents() {
		err := agentProxy.Client.Call("Agent.Invoke", &agent.Invocation{
			Command:   "LeaveGroup",
			Arguments: []string{},
		}, nil)
		if err != nil {
			return fmt.Errorf("ERROR[%s]: %v\n", agentProxy.Address, err)
		}
	}
	return nil
}

func (c *Controller) groupSend(parts []string) error {
	return c.internalSend(parts, "GroupSend")
}

func (c *Controller) send(parts []string) error {
	return c.internalSend(parts, "Send")
}

func (c *Controller) internalSend(parts []string, cmd string) error {
	partsLen := len(parts)
	if partsLen < 2 || partsLen > 3 {
		return fmt.Errorf("SYNTAX: s <clients> [interval_millis]")
	}
	clients, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("ERROR: ", err)
	}
	interval := 1000
	if partsLen >= 3 {
		interval, err = strconv.Atoi(parts[2])
		if err != nil {
			return fmt.Errorf("ERROR: ", err)
		}
	}
	if clients < 0 {
		clients = math.MaxInt32
	}
	for i, agentProxy := range c.clientAgents() {
		agentClients := c.SplitNumber(clients, i)
		err := agentProxy.Client.Call("Agent.Invoke", &agent.Invocation{
			Command:   cmd,
			Arguments: []string{strconv.Itoa(agentClients), strconv.Itoa(interval)},
		}, nil)
		if err != nil {
			return fmt.Errorf("ERROR[%s]: %v\n", agentProxy.Address, err)
		}
	}
	return nil
}

func (c *Controller) Run(config *benchmark.Config) error {
	if err := c.setupAgents(config); err != nil {
		return err
	}

	sig := make(chan os.Signal, 2)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		c.handleSigterm()
		os.Exit(1)
	}()

	if config.CmdFile == "" {
		return c.interactiveRun()
	} else {
		return c.batchRun(config)
	}
	return nil
}

func init() {
	headers := make([]string, len(counterFields))
	for i, field := range counterFields {
		parts := strings.SplitN(field, ":", 2)
		headers[i] = parts[1]
	}
	csvHeader = strings.Join(headers, ",")
}
