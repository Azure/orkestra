package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"time"

	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/aggregator"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/collector"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/yaml"
)

func main() {
	var spec string
	ctx := context.Background()

	flag.StringVar(&spec, "spec", "", "The spec of the helmrelease object to apply")
	flag.Parse()

	if spec == "" {
		os.Exit(1)
	}

	decodedSpec, err := base64.StdEncoding.DecodeString(spec)
	if err != nil {
		fmt.Printf("Failed to decode the string as a base64 string; got the string %v", spec)
		os.Exit(1)
	}

	hr := &fluxhelmv2beta1.HelmRelease{}
	yaml.Unmarshal(decodedSpec, hr)

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		fmt.Printf("Failed to initialize the client config with %v", err)
		os.Exit(1)
	}
	scheme := scheme.Scheme
	if err := fluxhelmv2beta1.AddToScheme(scheme); err != nil {
		fmt.Printf("Failed to add the flux helm scheme to the configuration scheme with %v", err)
		os.Exit(1)
	}
	clientSet, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		fmt.Printf("Failed to create the clientset with the given config with %v", err)
		os.Exit(1)
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: hr.Namespace,
		},
	}

	// Best try at creating the namespace if it doesn't exist
	clientSet.Create(ctx, ns)

	instance := &fluxhelmv2beta1.HelmRelease{}
	key := client.ObjectKey{
		Name:      hr.Name,
		Namespace: hr.Namespace,
	}
	if err := clientSet.Get(ctx, key, instance); client.IgnoreNotFound(err) != nil {
		fmt.Printf("Failed to get instance of the helmrelease with %v", err)
		os.Exit(1)
	} else if err != nil {
		// This means that the object was not found
		if err := clientSet.Create(ctx, hr); err != nil {
			fmt.Printf("Failed to create the helmrelease with %v", err)
			os.Exit(1)
		}
	} else {
		patch := client.MergeFrom(instance)
		if err := clientSet.Patch(ctx, hr, patch); err != nil {
			fmt.Printf("Failed to update the helmrelease with %v", err)
			os.Exit(1)
		}
	}

	identifiers := object.ObjMetadata{
		Namespace: hr.Namespace,
		Name:      hr.Name,
		GroupKind: schema.GroupKind{
			Group: "helm.toolkit.fluxcd.io",
			Kind:  "HelmRelease",
		},
	}

	// We give the poller two minutes before we time it out
	if err := PollStatus(ctx, clientSet, config, time.Minute*2, identifiers); err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}
}

func PollStatus(ctx context.Context, clientSet client.Client, config *rest.Config, timeout time.Duration, identifiers ...object.ObjMetadata) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	restMapper, err := apiutil.NewDynamicRESTMapper(config)
	if err != nil {
		return err
	}
	poller := polling.NewStatusPoller(clientSet, restMapper)
	eventsChan := poller.Poll(ctx, identifiers, polling.Options{PollInterval: time.Second})

	coll := collector.NewResourceStatusCollector(identifiers)
	done := coll.ListenWithObserver(eventsChan, desiredStatusNotifierFunc(cancel))

	<-done

	if coll.Error != nil || ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("timed out waiting for condition")
	}

	return nil
}

func desiredStatusNotifierFunc(cancelFunc context.CancelFunc) collector.ObserverFunc {
	return func(rsc *collector.ResourceStatusCollector, _ event.Event) {
		var rss []*event.ResourceStatus
		for _, rs := range rsc.ResourceStatuses {
			rss = append(rss, rs)
		}
		aggStatus := aggregator.AggregateStatus(rss, status.CurrentStatus)
		if aggStatus == status.CurrentStatus {
			cancelFunc()
		}
	}
}
