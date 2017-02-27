package e2e

import (
	"net/http"
	"text/template"
	"time"

	"github.com/appscode/errors"
	aci "github.com/appscode/k8s-addons/api"
	"github.com/appscode/log"
	"github.com/appscode/voyager/pkg/controller/ingress"
	"github.com/appscode/voyager/test/test-server/testserverclient"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/util/intstr"
)

const maxRetries = 50

var (
	defaultUrlTemplate = template.Must(template.New("svc-template").Parse("http://{{.IP}}:{{.Port}}"))
)

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
		if ing.t.config.Cleanup {
			ing.t.ExtensionClient.Ingress(baseIngress.Namespace).Delete(baseIngress.Name)
		}
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

	serverAddr, err := ing.getURLs(baseIngress)
	if err != nil {
		return err
	}
	log.Infoln("Loadbalancer created, calling http")
	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath/ok").Do()
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Internal()
		}
		log.Infoln("Response", *resp)
		if resp.Method != http.MethodGet {
			return errors.New().WithMessage("Method did not matched").Internal()
		}

		if resp.Path != "/testpath/ok" {
			return errors.New().WithMessage("Path did not matched").Internal()
		}
	}

	return nil
}

func (ing *IngressTestSuit) TestIngressDaemonCreate() error {
	var nodeSelector = func() string {
		if ing.t.config.ProviderName == "minikube" {
			return "kubernetes.io/hostname=minikube"
		} else {
			return "kubernetes.io/hostname=" + ing.t.config.ClusterName + "-master"
		}
		return ""
	}

	baseDaemonIngress := &aci.Ingress{
		ObjectMeta: api.ObjectMeta{
			Name:      "base-d-ingress",
			Namespace: "default",
			Annotations: map[string]string{
				ingress.LBType:             ingress.LBDaemon,
				ingress.DaemonNodeSelector: nodeSelector(),
			},
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

	_, err := ing.t.ExtensionClient.Ingress(baseDaemonIngress.Namespace).Create(baseDaemonIngress)
	if err != nil {
		return err
	}
	defer func() {
		if ing.t.config.Cleanup {
			ing.t.ExtensionClient.Ingress(baseDaemonIngress.Namespace).Delete(baseDaemonIngress.Name)
		}
	}()

	// Wait sometime to loadbalancer be opened up.
	time.Sleep(time.Second * 10)
	for i := 0; i < maxRetries; i++ {
		_, err := ing.t.KubeClient.Core().Services(baseDaemonIngress.Namespace).Get(ingress.VoyagerPrefix + baseDaemonIngress.Name)
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		log.Infoln("Waiting for service to be created")
	}
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}

	serverAddr, err := ing.getDaemonURLs(baseDaemonIngress)
	if err != nil {
		return err
	}
	log.Infoln("Loadbalancer created, calling http endpoints for test, Total url found", len(serverAddr))
	for _, url := range serverAddr {
		resp, err := testserverclient.NewTestHTTPClient(url).Method("GET").Path("/testpath/ok").Do()
		if err != nil {
			return errors.New().WithCause(err).WithMessage("Failed to connect with server").Internal()
		}
		log.Infoln("Response", *resp)
		if resp.Method != http.MethodGet {
			return errors.New().WithMessage("Method did not matched").Internal()
		}

		if resp.Path != "/testpath/ok" {
			return errors.New().WithMessage("Path did not matched").Internal()
		}
	}
	return nil
}
