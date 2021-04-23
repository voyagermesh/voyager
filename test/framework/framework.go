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

package framework

import (
	"context"
	"sync"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	cs "voyagermesh.dev/voyager/client/clientset/versioned"
	"voyagermesh.dev/voyager/pkg/operator"

	. "github.com/onsi/gomega"
	"gomodules.xyz/blobfs"
	"gomodules.xyz/cert/certstore"
	"gomodules.xyz/x/crypto/rand"
	core "k8s.io/api/core/v1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	MaxRetry   = 10
	NoRetry    = 1
	TestDomain = "http.appscode.test"
)

type Framework struct {
	KubeClient    kubernetes.Interface
	VoyagerClient cs.Interface
	CRDClient     crd_cs.Interface

	Config        *operator.OperatorConfig
	CertStore     *certstore.CertStore
	TestNamespace string
	Cleanup       bool

	Lock sync.Mutex
}

type Invocation struct {
	*rootInvocation
	Ingress     *ingressInvocation
	Certificate *certificateInvocation
}

type rootInvocation struct {
	*Framework
	app string
}

type ingressInvocation struct {
	*rootInvocation
}

type certificateInvocation struct {
	*rootInvocation
}

func New(cfg *operator.OperatorConfig, testNamespace string, cleanup bool) *Framework {
	cm, err := certstore.New(blobfs.NewInMemoryFS(), "/pki")
	Expect(err).NotTo(HaveOccurred())

	err = cm.InitCA()
	Expect(err).NotTo(HaveOccurred())

	return &Framework{
		KubeClient:    cfg.KubeClient,
		VoyagerClient: cfg.VoyagerClient,
		CRDClient:     cfg.CRDClient,
		Config:        cfg,
		CertStore:     cm,
		TestNamespace: testNamespace,
		Cleanup:       cleanup,
	}
}

func (f *Framework) Invoke() *Invocation {
	r := &rootInvocation{
		Framework: f,
		app:       rand.WithUniqSuffix("voyager-e2e"),
	}
	return &Invocation{
		rootInvocation: r,
		Ingress:        &ingressInvocation{rootInvocation: r},
		Certificate:    &certificateInvocation{rootInvocation: r},
	}
}

func (f *Framework) MinikubeIP() string {
	if f.Config.CloudProvider == api.ProviderMinikube {
		nodes, err := f.Config.KubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())
		for _, addr := range nodes.Items[0].Status.Addresses {
			if addr.Type == core.NodeInternalIP {
				return addr.Address
			}
		}
	}
	panic("MinikubeIP can't be determined when cloud provider is " + f.Config.CloudProvider)
}

func (ri *rootInvocation) App() string {
	return ri.app
}
