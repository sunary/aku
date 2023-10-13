package proxy

import (
	"net/http"
	"strings"

	"github.com/sunary/aku/helper"
)

type ipFilter struct {
	ips *helper.StringSet
}

func newIpFilter(ip []string) *ipFilter {
	return &ipFilter{
		ips: helper.NewStringSet(ip...),
	}
}

func (f ipFilter) Allow(ip string) bool {
	return ip == "" || f.ips.Empty() || f.ips.Has(ip)
}

func getIP(req *http.Request, header string) string {
	return strings.Split(req.Header.Get(header), ", ")[0]
}
