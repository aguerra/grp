package radius

import (
	"bytes"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

var buf = []byte{
	0x01, 0x02, 0x00, 0x1e, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
	0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
	0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
}

var packet = Packet{
	Header: Header{
		Code:       1,
		Identifier: 2,
		Length:     30,
	},
	Data: []byte{
		0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
		0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
		0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
	},
}

type ErrWriter struct{}

type FakeDispatcher struct{}

func (w *ErrWriter) Write(b []byte) (int, error) {
	return 0, errors.New("Error")
}

func (d *FakeDispatcher) Dispatch(p *Packet) (*Packet, error) {
	return p, nil
}

func packetsEqual(p, q *Packet) bool {
	return p.Code == q.Code && p.Identifier == q.Identifier &&
		p.Length == q.Length && bytes.Equal(p.Data, q.Data)
}

func serveEchoUDP(connc chan<- net.Conn) error {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	connc <- conn
	buf := make([]byte, maxPacket)
	for {
		_, raddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			return err
		}
		if _, err := conn.WriteToUDP(buf, raddr); err != nil {
			return err
		}
	}
}

func newUDPDispatcher(addr net.Addr, timeout time.Duration) *UDPDispatcher {
	return &UDPDispatcher{
		host:     "127.0.0.1",
		port:     addr.(*net.UDPAddr).Port,
		acctHost: "127.0.0.1",
		acctPort: addr.(*net.UDPAddr).Port,
		timeout:  timeout,
	}
}

func TestNewPacket(t *testing.T) {
	p, err := NewPacket(bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}
	if !packetsEqual(&packet, p) {
		t.Errorf("Got = %v; want %v", *p, packet)
	}
}

func TestNewPacketUnexpectedEOF(t *testing.T) {
	tmp := []byte{
		0x01, 0x02, 0x00, 0x1e, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
	}
	p, err := NewPacket(bytes.NewReader(tmp))
	if p != nil {
		t.Errorf("Packet = %v; want nil", p)
	}
	if err != io.ErrUnexpectedEOF {
		t.Errorf("Err = %v; want unexpected EOF", err)
	}
}

func TestPacketWriteTo(t *testing.T) {
	var w bytes.Buffer
	n, err := packet.WriteTo(&w)
	if err != nil {
		t.Fatal(err)
	}
	b := w.Bytes()
	if n != len(b) {
		t.Errorf("Written = %d; want %d", n, len(b))
	}
	if !bytes.Equal(buf, b) {
		t.Errorf("Got = %v; want %v", b, buf)
	}
}

func TestPacketWriteToErr(t *testing.T) {
	var w ErrWriter
	n, err := packet.WriteTo(&w)
	if n != 0 {
		t.Errorf("Written = %d; want 0", n)
	}
	if err == nil {
		t.Errorf("Got nil; want error")
	}
}

func TestPacketIsAcctFalse(t *testing.T) {
	if packet.IsAcct() {
		t.Errorf("Got unexpected acct packet %v", packet)
	}
}

func TestPacketIsAcctTrue(t *testing.T) {
	tmp := make([]byte, len(buf))
	copy(tmp, buf)
	p, err := NewPacket(bytes.NewReader(tmp))
	if err != nil {
		t.Fatal(err)
	}
	p.Code = acctCode
	if !p.IsAcct() {
		t.Errorf("Got unexpected non acct packet %v", p)
	}
}

func TestUDPDispatcher(t *testing.T) {
	var conn net.Conn
	connc := make(chan net.Conn, 1)
	errc := make(chan error, 1)
	go func() { errc <- serveEchoUDP(connc) }()
	select {
	case err := <-errc:
		t.Fatal(err)
	case conn = <-connc:
		defer conn.Close()
		break
	}
	d := newUDPDispatcher(conn.LocalAddr(), 2*time.Second)
	resp, err := d.Dispatch(&packet)
	if err != nil {
		t.Fatal(err)
	}
	if !packetsEqual(&packet, resp) {
		t.Errorf("Got = %v; want %v", *resp, packet)
	}
}

func TestUDPDispatcherTimeout(t *testing.T) {
	var conn net.Conn
	connc := make(chan net.Conn, 1)
	errc := make(chan error, 1)
	go func() { errc <- serveEchoUDP(connc) }()
	select {
	case err := <-errc:
		t.Fatal(err)
	case conn = <-connc:
		defer conn.Close()
		break
	}
	d := newUDPDispatcher(conn.LocalAddr(), 0)
	resp, err := d.Dispatch(&packet)
	if resp != nil {
		t.Errorf("Resp = %v; want nil", resp)
	}
	if err, ok := err.(net.Error); ok && !err.Timeout() {
		t.Errorf("Got net error = %v; want timeout", err)
	}
}

func TestHandler(t *testing.T) {
	d := &FakeDispatcher{}
	conf := &RadiusConfig{IdleTimeout: 2 * time.Second}
	h := NewHandler(conf)
	h.Dispatcher = d
	conn1, conn2 := net.Pipe()
	defer func() {
		conn1.Close()
		conn2.Close()
	}()
	go func() { h.Handle(conn2) }()
	_, err := packet.WriteTo(conn1)
	if err != nil {
		t.Fatal(err)
	}
	p, err := NewPacket(conn1)
	if err != nil {
		t.Fatal(err)
	}
	if !packetsEqual(&packet, p) {
		t.Errorf("Got = %v; want %v", *p, packet)
	}
}
