package operator

import (
	"bytes"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/appscode/log"
	"github.com/appscode/voyager/apis/voyager/v1beta1"
	voyagerv1beta1 "github.com/appscode/voyager/apis/voyager/v1beta1"
	acs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	"github.com/appscode/voyager/cmd/ssl-mounter/app/volume"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extv1beta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

type secretMounter struct {
	ing           *v1beta1.Ingress
	mountLocation string
	cmdFile       string

	KubeClient    clientset.Interface
	VoyagerClient acs.VoyagerV1beta1Interface
	resyncPeriod  time.Duration

	lock            sync.Mutex
	fileProjections map[string]volume.FileProjection
	writer          *volume.AtomicWriter
}

func NewIngressSecretMounter(client clientset.Interface, vclient acs.VoyagerV1beta1Interface, ing *v1beta1.Ingress, mountDir, cmd string, resyncPeriod time.Duration) (*secretMounter, error) {
	payloads := make(map[string]volume.FileProjection)
	for _, secret := range ing.Secrets() {
		sc, err := client.CoreV1().Secrets(ing.Namespace).Get(secret, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		payloads[secret+".pem"] = volume.FileProjection{Mode: 0777, Data: secretToPEMData(sc)}
	}

	location := strings.TrimSuffix(mountDir, "/")
	writer, err := volume.NewAtomicWriter(location)
	if err != nil {
		return nil, err
	}

	return &secretMounter{
		mountLocation:   location,
		cmdFile:         cmd,
		resyncPeriod:    resyncPeriod,
		KubeClient:      client,
		VoyagerClient:   vclient,
		ing:             ing,
		fileProjections: payloads,
		writer:          writer,
	}, nil
}

func (c *secretMounter) Run(stopCh <-chan struct{}) {
	c.initSecretInformer(stopCh)
	c.initIngressInformer(stopCh)
}

func (c *secretMounter) initSecretInformer(stopCh <-chan struct{}) {
	secretInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
				return c.KubeClient.CoreV1().Secrets(c.ing.Namespace).List(metav1.ListOptions{})
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return c.KubeClient.CoreV1().Secrets(c.ing.Namespace).Watch(metav1.ListOptions{})
			},
		},
		&apiv1.Secret{},
		c.resyncPeriod,
		cache.Indexers{},
	)

	secretInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if secret, ok := obj.(*apiv1.Secret); ok {
				if c.isSecretUsedInIngress(secret) {
					c.lock.Lock()
					defer c.lock.Unlock()

					c.fileProjections[secret.Name+".pem"] = volume.FileProjection{Mode: 0777, Data: secretToPEMData(secret)}
					c.MustMount()
				}
			}
		},
		UpdateFunc: func(old, new interface{}) {
			if oldSecret, oldOK := old.(*apiv1.Secret); oldOK {
				if newSecret, newOK := new.(*apiv1.Secret); newOK {
					if !reflect.DeepEqual(oldSecret.Data, newSecret.Data) {
						if c.isSecretUsedInIngress(newSecret) {
							c.lock.Lock()
							defer c.lock.Unlock()

							c.fileProjections[newSecret.Name+".pem"] = volume.FileProjection{Mode: 0777, Data: secretToPEMData(newSecret)}
							c.MustMount()
						}
					}
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			if secret, ok := obj.(*apiv1.Secret); ok {
				if c.isSecretUsedInIngress(secret) {
					c.lock.Lock()
					defer c.lock.Unlock()

					delete(c.fileProjections, secret.Name+".pem")
					c.MustMount()
				}
			}
		},
	})
	go secretInformer.Run(stopCh)
}

