package ingress

import (
	"encoding/json"
	"net"
	"reflect"
	"strings"

	"github.com/appscode/errors"
	stringutil "github.com/appscode/go/strings"
	"github.com/appscode/log"
	aci "github.com/appscode/voyager/api"
	_ "github.com/appscode/voyager/api/install"
	acs "github.com/appscode/voyager/client/clientset"
	"github.com/appscode/voyager/pkg/events"
	"github.com/appscode/voyager/pkg/stash"
	"github.com/appscode/voyager/third_party/forked/cloudprovider"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	fakecloudprovider "github.com/appscode/voyager/third_party/forked/cloudprovider/providers/fake"
	kapi "k8s.io/kubernetes/pkg/api"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/labels"
)

func NewEngressController(clusterName, providerName string,
	kubeClient clientset.Interface,
	extClient acs.ExtensionInterface,
	store *stash.Storage,
	ingressClass string) *EngressController {
	h := &EngressController{
		ClusterName:   clusterName,
		ProviderName:  providerName,
		IngressClass:  ingressClass,
		KubeClient:    kubeClient,
		ExtClient:     extClient,
		Storage:       store,
		EndpointStore: store.EndpointStore,
	}
	log.Infoln("Initializing cloud manager for provider", providerName)
	if providerName == "aws" || providerName == "gce" || providerName == "azure" {
		cloudInterface, err := cloudprovider.GetCloudProvider(providerName, nil)
		if err != nil {
			log.Errorln("Failed to initialize cloud provider:"+providerName, err)
		} else {
			log.Infoln("Initialized cloud provider: "+providerName, cloudInterface)
			h.CloudManager = cloudInterface
		}
	} else if providerName == "gke" {
		cloudInterface, err := cloudprovider.GetCloudProvider("gce", nil)
		if err != nil {
			log.Errorln("Failed to initialize cloud provider:"+providerName, err)
		} else {
			log.Infoln("Initialized cloud provider: "+providerName, cloudInterface)
			h.CloudManager = cloudInterface
		}
	} else if providerName == "minikube" {
		h.CloudManager = &fakecloudprovider.FakeCloud{}
	} else {
		log.Infoln("No cloud manager found for provider", providerName)
	}
	return h
}

