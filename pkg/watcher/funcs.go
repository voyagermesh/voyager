package watcher

import (
	acs "github.com/appscode/voyager/client/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func IngressListFunc(c clientset.Interface) func(metav1.ListOptions) (runtime.Object, error) {
	return func(opts metav1.ListOptions) (runtime.Object, error) {
		return c.ExtensionsV1beta1().Ingresses(apiv1.NamespaceAll).List(opts)
	}
}

func IngressWatchFunc(c clientset.Interface) func(options metav1.ListOptions) (watch.Interface, error) {
	return func(options metav1.ListOptions) (watch.Interface, error) {
		return c.ExtensionsV1beta1().Ingresses(apiv1.NamespaceAll).Watch(options)
	}
}

func ExtendedIngressListFunc(c acs.ExtensionInterface) func(metav1.ListOptions) (runtime.Object, error) {
	return func(opts metav1.ListOptions) (runtime.Object, error) {
		return c.Ingress(apiv1.NamespaceAll).List(opts)
	}
}

func ExtendedIngressWatchFunc(c acs.ExtensionInterface) func(options metav1.ListOptions) (watch.Interface, error) {
	return func(options metav1.ListOptions) (watch.Interface, error) {
		return c.Ingress(apiv1.NamespaceAll).Watch(options)
	}
}

func DaemonSetListFunc(c clientset.Interface) func(metav1.ListOptions) (runtime.Object, error) {
	return func(opts metav1.ListOptions) (runtime.Object, error) {
		return c.ExtensionsV1beta1().DaemonSets(apiv1.NamespaceAll).List(opts)
	}
}

func DaemonSetWatchFunc(c clientset.Interface) func(options metav1.ListOptions) (watch.Interface, error) {
	return func(options metav1.ListOptions) (watch.Interface, error) {
		return c.ExtensionsV1beta1().DaemonSets(apiv1.NamespaceAll).Watch(options)
	}
}

func CertificateListFunc(c acs.ExtensionInterface) func(metav1.ListOptions) (runtime.Object, error) {
	return func(opts metav1.ListOptions) (runtime.Object, error) {
		return c.Certificate(apiv1.NamespaceAll).List(opts)
	}
}

func CertificateWatchFunc(c acs.ExtensionInterface) func(options metav1.ListOptions) (watch.Interface, error) {
	return func(options metav1.ListOptions) (watch.Interface, error) {
		return c.Certificate(apiv1.NamespaceAll).Watch(options)
	}
}

func PodListFunc(c clientset.Interface) func(metav1.ListOptions) (runtime.Object, error) {
	return func(opts metav1.ListOptions) (runtime.Object, error) {
		return c.CoreV1().Pods(apiv1.NamespaceAll).List(opts)
	}
}

func PodWatchFunc(c clientset.Interface) func(options metav1.ListOptions) (watch.Interface, error) {
	return func(options metav1.ListOptions) (watch.Interface, error) {
		return c.CoreV1().Pods(apiv1.NamespaceAll).Watch(options)
	}
}

func ServiceListFunc(c clientset.Interface) func(metav1.ListOptions) (runtime.Object, error) {
	return func(opts metav1.ListOptions) (runtime.Object, error) {
		return c.CoreV1().Services(apiv1.NamespaceAll).List(opts)
	}
}

func ServiceWatchFunc(c clientset.Interface) func(options metav1.ListOptions) (watch.Interface, error) {
	return func(options metav1.ListOptions) (watch.Interface, error) {
		return c.CoreV1().Services(apiv1.NamespaceAll).Watch(options)
	}
}

func EndpointListFunc(c clientset.Interface) func(metav1.ListOptions) (runtime.Object, error) {
	return func(opts metav1.ListOptions) (runtime.Object, error) {
		return c.CoreV1().Endpoints(apiv1.NamespaceAll).List(opts)
	}
}

func EndpointWatchFunc(c clientset.Interface) func(options metav1.ListOptions) (watch.Interface, error) {
	return func(options metav1.ListOptions) (watch.Interface, error) {
		return c.CoreV1().Endpoints(apiv1.NamespaceAll).Watch(options)
	}
}

func DeploymentListFunc(c clientset.Interface) func(metav1.ListOptions) (runtime.Object, error) {
	return func(opts metav1.ListOptions) (runtime.Object, error) {
		return c.ExtensionsV1beta1().Deployments(apiv1.NamespaceAll).List(opts)
	}
}

func DeploymentWatchFunc(c clientset.Interface) func(options metav1.ListOptions) (watch.Interface, error) {
	return func(options metav1.ListOptions) (watch.Interface, error) {
		return c.ExtensionsV1beta1().Deployments(apiv1.NamespaceAll).Watch(options)
	}
}

func ConfigMapListFunc(c clientset.Interface) func(metav1.ListOptions) (runtime.Object, error) {
	return func(opts metav1.ListOptions) (runtime.Object, error) {
		return c.CoreV1().ConfigMaps(apiv1.NamespaceAll).List(opts)
	}
}

func ConfigMapWatchFunc(c clientset.Interface) func(options metav1.ListOptions) (watch.Interface, error) {
	return func(options metav1.ListOptions) (watch.Interface, error) {
		return c.CoreV1().ConfigMaps(apiv1.NamespaceAll).Watch(options)
	}
}
