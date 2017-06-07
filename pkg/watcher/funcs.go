package watcher

import (
	acs "github.com/appscode/voyager/client/clientset"
	kapi "k8s.io/kubernetes/pkg/api"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
)

func IngressListFunc(c clientset.Interface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Extensions().Ingresses(kapi.NamespaceAll).List(opts)
	}
}

func IngressWatchFunc(c clientset.Interface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Extensions().Ingresses(kapi.NamespaceAll).Watch(options)
	}
}

func ExtendedIngressListFunc(c acs.ExtensionInterface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Ingress(kapi.NamespaceAll).List(opts)
	}
}

func ExtendedIngressWatchFunc(c acs.ExtensionInterface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Ingress(kapi.NamespaceAll).Watch(options)
	}
}

func DaemonSetListFunc(c clientset.Interface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Extensions().DaemonSets(kapi.NamespaceAll).List(opts)
	}
}

func DaemonSetWatchFunc(c clientset.Interface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Extensions().DaemonSets(kapi.NamespaceAll).Watch(options)
	}
}

func CertificateListFunc(c acs.ExtensionInterface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Certificate(kapi.NamespaceAll).List(opts)
	}
}

func CertificateWatchFunc(c acs.ExtensionInterface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Certificate(kapi.NamespaceAll).Watch(options)
	}
}

func PodListFunc(c clientset.Interface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Core().Pods(kapi.NamespaceAll).List(opts)
	}
}

func PodWatchFunc(c clientset.Interface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Core().Pods(kapi.NamespaceAll).Watch(options)
	}
}

func ServiceListFunc(c clientset.Interface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Core().Services(kapi.NamespaceAll).List(opts)
	}
}

func ServiceWatchFunc(c clientset.Interface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Core().Services(kapi.NamespaceAll).Watch(options)
	}
}

func EndpointListFunc(c clientset.Interface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Core().Endpoints(kapi.NamespaceAll).List(opts)
	}
}

func EndpointWatchFunc(c clientset.Interface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Core().Endpoints(kapi.NamespaceAll).Watch(options)
	}
}

func DeploymentListFunc(c clientset.Interface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Extensions().Deployments(kapi.NamespaceAll).List(opts)
	}
}

func DeploymentWatchFunc(c clientset.Interface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Extensions().Deployments(kapi.NamespaceAll).Watch(options)
	}
}

func ConfigMapListFunc(c clientset.Interface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Core().ConfigMaps(kapi.NamespaceAll).List(opts)
	}
}

func ConfigMapWatchFunc(c clientset.Interface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Core().ConfigMaps(kapi.NamespaceAll).Watch(options)
	}
}
