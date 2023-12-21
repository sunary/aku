package proxy

import (
	"bytes"
	"io"
	"net"
	"strings"

	"github.com/sunary/aku/helper"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

type GrpcProxy struct {
	host          string
	protoService  string // key
	allow         *helper.StringSet
	disallow      *helper.StringSet
	plugins       []string
	ipRestriction *ipFilter
}

func (g GrpcProxy) AllowIp(ip string) bool {
	return g.ipRestriction.Allow(ip)
}

func (g GrpcProxy) AllowMethod(method string) bool {
	return g.allow.Has(method) || !(g.disallow.Empty() || g.disallow.Has(method))
}

func (p *Proxy) handleGrpc(conn *net.TCPConn, initial io.Reader, reqIp string) error {
	defer func() {
		_ = conn.Close()
	}()

	dataBuffer := bytes.NewBuffer(make([]byte, 0))
	reader := io.TeeReader(conn, dataBuffer)
	f := http2.NewFramer(conn, conn)
	err := f.WriteSettingsAck()
	if err != nil {
		return err
	}

	f = http2.NewFramer(io.Discard, reader)
	decoder := hpack.NewDecoder(1024, nil)

	var path string // "/pb.ProtoService/Method"
	for path == "" {
		frame, err := f.ReadFrame()
		if err != nil {
			return err
		}

		switch t := frame.(type) {
		case *http2.HeadersFrame:
			out, err := decoder.DecodeFull(t.HeaderBlockFragment())
			if err != nil {
				return err
			}

			for _, v := range out {
				if v.Name == ":path" {
					path = v.Value
					break
				}
			}
		}
	}

	pathFactor := strings.Split(path, "/")
	if len(pathFactor) != 3 {
		return errNotFound
	}

	gProxy, ok := p.grpcProxy[pathFactor[1]]
	if !ok {
		return errNotFound
	}

	ll.Info("handleGrpc conn", zap.String(gProxy.host, path))

	if !gProxy.AllowIp(reqIp) {
		return errNotAllow
	}

	if !gProxy.AllowMethod(pathFactor[2]) {
		return errNotAllow
	}

	return ioBind(conn, gProxy.host, p.grpcTimeout, initial, dataBuffer, conn)
}
