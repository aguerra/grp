package server

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"strconv"
	"time"

	"github.com/aguerra/grp/radius"
	log "github.com/inconshreveable/log15"
)

var testHookListenAndServe func(*Server, net.Listener) // used if non-nil

type Handler interface {
	Handle(net.Conn)
}

type ServerConfig struct {
	Port     int    `default:"2083"`
	CaFile   string `split_words:"true" default:"ca.crt"`
	CertFile string `split_words:"true" default:"server.crt"`
	KeyFile  string `split_words:"true" default:"server.key"`
	radius.RadiusConfig
}

type Server struct {
	conf *ServerConfig
	Handler
}

func (srv *Server) ListenAndServe() error {
	tlsConfig, err := srv.newTLSConfig()
	if err != nil {
		return err
	}
	addr := net.JoinHostPort("", strconv.Itoa(srv.conf.Port))
	ln, err := tls.Listen("tcp", addr, tlsConfig)
	if err != nil {
		return err
	}
	defer ln.Close()
	if fn := testHookListenAndServe; fn != nil {
		fn(srv, ln)
	}
	var tempDelay time.Duration
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				// Taken from net/http server
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.Warn("failed to accept conn", "err", err, "delay", tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		go srv.Handle(conn)
	}
}

func (srv *Server) newTLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(srv.conf.CertFile, srv.conf.KeyFile)
	if err != nil {
		return nil, err
	}
	ca, err := ioutil.ReadFile(srv.conf.CaFile)
	if err != nil {
		return nil, err
	}
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(ca)
	tls := &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}
	return tls, nil
}

func New(conf *ServerConfig) *Server {
	h := radius.NewHandler(&conf.RadiusConfig)
	return &Server{conf: conf, Handler: h}
}
