package kubeclient

import (
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/discovery"
)

var (
	kubeclient clientset.Interface
	// > 这俩client 有啥区别?
	discoveryClient discovery.DiscoveryInterface
)
