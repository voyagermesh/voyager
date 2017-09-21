package operator

import (
	"errors"
	"github.com/appscode/go/hold"
	voyagerv1beta1 "github.com/appscode/voyager/apis/voyager/v1beta1"
	acs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"time"
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

type Operator struct {
	*TLSMountConfig
}

func New(c *TLSMountConfig) *Operator {
	return &Operator{TLSMountConfig: c}
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

	mounter, err := NewIngressSecretMounter(op.KubeClient, op.VoyagerClient, ingress, op.MountLocation, "", time.Minute*5)
	if err != nil {
		return err
	}

	err = mounter.Mount()
	if err != nil {
		return err
	}
	op.VolumeMounter = mounter
	return nil
}

func (op *Operator) Run() {
	op.VolumeMounter.Run(wait.NeverStop)
	hold.Hold()
}
