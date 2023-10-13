package gateway

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/sunary/aku/config"
	"github.com/sunary/aku/crd"
	"github.com/sunary/aku/loging"
	"github.com/sunary/aku/proxy"
	"go.uber.org/zap"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	ll = loging.New()
)

type Gateway struct {
	proxy    *proxy.Proxy
	httpChan chan config.HttpRoute
	grpcChan chan config.GrpcMethod
}

func NewGateway(cfg *config.Config) *Gateway {
	return &Gateway{
		proxy:    proxy.NewProxy(cfg),
		httpChan: make(chan config.HttpRoute),
		grpcChan: make(chan config.GrpcMethod),
	}
}

func (g *Gateway) Start() error {
	go g.startController()
	go g.reloadProxy()

	wg := sync.WaitGroup{}

	if g.proxy.GrpcPort != g.proxy.HttpPort {
		ll.Info("start listen grpc", zap.Int("port", g.proxy.GrpcPort))
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := g.listen(g.proxy.GrpcPort)
			ll.Error("listen grpc err", loging.Err(err))
		}()
	}

	ll.Info("start listen http", zap.Int("port", g.proxy.HttpPort))
	err := g.listen(g.proxy.HttpPort)
	ll.Error("listen http err", loging.Err(err))

	wg.Wait()
	return nil
}

func (g *Gateway) startController() {
	ctx := context.Background()
	cfg, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		ll.Error("building kubeconfig", loging.Err(err))
		return
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		ll.Error("building kubernetes clientset", loging.Err(err))
		return
	}

	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, time.Second*30)
	controller := crd.NewController(kubeInformerFactory.Apps().V1().Deployments(), g.httpChan, g.grpcChan)
	kubeInformerFactory.Start(ctx.Done())

	stopCh := make(chan struct{})
	controller.Run(1, stopCh)
}

func (g *Gateway) listen(port int) error {
	addr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", port))

	list, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}

	for {
		conn, err := list.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
				return err
			}
			continue
		}

		go g.proxy.HandleTcpConn(conn.(*net.TCPConn))
	}
}

func (g *Gateway) reloadProxy() {
	for {
		httpEvent := <-g.httpChan
		ll.Info("received crd event", zap.String("name", httpEvent.Name))
	}
}
