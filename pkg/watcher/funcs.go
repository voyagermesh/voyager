package watcher

import (
	acs "github.com/appscode/voyager/client/clientset"
apiv1 "k8s.io/client-go/pkg/api/v1"
clientset "k8s.io/client-go/kubernetes"
"k8s.io/apimachinery/pkg/runtime"
"k8s.io/apimachinery/pkg/watch"
)

func IngressListFunc(c clientset.Interface) func(apiv1.ListOptions) (runtime.Object, error) {
	return func(opts apiv1.ListOptions) (runtime.Object, error) {
		return c.Extensions().Ingresses(apiv1.NamespaceAll).List(opts)
	}
}

func IngressWatchFunc(c clientset.Interface) func(options apiv1.ListOptions) (watch.Interface, error) {
	return func(options apiv1.ListOptions) (watch.Interface, error) {
		return c.Extensions().Ingresses(apiv1.NamespaceAll).Watch(options)
	}
}

func ExtendedIngressListFunc(c acs.ExtensionInterface) func(apiv1.ListOptions) (runtime.Object, error) {
	return func(opts apiv1.ListOptions) (runtime.Object, error) {
		return c.Ingress(apiv1.NamespaceAll).List(opts)
	}
}

func ExtendedIngressWatchFunc(c acs.ExtensionInterface) func(options apiv1.ListOptions) (watch.Interface, error) {
	return func(options apiv1.ListOptions) (watch.Interface, error) {
		return c.Ingress(apiv1.NamespaceAll).Watch(options)
	}
}

func DaemonSetListFunc(c clientset.Interface) func(apiv1.ListOptions) (runtime.Object, error) {
	return func(opts apiv1.ListOptions) (runtime.Object, error) {
		return c.Extensions().DaemonSets(apiv1.NamespaceAll).List(opts)
	}
}

func DaemonSetWatchFunc(c clientset.Interface) func(options apiv1.ListOptions) (watch.Interface, error) {
	return func(options apiv1.ListOptions) (watch.Interface, error) {
		return c.Extensions().DaemonSets(apiv1.NamespaceAll).Watch(options)
	}
}

func CertificateListFunc(c acs.ExtensionInterface) func(apiv1.ListOptions) (runtime.Object, error) {
	return func(opts apiv1.ListOptions) (runtime.Object, error) {
		return c.Certificate(apiv1.NamespaceAll).List(opts)
	}
}

func CertificateWatchFunc(c acs.ExtensionInterface) func(options apiv1.ListOptions) (watch.Interface, error) {
	return func(options apiv1.ListOptions) (watch.Interface, error) {
		return c.Certificate(apiv1.NamespaceAll).Watch(options)
	}
}

func PodListFunc(c clientset.Interface) func(apiv1.ListOptions) (runtime.Object, error) {
	return func(opts apiv1.ListOptions) (runtime.Object, error) {
		return c.Core().Pods(apiv1.NamespaceAll).List(opts)
	}
}

func PodWatchFunc(c clientset.Interface) func(options apiv1.ListOptions) (watch.Interface, error) {
	return func(options apiv1.ListOptions) (watch.Interface, error) {
		return c.Core().Pods(apiv1.NamespaceAll).Watch(options)
	}
}

func ServiceListFunc(c clientset.Interface) func(apiv1.ListOptions) (runtime.Object, error) {
	return func(opts apiv1.ListOptions) (runtime.Object, error) {
		return c.Core().Services(apiv1.NamespaceAll).List(opts)
	}
}

func ServiceWatchFunc(c clientset.Interface) func(options apiv1.ListOptions) (watch.Interface, error) {
	return func(options apiv1.ListOptions) (watch.Interface, error) {
		return c.Core().Services(apiv1.NamespaceAll).Watch(options)
	}
}

func EndpointListFunc(c clientset.Interface) func(apiv1.ListOptions) (runtime.Object, error) {
	return func(opts apiv1.ListOptions) (runtime.Object, error) {
		return c.Core().Endpoints(apiv1.NamespaceAll).List(opts)
	}
}

func EndpointWatchFunc(c clientset.Interface) func(options apiv1.ListOptions) (watch.Interface, error) {
	return func(options apiv1.ListOptions) (watch.Interface, error) {
		return c.Core().Endpoints(apiv1.NamespaceAll).Watch(options)
	}
}

func DeploymentListFunc(c clientset.Interface) func(apiv1.ListOptions) (runtime.Object, error) {
	return func(opts apiv1.ListOptions) (runtime.Object, error) {
		return c.Extensions().Deployments(apiv1.NamespaceAll).List(opts)
	}
}

func DeploymentWatchFunc(c clientset.Interface) func(options apiv1.ListOptions) (watch.Interface, error) {
	return func(options apiv1.ListOptions) (watch.Interface, error) {
		return c.Extensions().Deployments(apiv1.NamespaceAll).Watch(options)
	}
}

func ConfigMapListFunc(c clientset.Interface) func(apiv1.ListOptions) (runtime.Object, error) {
	return func(opts apiv1.ListOptions) (runtime.Object, error) {
		return c.Core().ConfigMaps(apiv1.NamespaceAll).List(opts)
	}
}

func ConfigMapWatchFunc(c clientset.Interface) func(options apiv1.ListOptions) (watch.Interface, error) {
	return func(options apiv1.ListOptions) (watch.Interface, error) {
		return c.Core().ConfigMaps(apiv1.NamespaceAll).Watch(options)
	}
}
