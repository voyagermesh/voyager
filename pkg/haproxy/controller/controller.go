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

package controller

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	cs "voyagermesh.dev/voyager/client/clientset/versioned"
	voyagerinformers "voyagermesh.dev/voyager/client/informers/externalversions"
	"voyagermesh.dev/voyager/pkg/eventer"

	"github.com/pkg/errors"
	atomic_writer "gomodules.xyz/atomic-writer"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"kmodules.xyz/client-go/tools/queue"
)

type Options struct {
	CloudProvider  string
	IngressRef     core.ObjectReference
	ConfigDir      string
	CertDir        string
	CmdFile        string
	QPS            float32
	Burst          int
	MaxNumRequeues int
	NumThreads     int
	ResyncPeriod   time.Duration
}

func (opts Options) UsesEngress() bool {
	return opts.IngressRef.APIVersion == api.SchemeGroupVersion.String()
}

type Controller struct {
	k8sClient              kubernetes.Interface
	VoyagerClient          cs.Interface
	options                Options
	kubeInformerFactory    informers.SharedInformerFactory
	voyagerInformerFactory voyagerinformers.SharedInformerFactory
	recorder               record.EventRecorder

	cfgWriter  *atomic_writer.AtomicWriter
	certWriter *atomic_writer.AtomicWriter

	cfgQueue    *queue.Worker
	cfgInformer cache.SharedIndexInformer

	secretQueue    *queue.Worker
	secretInformer cache.SharedIndexInformer

	ingQueue    *queue.Worker
	ingInformer cache.SharedIndexInformer

	engQueue    *queue.Worker
	engInformer cache.SharedIndexInformer
}

func New(client kubernetes.Interface, voyagerClient cs.Interface, opt Options) *Controller {
	return &Controller{
		k8sClient:              client,
		kubeInformerFactory:    informers.NewSharedInformerFactoryWithOptions(client, opt.ResyncPeriod, informers.WithNamespace(opt.IngressRef.Namespace)),
		VoyagerClient:          voyagerClient,
		voyagerInformerFactory: voyagerinformers.NewSharedInformerFactoryWithOptions(voyagerClient, opt.ResyncPeriod, voyagerinformers.WithNamespace(opt.IngressRef.Namespace)),
		options:                opt,
		recorder:               eventer.NewEventRecorder(client, "haproxy-controller"),
	}
}

func (c *Controller) getIngressWorker() *queue.Worker {
	if c.options.UsesEngress() {
		return c.engQueue
	}
	return c.ingQueue
}

func (c *Controller) Setup() (err error) {
	if c.options.IngressRef.APIVersion == "" {
		err = errors.New("ingress api version not found")
		return
	}
	if c.options.IngressRef.Name == "" {
		err = errors.New("ingress name not found")
		return
	}

	if c.options.IngressRef.APIVersion == api.SchemeGroupVersion.String() {
		c.initIngressCRDWatcher()
	} else {
		c.initIngressWatcher()
	}
	c.initConfigMapWatcher()
	c.initSecretWatcher()
	writerContext := fmt.Sprintf("%s Ingress %v/%v pod %v", c.options.IngressRef.APIVersion, c.options.IngressRef.Namespace, c.options.IngressRef.Name, os.Getenv("HOSTNAME"))
	c.cfgWriter, err = atomic_writer.NewAtomicWriter(strings.TrimSuffix(c.options.ConfigDir, "/"), writerContext)
	if err != nil {
		return
	}
	c.certWriter, err = atomic_writer.NewAtomicWriter(strings.TrimSuffix(c.options.CertDir, "/"), writerContext)
	if err != nil {
		return
	}

	var ing *api.Ingress
	ing, err = c.initIngressIndexer()
	if err != nil {
		return
	}
	err = c.initConfigCache()
	if err != nil {
		return
	}
	err = c.initTLSCache(ing)
	if err != nil {
		return
	}
	err = c.mountIngress(ing)
	return
}

func (c *Controller) initIngressIndexer() (*api.Ingress, error) {
	if c.options.UsesEngress() {
		obj, err := c.VoyagerClient.VoyagerV1beta1().Ingresses(c.options.IngressRef.Namespace).Get(context.TODO(), c.options.IngressRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		obj.Migrate()
		err = obj.IsValid(c.options.CloudProvider)
		if err != nil {
			return nil, err
		}

		return obj, c.engInformer.GetIndexer().Add(obj)
	}

	obj, err := c.k8sClient.NetworkingV1beta1().Ingresses(c.options.IngressRef.Namespace).Get(context.TODO(), c.options.IngressRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	ingress, err := api.NewEngressFromIngress(obj)
	if err != nil {
		return nil, err
	}
	err = ingress.IsValid(c.options.CloudProvider)
	if err != nil {
		return nil, err
	}

	return ingress, c.ingInformer.GetIndexer().Add(obj)
}

func (c *Controller) initConfigCache() error {
	cm, err := c.k8sClient.CoreV1().ConfigMaps(c.options.IngressRef.Namespace).Get(context.TODO(), api.VoyagerPrefix+c.options.IngressRef.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	return c.cfgInformer.GetIndexer().Add(cm)
}

func (c *Controller) initTLSCache(ing *api.Ingress) error {
	for _, tls := range ing.Spec.TLS {
		sc, err := c.k8sClient.CoreV1().Secrets(c.options.IngressRef.Namespace).Get(context.TODO(), tls.Ref.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		err = c.secretInformer.GetIndexer().Add(sc)
		if err != nil {
			return err
		}
	}

	if name := ing.AuthTLSSecret(); name != "" {
		stls, err := c.k8sClient.CoreV1().Secrets(c.options.IngressRef.Namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		err = c.secretInformer.GetIndexer().Add(stls)
		if err != nil {
			return err
		}
	} else {
		for _, fr := range ing.Spec.FrontendRules {
			if fr.Auth != nil && fr.Auth.TLS != nil {
				stls, err := c.k8sClient.CoreV1().Secrets(c.options.IngressRef.Namespace).Get(context.TODO(), fr.Auth.TLS.SecretName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				err = c.secretInformer.GetIndexer().Add(stls)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func certificateToPEMData(crt, key []byte) []byte {
	var buf bytes.Buffer
	buf.Write(bytes.TrimSpace(crt))
	buf.WriteRune('\n')
	buf.Write(bytes.TrimSpace(key))
	return buf.Bytes()
}

func runCmd() error {
	klog.Info("Running haproxy start/reload...")
	if err := startOrReloadHaproxy(); err != nil {
		return err
	}
	incReloadCounter()
	return nil
}

func (c *Controller) Run(stopCh chan struct{}) {
	defer runtime.HandleCrash()

	klog.Info("Starting haproxy-controller")
	c.kubeInformerFactory.Start(stopCh)
	c.voyagerInformerFactory.Start(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	for _, v := range c.kubeInformerFactory.WaitForCacheSync(stopCh) {
		if !v {
			runtime.HandleError(errors.Errorf("timed out waiting for caches to sync"))
			return
		}
	}
	for _, v := range c.voyagerInformerFactory.WaitForCacheSync(stopCh) {
		if !v {
			runtime.HandleError(errors.Errorf("timed out waiting for caches to sync"))
			return
		}
	}

	c.cfgQueue.Run(stopCh)
	c.secretQueue.Run(stopCh)
	c.getIngressWorker().Run(stopCh)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	go func() {
		for range ticker.C {
			startHaproxyIfNeeded()
		}
	}()

	<-stopCh
	klog.Info("Stopping haproxy-controller")
}
