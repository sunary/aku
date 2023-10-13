package proxy

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"

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

func (p *Proxy) handleHttpReq(req *http.Request, conn *net.TCPConn, fw fwHttp) error {
	overridePath := ""
	for i := range p.sortedHttpPrefix {
		if strings.HasPrefix(req.URL.Path, p.sortedHttpPrefix[i]) {
			overridePath = p.sortedHttpPrefix[i]
			break
		}
	}

	if overridePath == "" {
		return errNotFound
	}

	hProxy := p.httpProxy[overridePath]
	ll.Info("handle http req", zap.String("overridePath", overridePath))

	ip := getIP(req, p.ipForwardedHeader)
	if !hProxy.AllowIp(ip) {
		return errNotAllow
	}

	req.URL = &url.URL{
		Scheme: "http",
		Host:   hProxy.host,
		Path:   strings.Replace(req.URL.Path, overridePath, hProxy.upstreamPath, 1),
	}

	return fw(req, conn)
}

// TODO improve using pipe
func (p *Proxy) forwardHttp(req *http.Request, conn *net.TCPConn) error {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}

	req.Body = ioutil.NopCloser(bytes.NewReader(body))
	proxyReq, err := http.NewRequest(req.Method, req.URL.String(), bytes.NewReader(body))
	proxyReq.Header = make(http.Header)
	for h, val := range req.Header {
		proxyReq.Header[h] = val
	}

	httpClient := http.Client{
		Timeout: p.httpTimeout,
	}
	resp, err := httpClient.Do(proxyReq)
	if err != nil {
		return err
	}
	return resp.Write(conn)
}
