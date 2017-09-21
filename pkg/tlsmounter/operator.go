package tlsmounter

import (
	"errors"
	"time"

	"github.com/appscode/go/hold"
	voyagerv1beta1 "github.com/appscode/voyager/apis/voyager/v1beta1"
	acs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
)

type TLSMountConfig struct {
	Namespace     string
	IngressRef    apiv1.ObjectReference
	MountLocation string
	KubeConfig    *rest.Config
	KubeClient    clientset.Interface
	VoyagerClient acs.VoyagerV1beta1Interface
	VolumeMounter *secretMounter
}

type SSLMounter struct {
	*TLSMountConfig
}

func New(c *TLSMountConfig) *SSLMounter {
	return &SSLMounter{TLSMountConfig: c}
}

func (m *SSLMounter) Setup() error {
	var ingress *voyagerv1beta1.Ingress
	switch m.IngressRef.APIVersion {
	case voyagerv1beta1.SchemeGroupVersion.String():
		var err error
		ingress, err = m.VoyagerClient.Ingresses(m.Namespace).Get(m.IngressRef.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
	case "extensions/v1beta1":
		ing, err := m.KubeClient.ExtensionsV1beta1().Ingresses(m.Namespace).Get(m.IngressRef.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		ingress, err = voyagerv1beta1.NewEngressFromIngress(ing)
		if err != nil {
			return err
		}
	default:
		return errors.New("ingress API Schema unrecognized")
	}

	mounter, err := NewIngressSecretMounter(m.KubeClient, m.VoyagerClient, ingress, m.MountLocation, "", time.Minute*5)
	if err != nil {
		return err
	}

	err = mounter.Mount()
	if err != nil {
		return err
	}
	m.VolumeMounter = mounter
	return nil
}

func (m *SSLMounter) Run() {
	m.VolumeMounter.Run(wait.NeverStop)
	hold.Hold()
}
