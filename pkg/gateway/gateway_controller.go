package gateway

import (
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	gwclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
	gwinformers "sigs.k8s.io/gateway-api/pkg/client/informers/externalversions"
)

const (
	// maxRetries is the number of times a object will be retried before it is dropped out of the queue.
	// With the current rate-limiter in use (5ms*2^(maxRetries-1)) the following numbers represent the
	// sequence of delays between successive queuings of an object.
	//
	// 5ms, 10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.3s, 2.6s, 5.1s, 10.2s, 20.4s, 41s, 82s
	maxRetries     = 15
	controllerName = "gateway-controller"
)

// NewController returns a new *Controller.
func NewController(client clientset.Interface,
	gwclient gwclient.Interface,
	gatewayInformer gwinformers.SharedInformerFactory,
) *Controller {
	klog.V(4).Info("Creating event broadcaster")
	broadcaster := record.NewBroadcaster()
	broadcaster.StartStructuredLogging(0)
	broadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: client.CoreV1().Events("")})
	recorder := broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: controllerName})

	c := &Controller{
		client:           gwclient,
		queue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), controllerName),
		workerLoopPeriod: time.Second,
	}

	// services
	klog.Info("Setting up event handlers for services")

	c.eventBroadcaster = broadcaster
	c.eventRecorder = recorder

	return c
}

// Controller manages selector-based service endpoints.
type Controller struct {
	client           gwclient.Interface
	eventBroadcaster record.EventBroadcaster
	eventRecorder    record.EventRecorder

	queue workqueue.RateLimitingInterface

	// workerLoopPeriod is the time between worker runs. The workers process the queue of service and pod changes.
	workerLoopPeriod time.Duration
}

// Run will not return until stopCh is closed. workers determines how many
// endpoints will be handled in parallel.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Infof("Starting controller %s", controllerName)
	defer klog.Infof("Shutting down controller %s", controllerName)

	// Wait for the caches to be synced
	klog.Info("Waiting for informer caches to sync")
	if !cache.WaitForNamedCacheSync(controllerName, stopCh) {
		return fmt.Errorf("error syncing cache")
	}

	// Start the workers after the repair loop to avoid races
	klog.Info("Starting workers")
	for i := 0; i < workers; i++ {
		go wait.Until(c.worker, c.workerLoopPeriod, stopCh)
	}

	<-stopCh
	return nil
}

// worker runs a worker thread that just dequeues items, processes them, and
// marks them done. You may run as many of these in parallel as you wish; the
// workqueue guarantees that they will not end up processing the same service
// at the same time.
func (c *Controller) worker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	eKey, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(eKey)

	err := c.syncServices(eKey.(string))
	c.handleErr(err, eKey)

	return true
}

func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		c.queue.Forget(key)
		return
	}

	ns, name, keyErr := cache.SplitMetaNamespaceKey(key.(string))
	if keyErr != nil {
		klog.ErrorS(err, "Failed to split meta namespace cache key", "key", key)
	}

	if c.queue.NumRequeues(key) < maxRetries {
		klog.V(2).InfoS("Error syncing service, retrying", "service", klog.KRef(ns, name), "err", err)
		c.queue.AddRateLimited(key)
		return
	}

	klog.Warningf("Dropping service %q out of the queue: %v", key, err)
	c.queue.Forget(key)
	utilruntime.HandleError(err)
}

func (c *Controller) syncServices(key string) error {
	startTime := time.Now()
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	klog.Infof("Processing sync for service %s on namespace %s ", name, namespace)

	defer func() {
		klog.V(4).Infof("Finished syncing service %s on namespace %s : %v", name, namespace, time.Since(startTime))
	}()

	return nil
}
