package netutil

import (
	"net"
	"time"
)

// https://github.com/golang/go/blob/c80e0d374ba3caf8ee32c6fe4a5474fa33928086/src/net/http/server.go
//
// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type TCPKeepAliveListener struct {
	*net.TCPListener
}

func (ln TCPKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

func NewTCPKeepAliveListener(delegate *net.TCPListener) net.Listener {
	return &TCPKeepAliveListener{delegate}
}