func UpgradeAllEngress(service, clusterName, providerName string,
	kubeClient clientset.Interface,
	extClient acs.ExtensionInterface,
	store *stash.Storage,
	ingressClass string) error {
	ing, err := kubeClient.Extensions().Ingresses(kapi.NamespaceAll).List(kapi.ListOptions{
		LabelSelector: labels.Everything(),
	})
	if err != nil {
		return errors.FromErr(err).Err()
	}

	eng, err := extClient.Ingress(kapi.NamespaceAll).List(kapi.ListOptions{
		LabelSelector: labels.Everything(),
	})
	if err != nil {
		return errors.FromErr(err).Err()
	}

	items := make([]aci.Ingress, len(ing.Items))
	for i, item := range ing.Items {
		e, err := aci.NewEngressFromIngress(item)
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
				lbc := NewEngressController(clusterName, providerName, kubeClient, extClient, store, ingressClass)
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

func (lbc *EngressController) Handle(e *events.Event) error {
	log.Infof("Engress event %s/%s occurred for %s", e.EventType, e.ResourceType, e.MetaData.Name)
	// convert to extended ingress and then handle
	var engs []interface{}
	if e.ResourceType == events.ExtendedIngress {
		engs = e.RuntimeObj
	} else if e.ResourceType == events.Ingress {
		// convert to extended ingress and then handle
		engs = make([]interface{}, len(e.RuntimeObj))
		for i, ing := range e.RuntimeObj {
			engress, err := aci.NewEngressFromIngress(ing)
			if err != nil {
				return errors.FromErr(err).Err()
			}
			engs[i] = engress
		}
	}
	log.Infoln("Size of engs", len(engs), "Size of RuntimeObj", len(e.RuntimeObj))
	if e.EventType.IsAdded() {
		lbc.Resource = engs[0].(*aci.Ingress)
		if shouldHandleIngress(lbc.Resource, lbc.IngressClass) {
			if lbc.IsExists() {
				// Loadbalancer resource for this ingress is found in its place,
				// so no need to create the resources. First trying to update
				// the configMap only for the rules.
				// In case of any failure in soft update we will make hard update
				// to the resource. If hard update encounters errors then we will
				// recreate the resource from scratch.
				log.Infoln("Loadbalancer is exists, trying to update")

				if svc, err := lbc.KubeClient.Core().Services(lbc.Resource.Namespace).Get(lbc.OffshootName()); err == nil {
					// check port
					curPorts := make(map[int]int)
					for _, p := range svc.Spec.Ports {
						curPorts[int(p.Port)] = 1
					}

					var updateFW bool
					for _, p := range lbc.Ports {
						if _, ok := curPorts[p]; !ok {
							updateFW = true // new port has to be opened
							break
						} else {
							delete(curPorts, p)
						}
					}
					updateFW = len(curPorts) > 0 // additional port was open previously

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
		lbc.Resource = engs[0].(*aci.Ingress)
		if shouldHandleIngress(lbc.Resource, lbc.IngressClass) {
			lbc.Delete()
		}
	} else if e.EventType.IsUpdated() {
		lbc.Resource = engs[1].(*aci.Ingress)
		if !reflect.DeepEqual(engs[0].(*aci.Ingress).ObjectMeta.Annotations, engs[1].(*aci.Ingress).ObjectMeta.Annotations) {
			// Ingress Annotations Changed, Apply Changes to Targets
			// The following method do not update to HAProxy config or restart pod. It only sets the annotations
			// to the required targets.
			lbc.UpdateTargetAnnotations(engs[0].(*aci.Ingress), engs[1].(*aci.Ingress))
		}

		updateMode := updateType(0)
		if lbc.isKeepSourceChanged(engs[0].(*aci.Ingress), engs[1].(*aci.Ingress)) {
			updateMode |= UpdateConfig
		}

		if isStatsChanged(engs[0].(*aci.Ingress), engs[1].(*aci.Ingress)) {
			updateMode |= UpdateStats
		}

		if !reflect.DeepEqual(engs[0].(*aci.Ingress).Spec, engs[1].(*aci.Ingress).Spec) {
			if shouldHandleIngress(lbc.Resource, lbc.IngressClass) {
				if isNewPortChanged(engs[0], engs[1]) || isLoadBalancerSourceRangeChanged(engs[0], engs[1]) {
					updateMode |= UpdateFirewall
				} else if isNewSecretAdded(engs[0], engs[1]) {
					updateMode |= RestartHAProxy
				} else {
					updateMode |= UpdateConfig
				}
			}
		}
		if updateMode > 0 {
			// For ingress update update HAProxy once
			lbc.Update(updateMode)
		}
	}
	svcs, err := lbc.KubeClient.Core().Services(kapi.NamespaceAll).List(kapi.ListOptions{})
	if err == nil {
		for _, svc := range svcs.Items {
			ensureServiceAnnotations(lbc.KubeClient, lbc.Resource, svc.Namespace, svc.Name)
		}
	}
	return nil
}

type IngressValue struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type IngressValueList struct {
	Items []IngressValue `json:"items"`
}

const (
	ingressClassAnnotationKey   = "kubernetes.io/ingress.class"
	ingressClassAnnotationValue = "voyager"
)

// if ingressClass == "voyager", then only handle ingress that has voyager annotation
// if ingressClass == "", then handle no annotaion or voyager annotation
func shouldHandleIngress(resource *aci.Ingress, ingressClass string) bool {
	// https://github.com/appscode/voyager/blob/master/api/conversion_v1beta1.go#L44
	if resource.Annotations[aci.APISchema] == aci.EngressKey+"/"+aci.V1beta1SchemeGroupVersion.Version {
		// Resource Type is Extended Ingress So we should always Handle this
		return true
	}
	kubeAnnotation, _ := resource.Annotations[ingressClassAnnotationKey]
	return kubeAnnotation == ingressClass || kubeAnnotation == ingressClassAnnotationValue
}

func ensureServiceAnnotations(client clientset.Interface, ing *aci.Ingress, namespace, name string) {
	svc, err := client.Core().Services(namespace).Get(name)
	if err == nil {
		if ok, _, _ := isEngressHaveService(ing, name+"."+namespace); ok {
			list := &IngressValueList{}
			val, ok := svc.Annotations[aci.EngressKey]
			if ok {
				err := json.Unmarshal([]byte(val), list)
				if err == nil {
					found := false
					for _, engs := range list.Items {
						if engs.Name == ing.Name && engs.Namespace == ing.Namespace {
							found = true
							break
						}
					}
					if !found {
						list.Items = append(list.Items, IngressValue{
							Name:      ing.Name,
							Namespace: ing.Namespace,
						})
					}
				}
			} else {
				list.Items = make([]IngressValue, 0)
				list.Items = append(list.Items, IngressValue{
					Name:      ing.Name,
					Namespace: ing.Namespace,
				})
			}

			data, err := json.Marshal(list)
			if err == nil {
				if svc.Annotations == nil {
					svc.Annotations = make(map[string]string)
				}
				svc.Annotations[aci.EngressKey] = string(data)
			}

			client.Core().Services(namespace).Update(svc)
		} else {
			// Lets check if service still have the annotation for this ingress.
			val, ok := svc.Annotations[aci.EngressKey]
			if ok {
				list := &IngressValueList{}
				err := json.Unmarshal([]byte(val), list)
				if err == nil {
					for i, engs := range list.Items {
						if engs.Name == ing.Name && engs.Namespace == ing.Namespace {
							// remove the annotation key
							list.Items = append(list.Items[:i], list.Items[i+1:]...)
						}
					}
					data, err := json.Marshal(list)
					if err == nil {
						if svc.Annotations == nil {
							svc.Annotations = make(map[string]string)
						}
						svc.Annotations[aci.EngressKey] = string(data)
					}
				}
				client.Core().Services(namespace).Update(svc)
			}
		}
	}
}

func isNewPortChanged(old interface{}, new interface{}) bool {
	o := old.(*aci.Ingress)
	n := new.(*aci.Ingress)

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
	o := old.(*aci.Ingress)
	n := new.(*aci.Ingress)

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
	oldObj, oldOk := old.(*aci.Ingress)
	newObj, newOk := new.(*aci.Ingress)

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

func isStatsChanged(old *aci.Ingress, new *aci.Ingress) bool {
	return isMapKeyChanged(old.Annotations, new.Annotations, StatsOn) ||
		isMapKeyChanged(old.Annotations, new.Annotations, StatsPort) ||
		isMapKeyChanged(old.Annotations, new.Annotations, StatsServiceName) ||
		isMapKeyChanged(old.Annotations, new.Annotations, StatsSecret)
}

func (lbc *EngressController) isKeepSourceChanged(old *aci.Ingress, new *aci.Ingress) bool {
	return lbc.ProviderName == "aws" &&
		lbc.Resource.LBType() == LBTypeLoadBalancer &&
		isMapKeyChanged(old.Annotations, new.Annotations, KeepSourceIP)
}

func isMapKeyChanged(oldMap map[string]string, newMap map[string]string, key string) bool {
	oldValue, oldOk := oldMap[key]
	newValue, newOk := newMap[key]
	return oldOk != newOk || oldValue != newValue
}

func isEngressHaveService(ing *aci.Ingress, service string) (bool, string, string) {
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
