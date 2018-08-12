package benchmark

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"aspnet.com/util"
	"github.com/gorilla/websocket"
	"github.com/teris-io/shortid"
	"github.com/vmihailenco/msgpack"
)

type SignalrServiceHandshake struct {
	ServiceUrl string `json:"url"`
	JwtBearer  string `json:"accessToken"`
}
type ProtocolProcessing interface {
	IsJson() bool
	IsMsgpack() bool
	LatencyCheckTarget() string
	JoinGroupTarget() string
	LeaveGroupTarget() string
}

type SignalrCoreCommon struct {
	ProtocolProcessing
	WithCounter
	WithSessions
	JsonReceiveFuncs    []func(p ProtocolProcessing, content SignalRCoreInvocation, recvSize int64) bool
	MsgpackReceiveFuncs []func(p ProtocolProcessing, content MsgpackInvocation, recvSize int64) bool
}

func (s *SignalrCoreCommon) IsJson() bool {
	return false
}

func (s *SignalrCoreCommon) IsMsgpack() bool {
	return false
}

func (s *SignalrCoreCommon) LatencyCheckTarget() string {
	return "dummy"
}

func (s *SignalrCoreCommon) JoinGroupTarget() string {
	return "dummy"
}

func (s *SignalrCoreCommon) LeaveGroupTarget() string {
	return "dummy"
}

func (s *SignalrCoreCommon) Setup(config *Config, p ProtocolProcessing) error {
	s.host = config.Host
	s.useWss = config.UseWss
	s.sendSize = config.SendSize
	s.counter = util.NewCounter()
	s.sessions = make([]*Session, 0, 30000)
	s.received = make(chan MessageReceived)
	if p.IsJson() {
		s.JsonReceiveFuncs = make([]func(ProtocolProcessing, SignalRCoreInvocation, int64) bool, 0, 2)
		s.JsonReceiveFuncs = append(s.JsonReceiveFuncs, s.ProcessJsonLatency)
		s.JsonReceiveFuncs = append(s.JsonReceiveFuncs, s.ProcessJsonJoinLeaveGroup)
		go s.ProcessJson(p)
	} else if p.IsMsgpack() {
		s.MsgpackReceiveFuncs = make([]func(ProtocolProcessing, MsgpackInvocation, int64) bool, 0, 2)
		s.MsgpackReceiveFuncs = append(s.MsgpackReceiveFuncs, s.ProcessMsgPackLatency)
		s.MsgpackReceiveFuncs = append(s.MsgpackReceiveFuncs, s.ProcessMsgPackJoinLeaveGroup)
		go s.ProcessMsgPack(p)
	}
	return nil
}

func (s *SignalrCoreCommon) SignalrCoreBaseConnect(protocol string) (session *Session, err error) {
	defer func() {
		if err != nil {
			s.counter.Stat("connection:inprogress", -1)
			s.counter.Stat("connection:error", 1)
		}
	}()

	id, err := shortid.Generate()
	if err != nil {
		log.Println("ERROR: failed to generate uid due to", err)
		return
	}
	var sendName string
	if s.sendSize == 0 {
		sendName = id
	} else {
		sendName = RandStringBytesMaskImprSrc(s.sendSize)
	}

	s.counter.Stat("connection:inprogress", 1)
	wsURL := "ws://" + s.host
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		s.LogError("connection:error", id, "Failed to connect to websocket", err)
		return nil, err
	}

	session = NewSession(id, sendName, s.received, s.counter, c)
	if session != nil {
		s.counter.Stat("connection:inprogress", -1)
		s.counter.Stat("connection:established", 1)

		session.Start()
		session.NegotiateProtocol(protocol)
		return
	}

	err = fmt.Errorf("Nil session")
	return
}

func (s *SignalrCoreCommon) SignalrCoreJsonConnect() (*Session, error) {
	return s.SignalrCoreBaseConnect("json")
}

func (s *SignalrCoreCommon) SignalrCoreMsgPackConnect() (session *Session, err error) {
	return s.SignalrCoreBaseConnect("messagepack")
}

func (s *SignalrCoreCommon) SignalrServiceBaseConnect(protocol string) (session *Session, err error) {
	defer func() {
		if err != nil {
			s.counter.Stat("connection:inprogress", -1)
			s.counter.Stat("connection:error", 1)
		}
	}()

	s.counter.Stat("connection:inprogress", 1)
	id, err := shortid.Generate()
	if err != nil {
		log.Println("ERROR: failed to generate uid due to", err)
		return
	}
	var sendName string
	if s.sendSize == 0 {
		sendName = id
	} else {
		sendName = RandStringBytesMaskImprSrc(s.sendSize)
	}

	negotiateResponse, err := http.Get("http://" + s.host + "/negotiate")
	if err != nil {
		s.LogError("connection:error", id, "Failed to negotiate with the server", err)
		return
	}
	defer negotiateResponse.Body.Close()

	decoder := json.NewDecoder(negotiateResponse.Body)
	var handshake SignalrServiceHandshake
	err = decoder.Decode(&handshake)
	if err != nil {
		s.LogError("connection:error", id, "Failed to decode service URL and jwtBearer", err)
		return
	}

	baseURL := strings.Replace(handshake.ServiceUrl, "http", "ws", 1)
	wsURL := baseURL + "&access_token=" + handshake.JwtBearer

	dialer := &websocket.Dialer{
		NetDial: func(network, addr string) (net.Conn, error) {
			start := time.Now()
			conn, err := net.Dial(network, addr)
			duration := time.Now().Sub(start) / time.Millisecond
			s.LogLatency("dial", int64(duration))

			if e := conn.(*net.TCPConn).SetLinger(0); e != nil {
				log.Println("Fail to set linger", e)
			}

			return conn, err
		},
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
	}
	c, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		s.LogError("connection:error", id, "Failed to connect to websocket", err)
		return
	}
	session = NewSession(id, sendName, s.received, s.counter, c)
	if session != nil {
		s.counter.Stat("connection:inprogress", -1)
		s.counter.Stat("connection:established", 1)

		session.Start()
		session.NegotiateProtocol(protocol)
		return
	}

	err = fmt.Errorf("Nil session")
	return
}

