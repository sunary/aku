package transport

import (
	"crypto/tls"
	"crypto/x509"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func httpTls(insec bool) {
	var tlsConfig tls.Config
	if insec {
		tlsConfig = tls.Config{InsecureSkipVerify: true}
	} else {
		clientTLSCert, _ := tls.LoadX509KeyPair("", "")
		systemRoots, _ := x509.SystemCertPool()
		tlsConfig = tls.Config{
			RootCAs:      systemRoots,
			Certificates: []tls.Certificate{clientTLSCert},
		}
	}

	_ = tlsConfig
}

func grpcTls(insec bool) {
	var cred credentials.TransportCredentials
	if insec {
		cred = insecure.NewCredentials()
	} else {
		systemRoots, _ := x509.SystemCertPool()

		cred = credentials.NewTLS(&tls.Config{
			RootCAs: systemRoots,
		})
	}

	_ = cred
}
