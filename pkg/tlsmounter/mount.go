package tlsmounter

import (
	"reflect"

	"github.com/appscode/go/hold"
	"github.com/appscode/go/ioutil"
	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extv1beta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

// wait.NeverStop
func (c *Controller) Run2(stopCh <-chan struct{}) {
	c.initSecretInformer(stopCh)
	c.initIngressInformer(stopCh)
	c.initCertificateInformer(stopCh)
	hold.Hold()
}

func (c *Controller) initSecretInformer(stopCh <-chan struct{}) {
	secretInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (rt.Object, error) {
				return c.k8sClient.CoreV1().Secrets(c.Ingress.Namespace).List(metav1.ListOptions{})
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return c.k8sClient.CoreV1().Secrets(c.Ingress.Namespace).Watch(metav1.ListOptions{})
			},
		},
		&apiv1.Secret{},
		c.options.ResyncPeriod,
		cache.Indexers{},
	)

	secretInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if secret, ok := obj.(*apiv1.Secret); ok {
				if c.isSecretUsedInIngress(secret) {
					c.lock.Lock()
					defer c.lock.Unlock()

					c.projections[secret.Name+".pem"] = ioutil.FileProjection{Mode: 0777, Data: secretToPEMData(secret)}
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

							c.projections[newSecret.Name+".pem"] = ioutil.FileProjection{Mode: 0777, Data: secretToPEMData(newSecret)}
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

					delete(c.projections, secret.Name+".pem")
					c.MustMount()
				}
			}
		},
	})
	go secretInformer.Run(stopCh)
}

func (c *Controller) initIngressInformer(stopCh <-chan struct{}) {
	var ingressInformer cache.SharedIndexInformer
	switch c.Ingress.APIVersion {
	case api.SchemeGroupVersion.String():
		ingressInformer = cache.NewSharedIndexInformer(
			&cache.ListWatch{
				ListFunc: func(opts metav1.ListOptions) (rt.Object, error) {
					return c.VoyagerClient.Ingresses(c.Ingress.Namespace).List(metav1.ListOptions{
						FieldSelector: fields.OneTermEqualSelector("metadata.name", c.Ingress.Name).String(),
					})
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return c.VoyagerClient.Ingresses(c.Ingress.Namespace).Watch(metav1.ListOptions{
						FieldSelector: fields.OneTermEqualSelector("metadata.name", c.Ingress.Name).String(),
					})
				},
			},
			&api.Ingress{},
			c.options.ResyncPeriod,
			cache.Indexers{},
		)
	case "extensions/v1beta1":
		ingressInformer = cache.NewSharedIndexInformer(
			&cache.ListWatch{
				ListFunc: func(opts metav1.ListOptions) (rt.Object, error) {
					return c.k8sClient.ExtensionsV1beta1().Ingresses(c.Ingress.Namespace).List(metav1.ListOptions{
						FieldSelector: fields.OneTermEqualSelector("metadata.name", c.Ingress.Name).String(),
					})
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return c.k8sClient.ExtensionsV1beta1().Ingresses(c.Ingress.Namespace).Watch(metav1.ListOptions{
						FieldSelector: fields.OneTermEqualSelector("metadata.name", c.Ingress.Name).String(),
					})
				},
			},
			&extv1beta1.Ingress{},
			c.options.ResyncPeriod,
			cache.Indexers{},
		)
	}

	ingressInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, new interface{}) {
			var newIngress *api.Ingress
			ok := false
			switch c.Ingress.APIVersion {
			case api.SchemeGroupVersion.String():
				if _, oldOK := old.(*api.Ingress); oldOK {
					if newIng, newOK := new.(*api.Ingress); newOK {
						newIngress = newIng
						ok = true
					}
				}
			case "extensions/v1beta1":
				if _, oldOK := old.(*extv1beta1.Ingress); oldOK {
					if newIng, newOK := new.(*extv1beta1.Ingress); newOK {
						newExtIng, err := api.NewEngressFromIngress(newIng)
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
					if _, ok := c.projections[secret+".pem"]; !ok {
						sc, err := c.k8sClient.CoreV1().Secrets(c.Ingress.Namespace).Get(secret, metav1.GetOptions{})
						if err != nil {
							log.Fatalln(err)
						}
						c.projections[secret+".pem"] = ioutil.FileProjection{Mode: 0777, Data: secretToPEMData(sc)}
					}
					secretsUsedMaps[secret+".pem"] = struct{}{}
				}

				for _, cert := range newIngress.Certificates() {
					if certs, err := c.VoyagerClient.Certificates(cert.Namespace).Get(cert.Name, metav1.GetOptions{}); err != nil {
						if certs.Spec.Storage.Secret != nil {
							name := fileNameForCertificate(certs)
							if _, ok := c.projections[name+".pem"]; !ok {
								sc, err := c.k8sClient.CoreV1().Secrets(c.Ingress.Namespace).Get(name, metav1.GetOptions{})
								if err != nil {
									log.Fatalln(err)
								}
								c.projections[name+".pem"] = ioutil.FileProjection{Mode: 0777, Data: secretToPEMData(sc)}
							}
							secretsUsedMaps[name+".pem"] = struct{}{}
						} else if certs.Spec.Storage.Vault != nil {
							// Add from vault
						}
					}
				}

				for k := range c.projections {
					if _, ok := secretsUsedMaps[k+".pem"]; !ok {
						delete(c.projections, k)
					}
				}

				c.MustMount()

				// Update Ingress
				c.Ingress = newIngress
			}
		},
	})
	go ingressInformer.Run(stopCh)
}

