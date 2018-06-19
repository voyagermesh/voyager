package ingress

import (
	"context"
	"io/ioutil"
	"os"
	"sync"

	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	wpi "github.com/appscode/kubernetes-webhook-util/apis/workload/v1"
	wcs "github.com/appscode/kubernetes-webhook-util/client/workload/v1"
	core_util "github.com/appscode/kutil/core/v1"
	v1u "github.com/appscode/kutil/core/v1"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	cs "github.com/appscode/voyager/client/clientset/versioned"
	"github.com/appscode/voyager/pkg/config"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1"
	vault "github.com/hashicorp/vault/api"
	core "k8s.io/api/core/v1"
	kext_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
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

type controller struct {
	KubeClient      kubernetes.Interface
	WorkloadClient  wcs.Interface
	CRDClient       kext_cs.ApiextensionsV1beta1Interface
	VoyagerClient   cs.Interface
	PromClient      pcm.MonitoringV1Interface
	ServiceLister   core_listers.ServiceLister
	EndpointsLister core_listers.EndpointsLister

	recorder record.EventRecorder

	cfg config.Config

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
	workloadClient wcs.Interface,
	crdClient kext_cs.ApiextensionsV1beta1Interface,
	extClient cs.Interface,
	promClient pcm.MonitoringV1Interface,
	serviceLister core_listers.ServiceLister,
	endpointsLister core_listers.EndpointsLister,
	cfg config.Config,
	ingress *api.Ingress,
	recorder record.EventRecorder) Controller {
	switch ingress.LBType() {
	case api.LBTypeHostPort:
		return NewHostPortController(ctx, kubeClient, workloadClient, crdClient, extClient, promClient, serviceLister, endpointsLister, cfg, ingress, recorder)
	case api.LBTypeNodePort:
		return NewNodePortController(ctx, kubeClient, workloadClient, crdClient, extClient, promClient, serviceLister, endpointsLister, cfg, ingress, recorder)
	case api.LBTypeLoadBalancer:
		return NewLoadBalancerController(ctx, kubeClient, workloadClient, crdClient, extClient, promClient, serviceLister, endpointsLister, cfg, ingress, recorder)
	case api.LBTypeInternal:
		return NewInternalController(ctx, kubeClient, workloadClient, crdClient, extClient, promClient, serviceLister, endpointsLister, cfg, ingress, recorder)
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

func (c *controller) IsExists() bool {
	wk := c.Ingress.WorkloadKind()
	if wk == wpi.KindDeployment {
		_, err := c.KubeClient.AppsV1beta1().Deployments(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return false
		}
	} else if wk == wpi.KindDaemonSet {
		_, err := c.KubeClient.ExtensionsV1beta1().DaemonSets(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return false
		}
	}

	_, err := c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return false
	}
	_, err = c.KubeClient.CoreV1().ConfigMaps(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return false
	}
	if c.cfg.EnableRBAC {
		_, err = c.KubeClient.CoreV1().ServiceAccounts(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return false
		}
		_, err = c.KubeClient.RbacV1beta1().Roles(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return false
		}
		_, err = c.KubeClient.RbacV1beta1().RoleBindings(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return false
		}
	}
	return true
}

func (c *controller) deletePods() error {
	policy := metav1.DeletePropagationForeground
	options := &metav1.DeleteOptions{
		PropagationPolicy: &policy,
	}
	obj, err := wcs.NewObject(c.Ingress.WorkloadKind(), c.Ingress.OffshootName(), c.Ingress.Namespace)
	if err != nil {
		return err
	}
	err = c.WorkloadClient.Workloads(c.Ingress.Namespace).Delete(obj, options)
	if err != nil {
		return err
	}
	return core_util.RestartPods(c.KubeClient, c.Ingress.Namespace, &metav1.LabelSelector{MatchLabels: c.Ingress.OffshootSelector()})
}
