package proxy

import (
	"bytes"
	"io"
	"net"
	"strings"

	"github.com/sunary/aku/config"
	"go.uber.org/zap"
)

type HttpProxy struct {
	host          string
	overridePath  string // key
	upstreamPath  string
	plugins       []string
	cors          *config.HttpCors
	ipRestriction *ipFilter
}

func (h HttpProxy) AllowIp(ip string) bool {
	return h.ipRestriction.Allow(ip)
}

func (p *Proxy) handleHttp(conn *net.TCPConn, initial io.Reader, reqPath, reqIp string) error {
	defer func() {
		_ = conn.Close()
	}()

	var overridePath string
	for i := range p.sortedHttpPrefix {
		if strings.HasPrefix(reqPath, p.sortedHttpPrefix[i]) {
			overridePath = p.sortedHttpPrefix[i]
			break
		}
	}

	if overridePath == "" {
		return errNotFound
	}

	hProxy := p.httpProxy[overridePath]
	ll.Info("handleHttp req", zap.String("gateway", reqPath), zap.String(hProxy.host, overridePath))

	if !hProxy.AllowIp(reqIp) {
		return errNotAllow
	}

	buf := make([]byte, 4*1024)
	headerSize, _ := initial.Read(buf)

	indexStartPath, indexEndPath := 0, 0
	for i := 0; i < headerSize; i++ {
		if buf[i] == '/' && indexStartPath == 0 {
			indexStartPath = i
		}
		if buf[i] == ' ' && indexStartPath > 0 {
			indexEndPath = i
			break
		}
	}

	newPath := strings.Replace(reqPath, overridePath, hProxy.upstreamPath, 1)
	newBuf := bytes.Buffer{}
	newBuf.Write(buf[:indexStartPath])
	newBuf.WriteString(newPath)
	newBuf.Write(buf[indexEndPath:headerSize])

	return ioBind(conn, hProxy.host, p.httpTimeout, &newBuf, conn)
}