func (c *Controller) initCertificateInformer(stopCh <-chan struct{}) {
	certInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (rt.Object, error) {
				return c.VoyagerClient.Certificates(c.Ingress.Namespace).List(metav1.ListOptions{})
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return c.VoyagerClient.Certificates(c.Ingress.Namespace).Watch(metav1.ListOptions{})
			},
		},
		&api.Certificate{},
		c.options.ResyncPeriod,
		cache.Indexers{},
	)

	certInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if cert, ok := obj.(*api.Certificate); ok {
				if c.isCertificateUsedInIngress(cert) {
					c.lock.Lock()
					defer c.lock.Unlock()

					if cert.Spec.Storage.Secret != nil {
						name := fileNameForCertificate(cert)
						if _, ok := c.projections[name+".pem"]; !ok {
							sc, err := c.k8sClient.CoreV1().Secrets(c.Ingress.Namespace).Get(name, metav1.GetOptions{})
							if err != nil {
								log.Fatalln(err)
							}
							c.projections[name+".pem"] = ioutil.FileProjection{Mode: 0777, Data: secretToPEMData(sc)}
						}
					} else if cert.Spec.Storage.Vault != nil {
						// Add from vault
					}

					c.MustMount()
				}
			}
		},
		UpdateFunc: func(old, new interface{}) {
			if _, oldOK := old.(*api.Certificate); oldOK {
				if newCert, newOK := new.(*api.Certificate); newOK {
					if c.isCertificateUsedInIngress(newCert) {
						c.lock.Lock()
						defer c.lock.Unlock()

						if newCert.Spec.Storage.Secret != nil {
							name := fileNameForCertificate(newCert)
							if _, ok := c.projections[name+".pem"]; !ok {
								sc, err := c.k8sClient.CoreV1().Secrets(c.Ingress.Namespace).Get(name, metav1.GetOptions{})
								if err != nil {
									log.Fatalln(err)
								}
								c.projections[name+".pem"] = ioutil.FileProjection{Mode: 0777, Data: secretToPEMData(sc)}
							}
						} else if newCert.Spec.Storage.Vault != nil {
							// Add from vault
						}

						c.MustMount()
					}
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			if cert, ok := obj.(*api.Certificate); ok {
				if c.isCertificateUsedInIngress(cert) {
					c.lock.Lock()
					defer c.lock.Unlock()

					delete(c.projections, fileNameForCertificate(cert)+".pem")
					c.MustMount()
				}
			}
		},
	})
	go certInformer.Run(stopCh)
}

func (c *Controller) isSecretUsedInIngress(s *apiv1.Secret) bool {
	if s.Namespace != c.Ingress.Namespace {
		return false
	}

	for _, secret := range c.Ingress.Secrets() {
		if s.Name == secret {
			return true
		}
	}
	return false
}

func (c *Controller) isCertificateUsedInIngress(s *api.Certificate) bool {
	if s.Namespace != c.Ingress.Namespace {
		return false
	}

	for _, secret := range c.Ingress.Certificates() {
		if s.Name == secret.Name {
			return true
		}
	}
	return false
}

func (c *Controller) Mount() error {
	err := c.writer.Write(c.projections)
	if err != nil {
		return err
	}
	return nil
}

func (c *Controller) MustMount() {
	err := c.Mount()
	if err != nil {
		log.Fatalln(err)
	}
}
