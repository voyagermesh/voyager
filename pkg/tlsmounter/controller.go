package tlsmounter

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/appscode/go/ioutil"
	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	acs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	"github.com/appscode/voyager/pkg/certificate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

type Controller struct {
	KubeClient    clientset.Interface
	VoyagerClient acs.VoyagerV1beta1Interface
	IngressRef    apiv1.ObjectReference
	MountPath     string
	CmdFile       string
	store         *certificate.CertStore
	Ingress       *api.Ingress
	ResyncPeriod  time.Duration

	lock            sync.Mutex
	fileProjections map[string]ioutil.FileProjection
	writer          *ioutil.AtomicWriter
}

func (c *Controller) Setup() error {
	var err error
	c.store, err = certificate.NewCertStore(c.KubeClient, c.VoyagerClient)
	if err != nil {
		return err
	}

	switch c.IngressRef.APIVersion {
	case api.SchemeGroupVersion.String():
		var err error
		c.Ingress, err = c.VoyagerClient.Ingresses(c.IngressRef.Namespace).Get(c.IngressRef.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
	case "extensions/v1beta1":
		ing, err := c.KubeClient.ExtensionsV1beta1().Ingresses(c.IngressRef.Namespace).Get(c.IngressRef.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		c.Ingress, err = api.NewEngressFromIngress(ing)
		if err != nil {
			return err
		}
	default:
		return errors.New("ingress API Schema unrecognized")
	}

	payloads := make(map[string]ioutil.FileProjection)
	for _, secret := range c.Ingress.Secrets() {
		sc, err := c.KubeClient.CoreV1().Secrets(c.Ingress.Namespace).Get(secret, metav1.GetOptions{})
		if err != nil {
			return err
		}
		payloads[secret+".pem"] = ioutil.FileProjection{Mode: 0777, Data: secretToPEMData(sc)}
	}

	for _, cert := range c.Ingress.Certificates() {
		if certs, err := c.VoyagerClient.Certificates(cert.Namespace).Get(cert.Name, metav1.GetOptions{}); err != nil {
			if certs.Spec.Storage.Secret != nil {
				name := fileNameForCertificate(certs)
				sc, err := c.KubeClient.CoreV1().Secrets(c.Ingress.Namespace).Get(name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				payloads[name+".pem"] = ioutil.FileProjection{Mode: 0777, Data: secretToPEMData(sc)}
			} else if certs.Spec.Storage.Vault != nil {
				// Add from vault
			}
		}
	}

	location := strings.TrimSuffix(c.MountPath, "/")
	writer, err := ioutil.NewAtomicWriter(location)
	if err != nil {
		return err
	}
	c.fileProjections = payloads
	c.writer = writer

	return c.Mount()
}

var updateReceived, mountPerformed uint64

func incUpdateReceivedCounter() {
	atomic.AddUint64(&updateReceived, 1)
	log.Infoln("Update Received:", atomic.LoadUint64(&updateReceived))
}

func incMountCounter() {
	atomic.AddUint64(&mountPerformed, 1)
	log.Infoln("Mount Performed:", atomic.LoadUint64(&mountPerformed))
}

func runCmd(path string) error {
	log.Infoln("calling boot file to execute")
	output, err := exec.Command("sh", "-c", path).CombinedOutput()
	msg := fmt.Sprintf("%v", string(output))
	log.Infoln("Output:\n", msg)
	if err != nil {
		log.Errorln("failed to run cmd")
		return fmt.Errorf("error restarting %v: %v", msg, err)
	}
	log.Infoln("boot file executed")
	return nil
}
