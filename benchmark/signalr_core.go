package benchmark

import (
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack"
)

const SignalRMessageTerminator = '\x1e'

type SignalRCoreHandshakeResp struct {
	AvailableTransports []string `json:"availableTransports"`
	ConnectionId        string   `json:"connectionId"`
}

type SignalRCoreInvocation struct {
	InvocationId string   `json:"invocationId"`
	Type         int      `json:"type"`
	Target       string   `json:"target"`
	NonBlocking  bool     `json:"nonBlocking"`
	Arguments    []string `json:"arguments"`
}

type MsgpackInvocation struct {
	MessageType  int32
	InvocationID string
	Target       string
	Params       []string
}

func (m *MsgpackInvocation) EncodeMsgpack(enc *msgpack.Encoder) error {
	enc.EncodeArrayLen(4)
	return enc.Encode(m.MessageType, m.InvocationID, m.Target, m.Params)
}

func (m *MsgpackInvocation) DecodeMsgpack(dec *msgpack.Decoder) error {
	dec.DecodeArrayLen()
	messageType, err := dec.DecodeInt32()
	if err != nil {
		log.Printf("Failed to decode message %v\n", dec)
		return err
	}
	m.MessageType = messageType
	if messageType == 1 {
		return dec.Decode(&m.InvocationID, &m.Target, &m.Params)
	}
	return nil
}

type WithInterval struct {
	interval time.Duration
}

func (w *WithInterval) Interval() time.Duration {
	return w.interval
}

type SignalRCoreTextMessageGenerator struct {
	invocationId int
	WithInterval
}

var _ MessageGenerator = (*SignalRCoreTextMessageGenerator)(nil)

func (g *SignalRCoreTextMessageGenerator) Generate(uid string) Message {
	g.invocationId++
	msg, err := json.Marshal(&SignalRCoreInvocation{
		Type:         1,
		InvocationId: strconv.Itoa(g.invocationId),
		Target:       "echo",
		Arguments: []string{
			uid,
			strconv.FormatInt(time.Now().UnixNano(), 10),
		},
		NonBlocking: false,
	})
	if err != nil {
		log.Println("ERROR: failed to encoding SignalR message", err)
		return nil
	}
	msg = append(msg, SignalRMessageTerminator)
	return PlainMessage{websocket.TextMessage, msg}
}

type MessagePackMessageGenerator struct {
	invocationId int
	WithInterval
}

var _ MessageGenerator = (*MessagePackMessageGenerator)(nil)

func appendLength(bytes []byte) []byte {
	buffer := make([]byte, 0, 5+len(bytes))
	length := len(bytes)
	for length > 0 {
		current := byte(length & 0x7F)
		length >>= 7
		if length > 0 {
			current |= 0x80
		}
		buffer = append(buffer, current)
	}
	if len(buffer) == 0 {
		buffer = append(buffer, 0)
	}
	buffer = append(buffer, bytes...)
	return buffer
}

func (g MessagePackMessageGenerator) Generate(uid string) Message {
	g.invocationId++
	invocation := MsgpackInvocation{
		MessageType:  1,
		InvocationID: strconv.Itoa(g.invocationId),
		Target:       "echo",
		Params: []string{
			uid,
			strconv.FormatInt(time.Now().UnixNano(), 10),
		},
	}
	msg, err := msgpack.Marshal(&invocation)
	if err != nil {
		log.Fatalln("Fail to pack signalr core message", err)
		return nil
	}
	msg = appendLength(msg)
	return PlainMessage{websocket.BinaryMessage, msg}
}