package tlsmounter

import (
	"errors"
	"time"

	"github.com/appscode/go/hold"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	acs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/certificate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

type Controller struct {
	KubeClient    clientset.Interface
	VoyagerClient acs.VoyagerV1beta1Interface
	IngressRef    apiv1.ObjectReference
	MountLocation string
	VolumeMounter *secretMounter
	store         *certificate.CertStore
}

func (c *Controller) Setup() error {
	var err error
	c.store, err = certificate.NewCertStore(c.KubeClient, c.VoyagerClient)
	if err != nil {
		return err
	}

	var ingress *api.Ingress
	switch c.IngressRef.APIVersion {
	case api.SchemeGroupVersion.String():
		var err error
		ingress, err = c.VoyagerClient.Ingresses(c.IngressRef.Namespace).Get(c.IngressRef.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
	case "extensions/v1beta1":
		ing, err := c.KubeClient.ExtensionsV1beta1().Ingresses(c.IngressRef.Namespace).Get(c.IngressRef.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		ingress, err = api.NewEngressFromIngress(ing)
		if err != nil {
			return err
		}
	default:
		return errors.New("ingress API Schema unrecognized")
	}

	mounter, err := NewIngressSecretMounter(c.KubeClient, c.VoyagerClient, ingress, c.MountLocation, "", time.Minute*5)
	if err != nil {
		return err
	}

	err = mounter.Mount()
	if err != nil {
		return err
	}
	c.VolumeMounter = mounter
	return nil
}

func (c *Controller) Run() {
	c.VolumeMounter.Run(wait.NeverStop)
	hold.Hold()
}
