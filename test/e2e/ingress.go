package e2e

import (
	"os/exec"
	"time"

	"github.com/appscode/errors"
	aci "github.com/appscode/k8s-addons/api"
	"github.com/appscode/log"
	"github.com/appscode/voyager/pkg/controller/ingress"
	"github.com/appscode/voyager/test/test-server/testserverclient"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/util/intstr"
	"net/http"
	"strings"
)

const maxRetries = 50

func (i *IngressTestSuit) TestIngressEnsureTPR() error {
	var err error
	for it := 0; it < 10; it++ {
		log.Infoln(it, "Trying to get ingress.appscode.com")
		tpr, err := i.t.KubeClient.Extensions().ThirdPartyResources().Get("ingress.appscode.com")
		if err == nil {
			log.Infoln("Found tpr for ingress with name", tpr.Name)
			break
		}
		err = errors.New().WithCause(err).Internal()
		time.Sleep(time.Second * 5)
		continue
	}
	return err
}

func (ing *IngressTestSuit) TestIngressCreate() error {
	baseIngress := &aci.Ingress{
		ObjectMeta: api.ObjectMeta{
			Name:      "base-ingress",
			Namespace: "default",
		},
		Spec: aci.ExtendedIngressSpec{
			Rules: []aci.ExtendedIngressRule{
				{
					ExtendedIngressRuleValue: aci.ExtendedIngressRuleValue{
						HTTP: &aci.HTTPExtendedIngressRuleValue{
							Paths: []aci.HTTPExtendedIngressPath{
								{
									Path: "/testpath",
									Backend: aci.ExtendedIngressBackend{
										ServiceName: testServerSvc.Name,
										ServicePort: intstr.FromInt(80),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := ing.t.ExtensionClient.Ingress(baseIngress.Namespace).Create(baseIngress)
	if err != nil {
		return err
	}
	defer func() {
		ing.t.ExtensionClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 10)
	var svc *api.Service
	for i := 0; i < maxRetries; i++ {
		svc, err = ing.t.KubeClient.Core().Services(baseIngress.Namespace).Get(ingress.VoyagerPrefix + baseIngress.Name)
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for service to be created")
	}
	if err != nil {
		return err
	}
	log.Infoln("Service Created for loadbalancer, Checking for service endpoints")
	for i := 0; i < maxRetries; i++ {
		_, err = ing.t.KubeClient.Core().Endpoints(svc.Namespace).Get(svc.Name)
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for endpoints to be created")
	}
	if err != nil {
		return err
	}

	var serverAddr string
	if ing.t.config.ProviderName == "minikube" {
		out, err := exec.Command("minikube", "service", svc.Name, "--url").Output()
		if err != nil {
			return err
		}
		serverAddr = strings.TrimSpace(string(out))
	} else {
		for i := 0; i < maxRetries; i++ {
			svc, err = ing.t.KubeClient.Core().Services(baseIngress.Namespace).Get(ingress.VoyagerPrefix + baseIngress.Name)
			if err == nil {
				if len(svc.Status.LoadBalancer.Ingress) > 0 {
					serverAddr = "http://" + svc.Status.LoadBalancer.Ingress[0].IP + ":80"
					break
				}
			}
			time.Sleep(time.Second * 5)
			log.Infoln("Waiting for service to be created")
		}
		if err != nil {
			return err
		}
	}
	log.Infoln("Loadbalancer created, calling http")
	resp, err := testserverclient.NewTestHTTPClient(serverAddr).Method("GET").Path("/testpath/ok").Do()
	if err != nil {
		return err
	}
	log.Infoln("Response", *resp)
	if resp.Method != http.MethodGet {
		return errors.New().WithMessage("Method did not matched").Internal()
	}

	if resp.Path != "/testpath/ok" {
		return errors.New().WithMessage("Path did not matched").Internal()
	}

	return nil
}