func (c *secretMounter) initIngressInformer(stopCh <-chan struct{}) {
	var ingressInformer cache.SharedIndexInformer
	switch c.ing.APIVersion {
	case voyagerv1beta1.SchemeGroupVersion.String():
		ingressInformer = cache.NewSharedIndexInformer(
			&cache.ListWatch{
				ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
					return c.VoyagerClient.Ingresses(c.ing.Namespace).List(metav1.ListOptions{
						FieldSelector: fields.OneTermEqualSelector("metadata.name", c.ing.Name).String(),
					})
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return c.VoyagerClient.Ingresses(c.ing.Namespace).Watch(metav1.ListOptions{
						FieldSelector: fields.OneTermEqualSelector("metadata.name", c.ing.Name).String(),
					})
				},
			},
			&v1beta1.Ingress{},
			c.resyncPeriod,
			cache.Indexers{},
		)
	case "extensions/v1beta1":
		ingressInformer = cache.NewSharedIndexInformer(
			&cache.ListWatch{
				ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
					return c.KubeClient.ExtensionsV1beta1().Ingresses(c.ing.Namespace).List(metav1.ListOptions{
						FieldSelector: fields.OneTermEqualSelector("metadata.name", c.ing.Name).String(),
					})
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return c.KubeClient.ExtensionsV1beta1().Ingresses(c.ing.Namespace).Watch(metav1.ListOptions{
						FieldSelector: fields.OneTermEqualSelector("metadata.name", c.ing.Name).String(),
					})
				},
			},
			&extv1beta1.Ingress{},
			c.resyncPeriod,
			cache.Indexers{},
		)
	}

	ingressInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, new interface{}) {
			var newIngress *v1beta1.Ingress
			ok := false
			switch c.ing.APIVersion {
			case voyagerv1beta1.SchemeGroupVersion.String():
				if _, oldOK := old.(*v1beta1.Ingress); oldOK {
					if newIng, newOK := new.(*v1beta1.Ingress); newOK {
						newIngress = newIng
						ok = true
					}
				}
			case "extensions/v1beta1":
				if _, oldOK := old.(*extv1beta1.Ingress); oldOK {
					if newIng, newOK := new.(*extv1beta1.Ingress); newOK {
						newExtIng, err := v1beta1.NewEngressFromIngress(newIng)
						if err != nil {
							log.Errorln("Failed to convert ingress", err)
							return
						}
						newIngress = newExtIng
						ok = true
					}
				}
			}

			if ok {
				secretsUsedMaps := make(map[string]struct{})

				c.lock.Lock()
				c.lock.Unlock()

				for _, secret := range newIngress.Secrets() {
					if _, ok := c.fileProjections[secret+".pem"]; !ok {
						sc, err := c.KubeClient.CoreV1().Secrets(c.ing.Namespace).Get(secret, metav1.GetOptions{})
						if err != nil {
							log.Fatalln(err)
						}
						c.fileProjections[secret+".pem"] = volume.FileProjection{Mode: 0777, Data: secretToPEMData(sc)}
					}
					secretsUsedMaps[secret+".pem"] = struct{}{}
				}

				for k := range c.fileProjections {
					if _, ok := secretsUsedMaps[k+".pem"]; !ok {
						delete(c.fileProjections, k)
					}
				}

				c.MustMount()

				// Update Ingress
				c.ing = newIngress
			}
		},
	})
	go ingressInformer.Run(stopCh)
}

func (c *secretMounter) isSecretUsedInIngress(s *apiv1.Secret) bool {
	if s.Namespace != c.ing.Namespace {
		return false
	}

	for _, secret := range c.ing.Secrets() {
		if s.Name == secret {
			return true
		}
	}
	return false
}

func (c *secretMounter) Mount() error {
	err := c.writer.Write(c.fileProjections)
	if err != nil {
		return err
	}
	return nil
}

func (c *secretMounter) MustMount() {
	err := c.Mount()
	if err != nil {
		log.Fatalln(err)
	}
}

func secretToPEMData(s *apiv1.Secret) []byte {
	pemdata := bytes.NewBuffer(s.Data[apiv1.TLSCertKey])
	pemdata.Write([]byte("\n"))
	pemdata.Write(s.Data[apiv1.TLSPrivateKeyKey])
	return pemdata.Bytes()
}
