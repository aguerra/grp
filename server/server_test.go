package server

import (
	"crypto/tls"
	"io"
	"net"
	"testing"
)

type FakeHandler struct{}

func (h *FakeHandler) Handle(conn net.Conn) {
	buf := make([]byte, 1)
	for {
		if _, err := io.ReadFull(conn, buf); err != nil {
			return
		}
		if _, err := conn.Write(buf); err != nil {
			return
		}
	}
}

func newTestServer(port int) *Server {
	conf := &ServerConfig{
		Port:     port,
		CertFile: "testdata/server.crt",
		KeyFile:  "testdata/server.key",
		CaFile:   "testdata/ca.crt",
	}
	srv := New(conf)
	srv.Handler = &FakeHandler{}
	return srv
}

func newConn(keyFile, certFile, addr string) (*tls.Conn, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	conn, err := tls.Dial("tcp", addr, &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	})
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func TestListenAndServe(t *testing.T) {
	var ln net.Listener
	lnc := make(chan net.Listener, 1)
	errc := make(chan error, 1)
	testHookListenAndServe = func(s *Server, l net.Listener) {
		lnc <- l
	}
	srv := newTestServer(5000)
	go func() { errc <- srv.ListenAndServe() }()
	select {
	case err := <-errc:
		t.Fatal(err)
	case ln = <-lnc:
		defer ln.Close()
		break
	}
	certFile := "testdata/client.crt"
	keyFile := "testdata/client.key"
	conn, err := newConn(keyFile, certFile, "localhost:5000")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	name := "gopher"
	buf := make([]byte, len(name))
	if _, err := conn.Write([]byte(name)); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatal(err)
	}
	if got, want := string(buf), name; got != want {
		t.Errorf("Name = %v; want %v", got, want)
	}
}

func TestListenAndServeErrCredentials(t *testing.T) {
	var ln net.Listener
	lnc := make(chan net.Listener, 1)
	errc := make(chan error, 1)
	testHookListenAndServe = func(s *Server, l net.Listener) {
		lnc <- l
	}
	srv := newTestServer(6000)
	go func() { errc <- srv.ListenAndServe() }()
	select {
	case err := <-errc:
		t.Fatal(err)
	case ln = <-lnc:
		defer ln.Close()
		break
	}
	certFile := "testdata/client_err.crt"
	keyFile := "testdata/client_err.key"
	conn, err := newConn(keyFile, certFile, "localhost:6000")
	if conn != nil {
		t.Errorf("Conn = %v; want nil", conn)
	}
	if err == nil {
		t.Errorf("Got nil; want error")
	}
}
