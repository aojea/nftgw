package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aojea/nftgw/pkg/gateway"
	"github.com/aojea/nftgw/pkg/services"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	gwclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
	gwinformers "sigs.k8s.io/gateway-api/pkg/client/informers/externalversions"
)

func main() {
	var kubeconfig string
	var master string

	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	flag.StringVar(&master, "master", "", "master url")
	flag.Parse()

	// trap Ctrl+C and call cancel on the context
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	// Enable signal handler
	signalCh := make(chan os.Signal, 2)
	defer func() {
		close(signalCh)
		cancel()
	}()

	signal.Notify(signalCh, os.Interrupt, syscall.SIGINT)
	go func() {
		select {
		case <-signalCh:
			log.Printf("Exiting: received signal")
			cancel()
		case <-ctx.Done():
		}
	}()

	// creates the connection
	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		klog.Fatal(err)
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}

	noProxyName, err := labels.NewRequirement("service.kubernetes.io/service-proxy-name", selection.DoesNotExist, nil)
	if err != nil {
		klog.Fatal(err)
	}

	noHeadlessEndpoints, err := labels.NewRequirement(v1.IsHeadlessService, selection.DoesNotExist, nil)
	if err != nil {
		klog.Fatal(err)
	}

	labelSelector := labels.NewSelector()
	labelSelector = labelSelector.Add(*noProxyName, *noHeadlessEndpoints)

	informersFactory := informers.NewSharedInformerFactoryWithOptions(clientset, 0,
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.LabelSelector = labelSelector.String()
		}))
	svcController := services.NewController(
		clientset,
		informersFactory.Core().V1().Services(),
		informersFactory.Discovery().V1().EndpointSlices(),
	)

	informersFactory.Start(ctx.Done())
	go svcController.Run(5, ctx.Done())

	// creates the clientset
	gwclientset, err := gwclient.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}

	gwInformersFactory := gwinformers.NewSharedInformerFactoryWithOptions(gwclientset, 0)

	gwController := gateway.NewController(
		clientset,
		gwclientset,
		gwInformersFactory,
	)

	gwInformersFactory.Start(ctx.Done())
	go gwController.Run(5, ctx.Done())

	// Wait forever
	<-ctx.Done()
}
