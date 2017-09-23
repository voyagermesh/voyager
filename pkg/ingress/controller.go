package ingress

import (
	"sync"

	"github.com/appscode/go/types"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	acs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/config"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	core "k8s.io/client-go/listers/core/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/record"
)

type Controller interface {
	IsExists() bool
	Create() error
	Update(mode UpdateMode, old *api.Ingress) error
	Delete()
	EnsureFirewall(svc *apiv1.Service) error
}

type controller struct {
	KubeClient      clientset.Interface
	CRDClient       apiextensionsclient.Interface
	ExtClient       acs.VoyagerV1beta1Interface
	PromClient      pcm.MonitoringV1alpha1Interface
	ServiceLister   core.ServiceLister
	EndpointsLister core.EndpointsLister

	recorder record.EventRecorder

	Opt config.Options

	// Engress object that created or updated.
	Ingress *api.Ingress

	// contains raw configMap data parsed from the cfg file.
	HAProxyConfig string

	sync.Mutex
}

func NewController(
	kubeClient clientset.Interface,
	crdClient apiextensionsclient.Interface,
	extClient acs.VoyagerV1beta1Interface,
	promClient pcm.MonitoringV1alpha1Interface,
	serviceLister core.ServiceLister,
	endpointsLister core.EndpointsLister,
	opt config.Options,
	ingress *api.Ingress) Controller {
	switch ingress.LBType() {
	case api.LBTypeHostPort:
		return NewHostPortController(kubeClient, crdClient, extClient, promClient, serviceLister, endpointsLister, opt, ingress)
	case api.LBTypeNodePort:
		return NewNodePortController(kubeClient, crdClient, extClient, promClient, serviceLister, endpointsLister, opt, ingress)
	case api.LBTypeLoadBalancer:
		return NewLoadBalancerController(kubeClient, crdClient, extClient, promClient, serviceLister, endpointsLister, opt, ingress)
	}
	return nil
}

func (c *controller) ensureOwnerReference(in metav1.ObjectMeta) metav1.ObjectMeta {
	fi := -1
	for i, ref := range in.OwnerReferences {
		if ref.Kind == "Ingress" && ref.Name == c.Ingress.Name {
			fi = i
			break
		}
	}
	if fi == -1 {
		in.OwnerReferences = append(in.OwnerReferences, metav1.OwnerReference{})
		fi = len(in.OwnerReferences) - 1
	}
	in.OwnerReferences[fi].APIVersion = c.Ingress.APISchema()
	in.OwnerReferences[fi].Kind = "Ingress"
	in.OwnerReferences[fi].Name = c.Ingress.Name
	in.OwnerReferences[fi].UID = c.Ingress.UID
	in.OwnerReferences[fi].BlockOwnerDeletion = types.TrueP()
	return in
}
