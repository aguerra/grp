package radius

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"strconv"
	"time"
	"unsafe"

	log "github.com/inconshreveable/log15"
)

const (
	acctCode  = 4
	maxPacket = 4096
)

type Header struct {
	Code       uint8
	Identifier uint8
	Length     uint16
}

type Packet struct {
	Header
	Data []byte
}

type Dispatcher interface {
	Dispatch(*Packet) (*Packet, error)
}

type UDPDispatcher struct {
	host     string
	port     int
	acctHost string
	acctPort int
	timeout  time.Duration
}

type RadiusConfig struct {
	RadiusHost     string        `split_words:"true" default:"localhost"`
	RadiusPort     int           `split_words:"true" default:"1812"`
	RadiusAcctHost string        `split_words:"true" default:"localhost"`
	RadiusAcctPort int           `split_words:"true" default:"1813"`
	RadiusTimeout  time.Duration `split_words:"true" default:"10s"`
	IdleTimeout    time.Duration `split_words:"true" default:"60s"`
}

type Handler struct {
	Dispatcher
	conf *RadiusConfig
}

func NewPacket(r io.Reader) (*Packet, error) {
	var p Packet
	if err := binary.Read(r, binary.BigEndian, &p.Header); err != nil {
		return nil, err
	}
	s := unsafe.Sizeof(p.Header)
	p.Data = make([]byte, p.Length-uint16(s))
	if _, err := io.ReadFull(r, p.Data); err != nil {
		return nil, err
	}
	return &p, nil
}

func (p *Packet) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, p.Header); err != nil {
		return nil, err
	}
	if _, err := buf.Write(p.Data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *Packet) WriteTo(w io.Writer) (int, error) {
	b, err := p.Bytes()
	if err != nil {
		return 0, err
	}
	return w.Write(b)
}

func (p *Packet) IsAcct() bool {
	if p.Code == acctCode {
		return true
	}
	return false
}

func (d *UDPDispatcher) Dispatch(p *Packet) (*Packet, error) {
	host := d.host
	port := d.port
	if p.IsAcct() {
		host = d.acctHost
		port = d.acctPort
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	b, err := p.Bytes()
	if err != nil {
		return nil, err
	}
	if _, err := conn.Write(b); err != nil {
		return nil, err
	}
	buf := make([]byte, maxPacket)
	conn.SetReadDeadline(time.Now().Add(d.timeout))
	_, _, err = conn.ReadFromUDP(buf)
	if err != nil {
		return nil, err
	}
	return NewPacket(bytes.NewReader(buf))
}

func (h *Handler) Handle(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	conn.SetDeadline(time.Now().Add(h.conf.IdleTimeout))
	raddr := conn.RemoteAddr()
	for {
		p, err := NewPacket(r)
		if err != nil {
			log.Error("reading", "err", err, "raddr", raddr)
			return
		}
		conn.SetDeadline(time.Now().Add(h.conf.IdleTimeout))
		log.Debug("packet", "header", p.Header, "raddr", raddr)
		go func() {
			resp, err := h.Dispatch(p)
			if err != nil {
				log.Error("dispatching", "err", err, "raddr", raddr)
				return
			}
			log.Debug("response", "header", resp.Header, "raddr", raddr)
			if _, err := resp.WriteTo(conn); err != nil {
				log.Error("writing", "err", err, "raddr", raddr)
				return
			}
			conn.SetDeadline(time.Now().Add(h.conf.IdleTimeout))
		}()
	}
}

func NewHandler(conf *RadiusConfig) *Handler {
	d := &UDPDispatcher{
		host:     conf.RadiusHost,
		port:     conf.RadiusPort,
		acctHost: conf.RadiusAcctHost,
		acctPort: conf.RadiusAcctPort,
		timeout:  conf.RadiusTimeout,
	}
	return &Handler{Dispatcher: d, conf: conf}
}
