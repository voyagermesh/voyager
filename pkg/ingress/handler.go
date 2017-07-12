package ingress

import (
	"encoding/json"
	"net"
	"reflect"
	"strings"

	"github.com/appscode/errors"
	stringutil "github.com/appscode/go/strings"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	_ "github.com/appscode/voyager/api/install"
	acs "github.com/appscode/voyager/client/clientset"
	"github.com/appscode/voyager/pkg/config"
	"github.com/appscode/voyager/pkg/events"
	"github.com/appscode/voyager/pkg/monitor"
	"github.com/appscode/voyager/pkg/stash"
	"github.com/appscode/voyager/third_party/forked/cloudprovider"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	fakecloudprovider "github.com/appscode/voyager/third_party/forked/cloudprovider/providers/fake"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func NewController(
	kubeClient clientset.Interface,
	extClient acs.ExtensionInterface,
	promClient pcm.MonitoringV1alpha1Interface,
	opt config.Options) *IngressController {
	h := &IngressController{
		KubeClient: kubeClient,
		ExtClient:  extClient,
		PromClient: promClient,
		opt:        opt,
	}
	log.Infoln("Initializing cloud manager for provider", opt.CloudProvider)
	if opt.CloudProvider == "aws" || opt.CloudProvider == "gce" || opt.CloudProvider == "azure" {
		cloudInterface, err := cloudprovider.InitCloudProvider(opt.CloudProvider, opt.CloudConfigFile)
		if err != nil {
			log.Errorln("Failed to initialize cloud provider:"+opt.CloudProvider, err)
		} else {
			log.Infoln("Initialized cloud provider: "+opt.CloudProvider, cloudInterface)
			h.CloudManager = cloudInterface
		}
	} else if opt.CloudProvider == "gke" {
		cloudInterface, err := cloudprovider.InitCloudProvider("gce", opt.CloudConfigFile)
		if err != nil {
			log.Errorln("Failed to initialize cloud provider:"+opt.CloudProvider, err)
		} else {
			log.Infoln("Initialized cloud provider: "+opt.CloudProvider, cloudInterface)
			h.CloudManager = cloudInterface
		}
	} else if opt.CloudProvider == "minikube" {
		h.CloudManager = &fakecloudprovider.FakeCloud{}
	} else {
		log.Infoln("No cloud manager found for provider", opt.CloudProvider)
	}
	return h
}

func UpgradeAllEngress(service, providerName, cloudConfig string,
	kubeClient clientset.Interface,
	extClient acs.ExtensionInterface,
	promClient pcm.MonitoringV1alpha1Interface,
	store stash.Storage,
	ingressClass string,
	operatorServiceAccount string) error {
	ing, err := kubeClient.ExtensionsV1beta1().Ingresses(apiv1.NamespaceAll).List(metav1.ListOptions{
		LabelSelector: labels.Everything().String(),
	})
	if err != nil {
		return errors.FromErr(err).Err()
	}

	eng, err := extClient.Ingresses(apiv1.NamespaceAll).List(metav1.ListOptions{
		LabelSelector: labels.Everything().String(),
	})
	if err != nil {
		return errors.FromErr(err).Err()
	}

	items := make([]api.Ingress, len(ing.Items))
	for i, item := range ing.Items {
		e, err := api.NewEngressFromIngress(item)
		if err != nil {
			continue
		}
		items[i] = *e
	}
	items = append(items, eng.Items...)
	log.Infoln("Updating All Ingress, got total", len(items))
	for i, item := range items {
		engress := &items[i]
		if shouldHandleIngress(engress, ingressClass) {
			log.Infoln("Checking for service", service, "to be used to load balance via ingress", item.Name, item.Namespace)
			if ok, name, namespace := isEngressHaveService(engress, service); ok {
				lbc := NewController(providerName, cloudConfig, kubeClient, extClient, promClient, store, ingressClass, operatorServiceAccount)
				lbc.Resource = &items[i]
				log.Infoln("Trying to Update Ingress", item.Name, item.Namespace)
				if lbc.IsExists() {
					// Loadbalancer resource for this ingress is found in its place,
					// so no need to create the resources. First trying to update
					// the configMap only for the rules.
					// In case of any failure in soft update we will make hard update
					// to the resource. If hard update encounters errors then we will
					// recreate the resource from scratch.
					log.Infoln("Loadbalancer is exists, trying to update")
					cfgErr := lbc.Update(UpdateConfig)
					if cfgErr != nil {
						log.Warningln("Loadbalancer is exists but Soft Update failed. Retring Hard Update")
						restartErr := lbc.Update(RestartHAProxy)
						if restartErr != nil {
							log.Warningln("Loadbalancer is exists, But Hard Update is also failed, recreating with a cleanup")
							lbc.Create()
						}
					}
				} else {
					// This LB should be there. If it is no there. we should create it
					log.Infoln("Loadbalancer is not found, recreating with a cleanup")
					lbc.Create()
				}
				ensureServiceAnnotations(kubeClient, engress, namespace, name)
			}
		}
	}
	return nil
}

func (lbc *IngressController) Handle(e *events.Event) error {
	log.Infof("Engress event %s/%s occurred for %s", e.EventType, e.ResourceType, e.MetaData.Name)
	// convert to extended ingress and then handle
	var engs []interface{}
	if e.ResourceType == events.ExtendedIngress {
		engs = e.RuntimeObj
	} else if e.ResourceType == events.Ingress {
		// convert to extended ingress and then handle
		engs = make([]interface{}, len(e.RuntimeObj))
		for i, ing := range e.RuntimeObj {
			engress, err := api.NewEngressFromIngress(ing)
			if err != nil {
				return errors.FromErr(err).Err()
			}
			engs[i] = engress
		}
	}
	log.Infoln("Size of engs", len(engs), "Size of RuntimeObj", len(e.RuntimeObj))
	if e.EventType.IsAdded() {
		lbc.Resource = engs[0].(*api.Ingress)
		if shouldHandleIngress(lbc.Resource, lbc.opt.IngressClass) {
			if lbc.IsExists() {
				// Loadbalancer resource for this ingress is found in its place,
				// so no need to create the resources. First trying to update
				// the configMap only for the rules.
				// In case of any failure in soft update we will make hard update
				// to the resource. If hard update encounters errors then we will
				// recreate the resource from scratch.
				log.Infoln("Loadbalancer is exists, trying to update")

				if svc, err := lbc.KubeClient.CoreV1().Services(lbc.Resource.Namespace).Get(lbc.Resource.OffshootName(), metav1.GetOptions{}); err == nil {
					// check port
					curPorts := make(map[int]apiv1.ServicePort)
					for _, p := range svc.Spec.Ports {
						curPorts[int(p.Port)] = p
					}

					var updateFW bool
					for svcPort, targetPort := range lbc.Ports {
						if sp, ok := curPorts[svcPort]; !ok || sp.TargetPort.IntValue() != targetPort {
							updateFW = true // new port has to be opened
							break
						} else {
							delete(curPorts, svcPort)
						}
					}
					if len(curPorts) > 0 {
						updateFW = true // additional port was open previously
					}

					if updateFW {
						lbc.Update(UpdateFirewall | UpdateStats)
					} else {
						lbc.Update(UpdateConfig | UpdateStats)
					}
				} else {
					log.Warningln("Loadbalancer is exists but Soft Update failed. Retrying Hard Update")
					restartErr := lbc.Update(RestartHAProxy)
					if restartErr != nil {
						log.Warningln("Loadbalancer is exists, But Hard Update is also failed, recreating with a cleanup")
						lbc.Create()
					}
				}
				return nil
			}
			lbc.Create()
		}
	} else if e.EventType.IsDeleted() {
		lbc.Resource = engs[0].(*api.Ingress)
		if shouldHandleIngress(lbc.Resource, lbc.opt.IngressClass) {
			lbc.Delete()
		}
	} else if e.EventType.IsUpdated() {
		old := engs[0].(*api.Ingress)
		new := engs[1].(*api.Ingress)
		lbc.Resource = new
		if !shouldHandleIngress(lbc.Resource, lbc.opt.IngressClass) {
			return nil
		}

		updateMode := updateType(0)

		if !reflect.DeepEqual(old.ObjectMeta.Annotations, new.ObjectMeta.Annotations) {
			// Ingress Annotations Changed, Apply Changes to Targets
			// The following method do not update to HAProxy config or restart pod. It only sets the annotations
			// to the required targets.
			lbc.UpdateTargetAnnotations(old, new)

			if lbc.isKeepSourceChanged(old, new) {
				updateMode |= UpdateConfig
			}
			if isStatsChanged(old, new) {
				updateMode |= UpdateStats
			}
			// Check for changes in ingress.appscode.com/monitoring-agent
			if newMonSpec, newErr := new.MonitorSpec(); newErr == nil {
				if oldMonSpec, oldErr := old.MonitorSpec(); oldErr == nil {
					if !reflect.DeepEqual(oldMonSpec, newMonSpec) {
						ctrl := monitor.NewPrometheusController(lbc.KubeClient, lbc.PromClient)
						err := ctrl.UpdateMonitor(lbc.Resource, oldMonSpec, newMonSpec)
						if err != nil {
							return errors.FromErr(err).Err()
						}
					}
					if (oldMonSpec == nil && newMonSpec != nil) ||
						(oldMonSpec != nil && newMonSpec == nil) {
						updateMode |= UpdateStats
					}
				}
			}
		}

		if isNewPortChanged(engs[0], engs[1]) || isLoadBalancerSourceRangeChanged(engs[0], engs[1]) {
			updateMode |= UpdateFirewall
		} else if isNewSecretAdded(engs[0], engs[1]) {
			updateMode |= RestartHAProxy
		} else {
			updateMode |= UpdateConfig
		}
		if updateMode > 0 {
			// For ingress update update HAProxy once
			lbc.Update(updateMode)
		}
	}
	svcs, err := lbc.KubeClient.CoreV1().Services(apiv1.NamespaceAll).List(metav1.ListOptions{})
	if err == nil {
		for _, svc := range svcs.Items {
			ensureServiceAnnotations(lbc.KubeClient, lbc.Resource, svc.Namespace, svc.Name)
		}
	}
	return nil
}

type IngressRef struct {
	APISchema string `json:"apiSchema"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

const (
	ingressClassAnnotationKey   = "kubernetes.io/ingress.class"
	ingressClassAnnotationValue = "voyager"
)

// if ingressClass == "voyager", then only handle ingress that has voyager annotation
// if ingressClass == "", then handle no annotaion or voyager annotation
func shouldHandleIngress(resource *api.Ingress, ingressClass string) bool {
	// https://github.com/appscode/voyager/blob/master/api/conversion_v1beta1.go#L44
	if resource.Annotations[api.APISchema] == api.APISchemaEngress {
		// Resource Type is Extended Ingress So we should always Handle this
		return true
	}
	kubeAnnotation, _ := resource.Annotations[ingressClassAnnotationKey]
	return kubeAnnotation == ingressClass || kubeAnnotation == ingressClassAnnotationValue
}

func ensureServiceAnnotations(client clientset.Interface, r *api.Ingress, namespace, name string) {
	svc, err := client.CoreV1().Services(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return
	}
	if svc.Annotations == nil {
		svc.Annotations = make(map[string]string)
	}

	if ok, _, _ := isEngressHaveService(r, name+"."+namespace); ok {
		list := make([]IngressRef, 0)
		val, ok := svc.Annotations[api.EgressPoints]
		if ok {
			err := json.Unmarshal([]byte(val), list)
			if err == nil {
				found := false
				for _, engs := range list {
					if engs.APISchema == r.APISchema() && engs.Name == r.Name && engs.Namespace == r.Namespace {
						found = true
						break
					}
				}
				if !found {
					list = append(list, IngressRef{
						APISchema: r.APISchema(),
						Name:      r.Name,
						Namespace: r.Namespace,
					})
				}
			}
		} else {
			list = append(list, IngressRef{
				APISchema: r.APISchema(),
				Name:      r.Name,
				Namespace: r.Namespace,
			})
		}

		data, err := json.Marshal(list)
		if err == nil {
			svc.Annotations[api.EgressPoints] = string(data)
		}
		client.CoreV1().Services(namespace).Update(svc)
		return
	}
	// Lets check if service still have the annotation for this ingress.
	val, ok := svc.Annotations[api.EgressPoints]
	if ok {
		list := make([]IngressRef, 0)
		err := json.Unmarshal([]byte(val), list)
		if err == nil {
			for i, engs := range list {
				if engs.APISchema == r.APISchema() && engs.Name == r.Name && engs.Namespace == r.Namespace {
					// remove the annotation key
					list = append(list[:i], list[i+1:]...)
				}
			}
			data, err := json.Marshal(list)
			if err == nil {
				svc.Annotations[api.EgressPoints] = string(data)
			}
		}
		client.CoreV1().Services(namespace).Update(svc)
	}
}

func isNewPortChanged(old interface{}, new interface{}) bool {
	o := old.(*api.Ingress)
	n := new.(*api.Ingress)

	oldPortLists := make([]string, 0)
	for _, rs := range o.Spec.Rules {
		for _, port := range rs.TCP {
			oldPortLists = append(oldPortLists, port.Port.String())
		}
	}

	for _, rs := range n.Spec.Rules {
		for _, port := range rs.TCP {
			if !stringutil.Contains(oldPortLists, port.Port.String()) {
				return true
			}
		}
	}
	return false
}

func isNewSecretAdded(old interface{}, new interface{}) bool {
	o := old.(*api.Ingress)
	n := new.(*api.Ingress)

	oldSecretLists := make([]string, 0)
	for _, rs := range o.Spec.TLS {
		oldSecretLists = append(oldSecretLists, rs.SecretName)
	}
	for _, rs := range o.Spec.Rules {
		for _, tcp := range rs.TCP {
			oldSecretLists = append(oldSecretLists, tcp.SecretName)
		}
	}

	for _, rs := range n.Spec.Rules {
		for _, port := range rs.TCP {
			if !stringutil.Contains(oldSecretLists, port.SecretName) {
				return true
			}
		}
	}
	for _, rs := range n.Spec.TLS {
		if !stringutil.Contains(oldSecretLists, rs.SecretName) {
			return true
		}
	}
	return false
}

func isLoadBalancerSourceRangeChanged(old interface{}, new interface{}) bool {
	oldObj, oldOk := old.(*api.Ingress)
	newObj, newOk := new.(*api.Ingress)

	if oldOk && newOk {
		oldipset := make(map[string]bool)
		for _, oldrange := range oldObj.Spec.LoadBalancerSourceRanges {
			k, ok := ipnet(oldrange)
			if ok {
				oldipset[k] = true
			}
		}

		newipset := make(map[string]bool)
		for _, newrange := range newObj.Spec.LoadBalancerSourceRanges {
			k, ok := ipnet(newrange)
			if ok {
				newipset[k] = true
				if _, found := oldipset[k]; !found {
					return true
				}
			}
		}

		if len(newipset) != len(oldipset) {
			return true
		}
	}

	return false
}

func ipnet(spec string) (string, bool) {
	spec = strings.TrimSpace(spec)
	_, ipnet, err := net.ParseCIDR(spec)
	if err != nil {
		return "", false
	}
	return ipnet.String(), true
}

func isStatsChanged(old *api.Ingress, new *api.Ingress) bool {
	return isMapKeyChanged(old.Annotations, new.Annotations, api.StatsOn) ||
		isMapKeyChanged(old.Annotations, new.Annotations, api.StatsPort) ||
		isMapKeyChanged(old.Annotations, new.Annotations, api.StatsServiceName) ||
		isMapKeyChanged(old.Annotations, new.Annotations, api.StatsSecret)
}

func (lbc *IngressController) isKeepSourceChanged(old *api.Ingress, new *api.Ingress) bool {
	return lbc.opt.CloudProvider == "aws" &&
		lbc.Resource.LBType() == api.LBTypeLoadBalancer &&
		isMapKeyChanged(old.Annotations, new.Annotations, api.KeepSourceIP)
}

func isMapKeyChanged(oldMap map[string]string, newMap map[string]string, key string) bool {
	oldValue, oldOk := oldMap[key]
	newValue, newOk := newMap[key]
	return oldOk != newOk || oldValue != newValue
}

func isEngressHaveService(ing *api.Ingress, service string) (bool, string, string) {
	serviceNotWithDefault := service
	if strings.HasSuffix(serviceNotWithDefault, "."+ing.Namespace) {
		serviceNotWithDefault = serviceNotWithDefault[:strings.Index(serviceNotWithDefault, "."+ing.Namespace)]
	}
	log.Infoln("Checking Ingress", ing.Name, "for service name", serviceNotWithDefault)
	if ing.Spec.Backend != nil {
		if ing.Spec.Backend.ServiceName == service || ing.Spec.Backend.ServiceName == serviceNotWithDefault {
			name, namespace := splitNameNamespace(service, serviceNotWithDefault, ing.Namespace)
			return true, name, namespace
		}
	}

	for _, rules := range ing.Spec.Rules {
		if rules.HTTP != nil {
			for _, svc := range rules.HTTP.Paths {
				if svc.Backend.ServiceName == service || svc.Backend.ServiceName == serviceNotWithDefault {
					name, namespace := splitNameNamespace(service, serviceNotWithDefault, ing.Namespace)
					return true, name, namespace
				}
			}
		}

		for _, svc := range rules.TCP {
			if svc.Backend.ServiceName == service || svc.Backend.ServiceName == serviceNotWithDefault {
				name, namespace := splitNameNamespace(service, serviceNotWithDefault, ing.Namespace)
				return true, name, namespace
			}
		}
	}
	return false, "", ""
}

func splitNameNamespace(fqdn, name, namespace string) (string, string) {
	if fqdn == (name+"."+namespace) || fqdn == name {
		return name, namespace
	}
	if strings.HasPrefix(fqdn, name+".") {
		return name, name[:strings.LastIndex(name, ".")]
	}
	return name, namespace
}
