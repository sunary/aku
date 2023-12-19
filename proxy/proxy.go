package proxy

import (
	"bytes"
	"errors"
	"net"
	"net/http"
	"sort"
	"time"

	"github.com/sunary/aku/config"
	"github.com/sunary/aku/helper"
	"github.com/sunary/aku/loging"
	"go.uber.org/zap"
)

var (
	ll          = loging.New()
	errNotAllow = errors.New("not allow")
	errNotFound = errors.New("not found")
	httpResp    = http.Response{
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}
)

type Proxy struct {
	HttpPort           int
	healthcheckPath    string
	wsPath             string
	GrpcPort           int
	keepGrpcConnection bool

	sortedHttpPrefix []string
	httpProxy        map[string]HttpProxy
	grpcProxy        map[string]GrpcProxy

	ipForwardedHeader string
	httpTimeout       time.Duration
	grpcTimeout       time.Duration
}

func NewProxy(cfg *config.Config) *Proxy {
	httpProxy := map[string]HttpProxy{}
	grpcProxy := map[string]GrpcProxy{}

	sortedPrefix := []string{}
	for _, route := range cfg.Http.RouteMaps {
		if _, ok := httpProxy[route.OverridePath]; ok {
			ll.Fatal("duplicate http overridePath", zap.String("path", route.OverridePath))
		}

		httpProxy[route.OverridePath] = HttpProxy{
			host:          route.Host,
			overridePath:  route.OverridePath,
			upstreamPath:  route.UpstreamPath,
			plugins:       route.Plugins,
			cors:          route.Cors,
			ipRestriction: newIpFilter(route.IpRestriction),
		}

		sortedPrefix = append(sortedPrefix, route.OverridePath)
	}

	// sort url prefix by longest length
	sort.Slice(sortedPrefix, func(i, j int) bool {
		return len(sortedPrefix[i]) > len(sortedPrefix[j])
	})

	for _, method := range cfg.Grpc.MethodMaps {
		if _, ok := grpcProxy[method.ProtoService]; ok {
			ll.Fatal("duplicate grpc protoService", zap.String("service", method.ProtoService))
		}

		grpcProxy[method.ProtoService] = GrpcProxy{
			host:          method.Host,
			protoService:  method.ProtoService,
			allow:         helper.NewStringSet(method.Allow...),
			disallow:      helper.NewStringSet(method.Disallow...),
			plugins:       method.Plugins,
			ipRestriction: newIpFilter(method.IpRestriction),
		}
	}

	return &Proxy{
		HttpPort:           cfg.Http.Port,
		healthcheckPath:    cfg.Http.Healthcheck,
		wsPath:             cfg.Http.Ws,
		GrpcPort:           cfg.Grpc.Port,
		keepGrpcConnection: cfg.Grpc.KeepConnection,
		sortedHttpPrefix:   sortedPrefix,
		httpProxy:          httpProxy,
		grpcProxy:          grpcProxy,
		ipForwardedHeader:  cfg.IpForwardedHeader,
		httpTimeout:        time.Duration(cfg.Http.Timeout) * time.Second,
		grpcTimeout:        time.Duration(cfg.Grpc.Timeout) * time.Second,
	}
}

func (p *Proxy) HandleTcpConn(conn *net.TCPConn) {
	defer conn.Close()

	buf := make([]byte, 4*1024)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}

	reqIp := "0.0.0.0"
	if addr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		reqIp = addr.IP.String()
	}

	firstLineIndex := 0
	for i := 0; i < n; i++ {
		if buf[i] == '\n' {
			firstLineIndex = i
			break
		}
	}

	header := bytes.Split(buf[:firstLineIndex], []byte(" "))
	reqPath := string(header[1])
	reqVersion := string(header[2][:6])

	switch reqVersion {
	case "HTTP/1":
		err = p.handleHttp(conn, bytes.NewBuffer(buf[:n]), reqPath, reqIp)
		if err != nil {
			ll.Error("handle http req", zap.String("path", string(header[1])), loging.Err(err))

			resp := httpResp
			resp.StatusCode = http.StatusInternalServerError
			resp.Write(conn)
			conn.Close()
		}

	case "HTTP/2":
		err = p.handleGrpc(conn, bytes.NewBuffer(buf[:n]), reqIp)
		if err != nil {
			ll.Error("handle grpc req", loging.Err(err))
		}

	default:
		ll.Error("HandleTcpConn unknow http version", zap.String(string(header[0]), reqVersion))
	}

}
