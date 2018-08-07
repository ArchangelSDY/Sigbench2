package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"strings"
	"time"

	"aspnet.com/agent"
	"aspnet.com/benchmark"
	"aspnet.com/forwarder"
	"aspnet.com/master"

	flags "github.com/jessevdk/go-flags"
)

var opts struct {
	Mode             string `short:"m" long:"mode" description:"Run mode" default:"agent" choice:"agent" choice:"master" choice:"forwarder"`
	OutputDir        string `short:"o" long:"output-dir" description:"Output directory" default:"output"`
	ListenAddress    string `short:"l" long:"listen-address" description:"Listen address" default:":7000"`
	Agents           string `short:"a" long:"agents" description:"Agent addresses separated by comma"`
	Role             string `long:"role" description:"Agent role"`
	Collectors       string `long:"collectors" description:"Collector agent addresses separated by comma"`
	CollectProcesses string `long:"collect-processes" description:"Process names to collect metrics data"`
	Server           string `short:"s" long:"server" description:"Websocket server host:port"`
	Subject          string `short:"t" long:"test-subject" description:"Test subject"`
	CmdFile          string `short:"c" long:"cmd-file" description:"Command file"`
	UseWss           bool   `short:"u" long:"use-security-connection" description:"wss connection"`
	SendSize         int    `short:"b" long:"send-size" description:"send message size (byte), default is 0, 0 means: a shortID + timestamp" default:"0"`
	ReverseAgent     bool   `short:"r" long:"reverse" description:"Reverse agent mode"`

	InfluxDBAddr string `long:"influxdb-addr" description:"Output InfluxDB address"`
	InfluxDBName string `long:"influxdb-name" description:"Output InfluxDB database name"`
}

type agentConfig struct {
	Host string
	Role string
}

func parseAgentConfigs(data string) []agentConfig {
	if _, err := os.Stat(data); os.IsNotExist(err) {
		// Parameter is not a file path
		hosts := strings.Split(data, ",")
		cfgs := make([]agentConfig, 0, len(hosts))
		for _, host := range hosts {
			cfgs = append(cfgs, agentConfig{
				Host: host,
				Role: master.AgentRoleClient,
			})
		}
		return cfgs
	}

	// Parameter is a file path
	f, err := os.Open(data)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	cfgs := []agentConfig{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			log.Fatalf("Invalid agent config: %s", line)
		}

		cfgs = append(cfgs, agentConfig{
			Host: parts[0],
			Role: parts[1],
		})
	}

	if err = scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return cfgs
}

func startMaster() {
	if opts.Server == "" {
		log.Fatalln("Server host:port was not specified")
	}

	if opts.Subject == "" {
		log.Fatalln("Subject was not specified")
	}

	genPidFile("/tmp/websocket-bench-master.pid")

	snapshotWriters := []master.SnapshotWriter{}
	if opts.OutputDir != "" {
		if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
			log.Fatalln(err)
		}
		log.Println("Ouptut directory: ", opts.OutputDir)
		snapshotWriters = append(snapshotWriters, master.NewJsonSnapshotWriter(
			opts.OutputDir))
	}
	if opts.InfluxDBAddr != "" && opts.InfluxDBName != "" {
		log.Println("Write to InfluxDB", opts.InfluxDBAddr, opts.InfluxDBName)
		snapshotWriters = append(snapshotWriters, master.NewInfluxDBSnapshotWriter(
			opts.InfluxDBAddr, opts.InfluxDBName))
	}

	c := master.NewController(snapshotWriters)

	agentCfgs := parseAgentConfigs(opts.Agents)
	log.Println("Agent configs", agentCfgs)

	if len(agentCfgs) == 0 {
		log.Fatal("No agent defined")
	}

	for _, cfg := range agentCfgs {
		if err := c.RegisterAgent(cfg.Host, cfg.Role); err != nil {
			log.Println("Failed to register agent: ", cfg.Host, cfg.Role, err)
		}
	}

	if len(c.Agents) == 0 {
		log.Fatal("No agent can be connected")
	}

	if opts.CollectProcesses != "" {
		c.CollectProcesses = strings.Split(opts.CollectProcesses, ",")
	}

	c.Run(&benchmark.Config{
		Host:     opts.Server,
		Subject:  opts.Subject,
		CmdFile:  opts.CmdFile,
		UseWss:   opts.UseWss,
		SendSize: opts.SendSize,
	})
}

func genPidFile(pidfile string) {
	f, _ := os.Create(pidfile)
	defer func() {
		cerr := f.Close()
		if cerr != nil {
			log.Fatalln("Failed to close the pid file: ", cerr)
		}
	}()
	_, err := f.WriteString(fmt.Sprintf("%d", os.Getpid()))
	if err != nil {
		log.Println("Fail to write pidfile")
	}
}

func startAgent() {
	genPidFile("/tmp/websocket-bench.pid")
	rpc.RegisterName("Agent", &agent.Controller{
		AgentRole: opts.Role,
	})
	if !opts.ReverseAgent {
		l, err := net.Listen("tcp", opts.ListenAddress)
		if err != nil {
			log.Fatal("Failed to listen on "+opts.ListenAddress, err)
		}
		log.Println("Listen on ", l.Addr())
		for {
			conn, err := l.Accept()
			if err == nil {
				log.Println("Accepted conn", conn.RemoteAddr())
				rpc.ServeConn(conn)
				log.Println("Ended conn", conn.RemoteAddr())
				conn.Close()
			} else {
				log.Println(err)
			}
		}
	} else {
		for {
			log.Println("Dialing to forwarder", opts.ListenAddress)
			conn, err := net.Dial("tcp", opts.ListenAddress)
			if err == nil {
				log.Println("Connected")
				rpc.ServeConn(conn)
				conn.Close()
			} else {
				log.Println(err)
				time.Sleep(5 * time.Second)
			}
		}
	}
}

func startForwarder() {
	f := forwarder.NewForwarder()
	addrs := strings.Split(opts.ListenAddress, ",")
	if len(addrs) < 2 {
		log.Fatal("Invalid listen address", opts.ListenAddress)
	}
	if err := f.Listen(addrs[0], addrs[1]); err != nil {
		log.Println(err)
	}
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	switch opts.Mode {
	case "master":
		startMaster()
	case "forwarder":
		startForwarder()
	default:
		startAgent()
	}
}
