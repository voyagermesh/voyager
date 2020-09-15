/*
Copyright The Voyager Authors.

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

package operator

import (
	cs "voyagermesh.dev/voyager/client/clientset/versioned"
	voyagerinformers "voyagermesh.dev/voyager/client/informers/externalversions"
	"voyagermesh.dev/voyager/pkg/config"
	"voyagermesh.dev/voyager/pkg/eventer"

	prom "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/typed/monitoring/v1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	reg_util "kmodules.xyz/client-go/admissionregistration/v1beta1"
	"kmodules.xyz/client-go/discovery"
	hooks "kmodules.xyz/webhook-runtime/admission/v1beta1"
	wcs "kmodules.xyz/webhook-runtime/client/workload/v1"
)

const (
	validatingWebhookConfig = "admission.voyager.appscode.com"
)

type OperatorConfig struct {
	config.Config

	ClientConfig   *rest.Config
	KubeClient     kubernetes.Interface
	WorkloadClient wcs.Interface
	CRDClient      crd_cs.Interface
	VoyagerClient  cs.Interface
	PromClient     prom.MonitoringV1Interface
	AdmissionHooks []hooks.AdmissionHook
}

func NewOperatorConfig(clientConfig *rest.Config) *OperatorConfig {
	return &OperatorConfig{
		ClientConfig: clientConfig,
	}
}

func (c *OperatorConfig) New() (*Operator, error) {
	if err := discovery.IsDefaultSupportedVersion(c.KubeClient); err != nil {
		return nil, err
	}

	op := &Operator{
		Config:                 c.Config,
		ClientConfig:           c.ClientConfig,
		KubeClient:             c.KubeClient,
		WorkloadClient:         c.WorkloadClient,
		kubeInformerFactory:    informers.NewFilteredSharedInformerFactory(c.KubeClient, c.ResyncPeriod, c.WatchNamespace, nil),
		CRDClient:              c.CRDClient,
		VoyagerClient:          c.VoyagerClient,
		voyagerInformerFactory: voyagerinformers.NewFilteredSharedInformerFactory(c.VoyagerClient, c.ResyncPeriod, c.WatchNamespace, nil),
		PromClient:             c.PromClient,
		recorder:               eventer.NewEventRecorder(c.KubeClient, "voyager-operator"),
	}

	if err := op.ensureCustomResourceDefinitions(); err != nil {
		return nil, err
	}

	if c.EnableValidatingWebhook {
		if err := reg_util.UpdateValidatingWebhookCABundle(c.ClientConfig, validatingWebhookConfig); err != nil {
			return nil, err
		}
	}

	op.initIngressCRDWatcher()
	op.initIngressWatcher()
	op.initDeploymentWatcher()
	op.initStatefulSetWatcher()
	op.initDaemonSetWatcher()
	op.initServiceWatcher()
	op.initConfigMapWatcher()
	op.initEndpointWatcher()
	op.initSecretWatcher()
	op.initNodeWatcher()
	op.initServiceMonitorWatcher()
	op.initNamespaceWatcher()
	op.initCertificateCRDWatcher()

	return op, nil
}
