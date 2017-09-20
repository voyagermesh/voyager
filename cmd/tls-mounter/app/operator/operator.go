package operator

import (
	"errors"
	voyagerv1beta1 "github.com/appscode/voyager/apis/voyager/v1beta1"
	acs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
}

type Operator struct {
	*TLSMountConfig
}

func New(c *TLSMountConfig) *Operator {
	return &Operator{c}
}

func (op *Operator) Setup() error {
	var ingress *voyagerv1beta1.Ingress
	switch op.IngressRef.APIVersion {
	case voyagerv1beta1.SchemeGroupVersion.String():
		var err error
		ingress, err = op.VoyagerClient.Ingresses(op.Namespace).Get(op.IngressRef.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
	case "extensions/v1beta1":
		ing, err := op.KubeClient.ExtensionsV1beta1().Ingresses(op.Namespace).Get(op.IngressRef.Name, metav1.GetOptions{})
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

	for _, secret := range ingress.Secrets() {

	}
	return nil
}

func (op *Operator) Watch() error {

	return nil
}
