/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ingress

import (
	"context"
	"io/ioutil"
	"os"
	"sync"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	cs "voyagermesh.dev/voyager/client/clientset/versioned"
	"voyagermesh.dev/voyager/pkg/config"
	_ "voyagermesh.dev/voyager/third_party/forked/cloudprovider/providers"

	"github.com/go-logr/logr"
	vault "github.com/hashicorp/vault/api"
	pcm "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/typed/monitoring/v1"
	core "k8s.io/api/core/v1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	core_listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"
	core_util "kmodules.xyz/client-go/core/v1"
	v1u "kmodules.xyz/client-go/core/v1"
	wpi "kmodules.xyz/webhook-runtime/apis/workload/v1"
	wcs "kmodules.xyz/webhook-runtime/client/workload/v1"
)

type Controller interface {
	IsExists() bool
	Reconcile() error
	Delete()
	FirewallSupported() bool
	EnsureFirewall(svc *core.Service) error
}

type controller struct {
	KubeClient      kubernetes.Interface
	WorkloadClient  wcs.Interface
	CRDClient       crd_cs.Interface
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

	logger logr.Logger
	sync.Mutex
}

func NewController(
	ctx context.Context,
	kubeClient kubernetes.Interface,
	workloadClient wcs.Interface,
	crdClient crd_cs.Interface,
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
		_, err := c.KubeClient.AppsV1().Deployments(c.Ingress.Namespace).Get(context.TODO(), c.Ingress.OffshootName(), metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return false
		}
	} else if wk == wpi.KindStatefulSet {
		_, err := c.KubeClient.AppsV1().StatefulSets(c.Ingress.Namespace).Get(context.TODO(), c.Ingress.OffshootName(), metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return false
		}
	} else if wk == wpi.KindDaemonSet {
		_, err := c.KubeClient.AppsV1().DaemonSets(c.Ingress.Namespace).Get(context.TODO(), c.Ingress.OffshootName(), metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return false
		}
	}

	_, err := c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Get(context.TODO(), c.Ingress.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return false
	}
	_, err = c.KubeClient.CoreV1().ConfigMaps(c.Ingress.Namespace).Get(context.TODO(), c.Ingress.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return false
	}
	_, err = c.KubeClient.CoreV1().ServiceAccounts(c.Ingress.Namespace).Get(context.TODO(), c.Ingress.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return false
	}
	_, err = c.KubeClient.RbacV1().Roles(c.Ingress.Namespace).Get(context.TODO(), c.Ingress.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return false
	}
	_, err = c.KubeClient.RbacV1().RoleBindings(c.Ingress.Namespace).Get(context.TODO(), c.Ingress.OffshootName(), metav1.GetOptions{})

	return !kerr.IsNotFound(err)
}

func (c *controller) deletePods() error {
	policy := metav1.DeletePropagationForeground
	options := metav1.DeleteOptions{
		PropagationPolicy: &policy,
	}
	obj, err := wcs.NewObject(c.Ingress.WorkloadKind(), c.Ingress.OffshootName(), c.Ingress.Namespace)
	if err != nil {
		return err
	}
	err = c.WorkloadClient.Workloads(c.Ingress.Namespace).Delete(context.TODO(), obj, options)
	if err != nil {
		return err
	}
	return core_util.RestartPods(context.TODO(), c.KubeClient, c.Ingress.Namespace, &metav1.LabelSelector{MatchLabels: c.Ingress.OffshootSelector()})
}

func (c *controller) isHPAControlled() bool {
	list, err := c.KubeClient.AutoscalingV1().HorizontalPodAutoscalers(c.Ingress.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		// in case an error happen when getting hpa, deciding to update replicas
		return false
	}
	for _, hpa := range list.Items {
		if hpa.Spec.ScaleTargetRef.Kind == c.Ingress.WorkloadKind() && hpa.Spec.ScaleTargetRef.Name == c.Ingress.OffshootName() {
			return true
		}
	}
	return false
}
