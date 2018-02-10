package ingress

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	v1u "github.com/appscode/kutil/core/v1"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	cs "github.com/appscode/voyager/client"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1"
	vault "github.com/hashicorp/vault/api"
	core "k8s.io/api/core/v1"
	kext_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	core_listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"
)

type Controller interface {
	IsExists() bool
	Reconcile() error
	Delete()
	EnsureFirewall(svc *core.Service) error
}

type Config struct {
	CloudProvider               string
	OperatorNamespace           string
	OperatorService             string
	EnableRBAC                  bool
	DockerRegistry              string
	HAProxyImageTag             string
	ExporterImageTag            string
	QPS                         float32
	Burst                       int
	RestrictToOperatorNamespace bool
	CloudConfigFile             string
}

func (cfg Config) HAProxyImage() string {
	return fmt.Sprintf("%s/haproxy:%s", cfg.DockerRegistry, cfg.HAProxyImageTag)
}

func (cfg Config) ExporterImage() string {
	return fmt.Sprintf("%s/voyager:%s", cfg.DockerRegistry, cfg.ExporterImageTag)
}

type controller struct {
	KubeClient      kubernetes.Interface
	CRDClient       kext_cs.ApiextensionsV1beta1Interface
	VoyagerClient   cs.Interface
	PromClient      pcm.MonitoringV1Interface
	ServiceLister   core_listers.ServiceLister
	EndpointsLister core_listers.EndpointsLister

	recorder record.EventRecorder

	cfg Config

	// Engress object that created or updated.
	Ingress *api.Ingress

	// contains raw configMap data parsed from the cfg file.
	HAProxyConfig string

	logger *log.Logger
	sync.Mutex
}

func NewController(
	ctx context.Context,
	kubeClient kubernetes.Interface,
	crdClient kext_cs.ApiextensionsV1beta1Interface,
	extClient cs.Interface,
	promClient pcm.MonitoringV1Interface,
	serviceLister core_listers.ServiceLister,
	endpointsLister core_listers.EndpointsLister,
	opt Config,
	ingress *api.Ingress) Controller {
	switch ingress.LBType() {
	case api.LBTypeHostPort:
		return NewHostPortController(ctx, kubeClient, crdClient, extClient, promClient, serviceLister, endpointsLister, opt, ingress)
	case api.LBTypeNodePort:
		return NewNodePortController(ctx, kubeClient, crdClient, extClient, promClient, serviceLister, endpointsLister, opt, ingress)
	case api.LBTypeLoadBalancer:
		return NewLoadBalancerController(ctx, kubeClient, crdClient, extClient, promClient, serviceLister, endpointsLister, opt, ingress)
	case api.LBTypeInternal:
		return NewInternalController(ctx, kubeClient, crdClient, extClient, promClient, serviceLister, endpointsLister, opt, ingress)
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

func (c *controller) ensureEnvVars(vars []core.EnvVar) []core.EnvVar {
	if addr := os.Getenv(vault.EnvVaultAddress); addr != "" {
		vars = v1u.UpsertEnvVars(vars, core.EnvVar{
			Name:  vault.EnvVaultAddress,
			Value: addr,
		})
		if caCert := os.Getenv(vault.EnvVaultCACert); caCert != "" {
			vars = v1u.UpsertEnvVars(vars, core.EnvVar{
				Name:  vault.EnvVaultCACert,
				Value: caCert,
			})
		}
		if caPath := os.Getenv(vault.EnvVaultCAPath); caPath != "" {
			caCert, _ := ioutil.ReadFile(caPath)
			vars = v1u.UpsertEnvVars(vars, core.EnvVar{
				Name:  vault.EnvVaultCACert,
				Value: string(caCert),
			})
		}
	}
	return vars
}
