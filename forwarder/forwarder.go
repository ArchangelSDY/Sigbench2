package forwarder

import (
	"bufio"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
)

type Forwarder struct {
	lock        *sync.Mutex
	listenAddrs map[string]bool
}

func NewForwarder() *Forwarder {
	return &Forwarder{
		lock:        &sync.Mutex{},
		listenAddrs: make(map[string]bool),
	}
}

func (f *Forwarder) Listen(addr, managementAddr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	go f.listenManagement(managementAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Error to accept new connection", err)
			continue
		}

		go f.handleAgentConnection(conn)
	}
}

func (f *Forwarder) handleAgentConnection(conn net.Conn) {
	defer conn.Close()

	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Println("Error to listen on a local for", conn.RemoteAddr().String(), err)
		return
	}

	log.Println("Forwarding", ln.Addr().String(), "->", conn.RemoteAddr())

	f.lock.Lock()
	f.listenAddrs[ln.Addr().String()] = true
	f.lock.Unlock()

	defer func() {
		log.Println("Stop forwarding", ln.Addr().String(), "->", conn.RemoteAddr())

		f.lock.Lock()
		delete(f.listenAddrs, ln.Addr().String())
		f.lock.Unlock()
	}()

	pr, pw := io.Pipe()
	go func() {
		io.Copy(pw, conn)
		ln.Close()
	}()

	fconn, err := ln.Accept()
	if err != nil {
		return
	}
	defer fconn.Close()

	log.Printf("Established connection %s - %s - %s",
		fconn.RemoteAddr().String(),
		fconn.LocalAddr().String(),
		conn.RemoteAddr().String())

	c := make(chan error, 2)
	go func() {
		_, err := io.Copy(conn, fconn)
		c <- err
	}()
	go func() {
		_, err := io.Copy(fconn, pr)
		c <- err
	}()

	<-c
}

func (f *Forwarder) listenManagement(addr string) error {
	http.HandleFunc("/agents", func(w http.ResponseWriter, req *http.Request) {
		f.lock.Lock()
		bw := bufio.NewWriter(w)
		for k, _ := range f.listenAddrs {
			bw.WriteString(k + "\n")
		}
		bw.Flush()
		f.lock.Unlock()
	})
	return http.ListenAndServe(addr, nil)
}
