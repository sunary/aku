package crd

import (
	"fmt"
	"time"

	"github.com/sunary/aku/config"
	"github.com/sunary/aku/loging"
	"go.uber.org/zap"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	ControllerName = "Pod" // "AkuIngress"
)

var (
	ll = loging.New()
)

type Controller struct {
	deployments       appslisters.DeploymentLister
	deploymentsSynced cache.InformerSynced

	// queue is where incoming work is placed to de-dup and to allow "easy"
	// rate limited requeues on errors
	queue workqueue.RateLimitingInterface

	httpChan chan<- config.HttpRoute
	grpcChan chan<- config.GrpcMethod
}

func NewController(deploymentInformer appsinformers.DeploymentInformer,
	httpChan chan<- config.HttpRoute, grpcChan chan<- config.GrpcMethod) *Controller {
	c := &Controller{
		deployments:       deploymentInformer.Lister(),
		deploymentsSynced: deploymentInformer.Informer().HasSynced,

		queue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), ControllerName),
		httpChan: httpChan,
		grpcChan: grpcChan,
	}

	// register event handlers to fill the queue with pod creations, updates and deletions
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				c.queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				c.queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				c.queue.Add(key)
			}
		},
	})

	return c
}

func (c *Controller) Run(threadiness int, stopCh chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	ll.Info("start crd controller")
	if !cache.WaitForCacheSync(stopCh, c.deploymentsSynced) {
		ll.Warn("cannot sync deployment")
		return
	}

	ll.Info("start workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	// wait until we're told to stop
	<-stopCh
	ll.Info("shutting down crd controller")
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem deals with one key off the queue.  It returns false
// when it's time to quit.
func (c *Controller) processNextWorkItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	err := c.syncHandler(key.(string))
	if err == nil {
		c.queue.Forget(key)
		return true
	}

	utilruntime.HandleError(fmt.Errorf("%v failed with : %v", key, err))
	c.queue.AddRateLimited(key)

	return true
}

func (c *Controller) syncHandler(key string) error {
	namespace, name, _ := cache.SplitMetaNamespaceKey(key)
	pod, err := c.deployments.Deployments(namespace).Get(name)
	if err != nil {
		ll.Error("sync handler", zap.String(namespace, name), loging.Err(err))
	}

	if pod.Name == "aku" {
		c.httpChan <- config.HttpRoute{
			Name: "event pod " + pod.Name,
		}
	}

	return nil
}
