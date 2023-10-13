package proxy

import (
	"net"
	"net/http"
)

// https://github.com/koding/websocketproxy/blob/master/websocketproxy.go
func (p *Proxy) forwardWs(req *http.Request, conn *net.TCPConn) error {
	return nil
}