func (s *SignalrCoreCommon) SignalrServiceJsonConnect() (session *Session, err error) {
	return s.SignalrServiceBaseConnect("json")
}

func (s *SignalrCoreCommon) SignalrServiceMsgPackConnect() (session *Session, err error) {
	return s.SignalrServiceBaseConnect("messagepack")
}

var numBitsToShift = []uint{0, 7, 14, 21, 28}

func (s *SignalrCoreCommon) ParseBinaryMessage(bytes []byte) ([]byte, error) {
	moreBytes := true
	msgLen := 0
	numBytes := 0
	for moreBytes && numBytes < len(bytes) && numBytes < 5 {
		byteRead := bytes[numBytes]
		msgLen = msgLen | int(uint(byteRead&0x7F)<<numBitsToShift[numBytes])
		numBytes++
		moreBytes = (byteRead & 0x80) != 0
	}

	if msgLen+numBytes > len(bytes) {
		return nil, fmt.Errorf("Not enough data in message, message length = %d, length section bytes = %d, data length = %d", msgLen, numBytes, len(bytes))
	}

	return bytes[numBytes : numBytes+msgLen], nil
}

func (s *SignalrCoreCommon) ProcessJsonLatency(p ProtocolProcessing, content SignalRCoreInvocation, recvSize int64) bool {
	if content.Type == 1 && content.Target == p.LatencyCheckTarget() {
		sendStart, err := strconv.ParseInt(content.Arguments[1], 10, 64)
		if err != nil {
			s.LogError("message:decode_error", "", "Failed to decode start timestamp", err)
			return false
		}
		s.counter.Stat("message:received", 1)
		s.counter.Stat("message:recvSize", recvSize)
		s.LogLatency("message", (time.Now().UnixNano()-sendStart)/1000000)
		return true
	}
	return false
}

func (s *SignalrCoreCommon) ProcessJsonJoinLeaveGroup(p ProtocolProcessing, content SignalRCoreInvocation, recvSize int64) bool {
	if content.Type == 1 {
		if content.Target == p.JoinGroupTarget() {
			s.counter.Stat("connection:groupjoin", 1)
			return true
		} else if content.Target == p.LeaveGroupTarget() {
			s.counter.Stat("connection:groupjoin", -1)
			return true
		}
	}
	return false
}

func (s *SignalrCoreCommon) ProcessJson(p ProtocolProcessing) {
	for msgReceived := range s.received {
		// Multiple json responses may be merged to be a list.
		// Split them and remove '0x1e' terminator.
		dataArray := bytes.Split(msgReceived.Content, []byte{0x1e})
		for _, msg := range dataArray {
			if len(msg) == 0 {
				// ignore empty msg caused by split
				continue
			}
			var common SignalRCommon
			err := json.Unmarshal(msg, &common)
			if err != nil {
				fmt.Printf("%s\n", msg)
				s.LogError("message:decode_error", msgReceived.ClientID, "Failed to decode incoming message common header", err)
				continue
			}

			// ignore ping
			if common.Type != 1 {
				continue
			}

			var content SignalRCoreInvocation
			err = json.Unmarshal(msg, &content)
			if err != nil {
				s.LogError("message:decode_error", msgReceived.ClientID, "Failed to decode incoming SignalR invocation message", err)
				continue
			}
			for _, recvFunc := range s.JsonReceiveFuncs {
				recvFunc(p, content, int64(len(msg)))
			}
		}
	}
}

func (s *SignalrCoreCommon) ProcessMsgPackLatency(p ProtocolProcessing, content MsgpackInvocation, recvSize int64) bool {
	if content.MessageType == 1 && content.Target == p.LatencyCheckTarget() {
		sendStart, err := strconv.ParseInt(content.Params[1], 10, 64)
		if err != nil {
			s.LogError("message:decode_error", "", "Failed to decode start timestamp", err)
			return false
		}
		s.counter.Stat("message:received", 1)
		s.counter.Stat("message:recvSize", recvSize)
		s.LogLatency("message", (time.Now().UnixNano()-sendStart)/1000000)
		return true
	}
	return true
}

func (s *SignalrCoreCommon) ProcessMsgPackJoinLeaveGroup(p ProtocolProcessing, content MsgpackInvocation, recvSize int64) bool {
	if content.MessageType == 1 {
		if content.Target == p.JoinGroupTarget() {
			s.counter.Stat("message:groupjoin", 1)
			return true
		} else if content.Target == p.LeaveGroupTarget() {
			s.counter.Stat("message:groupjoin", -1)
			return true
		}
	}
	return false
}

func (s *SignalrCoreCommon) ProcessMsgPack(p ProtocolProcessing) {
	for msgReceived := range s.received {
		msg, err := s.ParseBinaryMessage(msgReceived.Content)
		if err != nil {
			s.LogError("message:decode_error", msgReceived.ClientID, "Failed to parse incoming message", err)
			continue
		}
		var content MsgpackInvocation
		err = msgpack.Unmarshal(msg, &content)
		if err != nil {
			s.LogError("message:decode_error", msgReceived.ClientID, "Failed to decode incoming message", err)
			continue
		}

		for _, recvFunc := range s.MsgpackReceiveFuncs {
			if recvFunc(p, content, int64(len(msgReceived.Content))) {
				break
			}
		}
	}
}
