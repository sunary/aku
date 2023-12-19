package proxy

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sunary/aku/config"
	"go.uber.org/zap"
)

type (
	fwHttp = func(req *http.Request, conn *net.TCPConn) error
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

	overridePath := ""
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
	ll.Info("handle http req", zap.String("overridePath", overridePath))

	if !hProxy.AllowIp(reqIp) {
		return errNotAllow
	}

	buf := make([]byte, 4*1024)
	nr, _ := initial.Read(buf)

	indexStartPath, indexEndPath := 0, 0
	for i := 0; i < nr; i++ {
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
	newBuf.Write(buf[indexEndPath:nr])

	remoteConn, err := net.Dial("tcp", hProxy.host)
	if err != nil {
		return err
	}

	remoteConn.SetReadDeadline(time.Now().Add(time.Duration(p.grpcTimeout) * time.Second))
	defer remoteConn.Close()

	wg := sync.WaitGroup{}
	dataSent := int64(0)
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Copy any data we receive from the host into the original connection
		dataSent, _ = io.Copy(conn, remoteConn)
		conn.CloseWrite()
	}()

	_, err = io.Copy(remoteConn, io.MultiReader(&newBuf, conn))
	wg.Wait()

	if errors.Is(err, os.ErrDeadlineExceeded) && dataSent > 0 {
		return nil
	}

	return err
}
