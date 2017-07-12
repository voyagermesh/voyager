package operator

import (
	"encoding/json"
	"strings"

	acrt "github.com/appscode/go/runtime"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	_ "github.com/appscode/voyager/api/install"
	"github.com/appscode/voyager/pkg/ingress"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (op *Operator) WatchServices() {
	defer acrt.HandleCrash()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return op.KubeClient.CoreV1().Services(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.KubeClient.CoreV1().Services(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&apiv1.Service{},
		op.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if svc, ok := obj.(*apiv1.Service); ok {
					log.Infof("Service %s@%s added", svc.Name, svc.Namespace)
					op.updateHAProxies(svc)
				}
			},
			DeleteFunc: func(obj interface{}) {
				if svc, ok := obj.(*apiv1.Service); ok {
					log.Infof("Service %s@%s deleted", svc.Name, svc.Namespace)

					op.restoreServiceIfRequired(svc)
					op.updateHAProxies(svc)
				}
			},
		},
	)
	ctrl.Run(wait.NeverStop)
}

func (op *Operator) restoreServiceIfRequired(svc *apiv1.Service) error {
	if svc.Annotations == nil {
		return nil
	}

	// deleted resource have source reference
	engress, err := op.findOrigin(svc.ObjectMeta)
	if err != nil {
		return err
	}

	if svc.Name == engress.StatsServiceName() && !engress.Stats() {
		return nil
	}

	// Ingress Still exists, restore resource
	log.Infof("Service %s@%s requires restoration", svc.Name, svc.Namespace)
	svc.Spec.ClusterIP = "" // Remove cluster IP
	svc.SelfLink = ""
	svc.ResourceVersion = ""
	// Old resource and annotations are missing so we need to add the annotations
	if svc.Annotations == nil {
		svc.Annotations = make(map[string]string)
	}
	svc.Annotations[api.OriginAPISchema] = engress.APISchema()
	svc.Annotations[api.OriginName] = engress.Name

	_, err = op.KubeClient.CoreV1().Services(svc.Namespace).Create(svc)
	return err
}

func (op *Operator) updateHAProxies(svc *apiv1.Service) error {
	ing, err := op.KubeClient.ExtensionsV1beta1().Ingresses(apiv1.NamespaceAll).List(metav1.ListOptions{LabelSelector: labels.Everything().String()})
	if err != nil {
		return err
	}
	eng, err := op.ExtClient.Ingresses(apiv1.NamespaceAll).List(metav1.ListOptions{LabelSelector: labels.Everything().String()})
	if err != nil {
		return err
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
		if engress.ShouldHandleIngress(op.Opt.IngressClass) {
			log.Infoln("Checking for service", svc, "to be used to load balance via ingress", item.Name, item.Namespace)
			if op.IngressUsesService(engress, svc) {
				ctrl := ingress.NewController(op.KubeClient, op.ExtClient, op.PromClient, op.Opt, engress)
				if ctrl.IsExists() {
					// Loadbalancer resource for this ingress is found in its place,
					// so no need to create the resources. First trying to update
					// the configMap only for the rules.
					// In case of any failure in soft update we will make hard update
					// to the resource. If hard update encounters errors then we will
					// recreate the resource from scratch.
					log.Infoln("Loadbalancer is exists, trying to update")
					cfgErr := ctrl.Update(ingress.UpdateConfig)
					if cfgErr != nil {
						log.Warningln("Loadbalancer is exists but Soft Update failed. Retring Hard Update")
						restartErr := ctrl.Update(ingress.RestartHAProxy)
						if restartErr != nil {
							log.Warningln("Loadbalancer is exists, But Hard Update is also failed, recreating with a cleanup")
							ctrl.Create()
						}
					}
				} else {
					// This LB should be there. If it is no there. we should create it
					log.Infoln("Loadbalancer is not found, recreating with a cleanup")
					ctrl.Create()
				}
				op.ensureServiceAnnotations(engress, svc)
			}
		}
	}
	return nil
}

func (op *Operator) ensureServiceAnnotations(r *api.Ingress, svc *apiv1.Service) {
	if svc.Annotations == nil {
		svc.Annotations = make(map[string]string)
	}
	if op.IngressUsesService(r, svc) {
		list := make([]api.IngressRef, 0)
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
					list = append(list, api.IngressRef{
						APISchema: r.APISchema(),
						Name:      r.Name,
						Namespace: r.Namespace,
					})
				}
			}
		} else {
			list = append(list, api.IngressRef{
				APISchema: r.APISchema(),
				Name:      r.Name,
				Namespace: r.Namespace,
			})
		}

		data, err := json.Marshal(list)
		if err == nil {
			svc.Annotations[api.EgressPoints] = string(data)
		}
		op.KubeClient.CoreV1().Services(svc.Namespace).Update(svc)
		return
	}
	// Lets check if service still have the annotation for this ingress.
	val, ok := svc.Annotations[api.EgressPoints]
	if ok {
		list := make([]api.IngressRef, 0)
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
		op.KubeClient.CoreV1().Services(svc.Namespace).Update(svc)
	}
}

func (op *Operator) IngressUsesService(ing *api.Ingress, svc *apiv1.Service) bool {
	svcFQN := svc.Name + "." + svc.Namespace

	fqn := func(svcName string) string {
		if strings.ContainsRune(svcName, '.') {
			return svcName
		}
		return svcName + "." + ing.Namespace
	}

	if ing.Spec.Backend != nil {
		if fqn(ing.Spec.Backend.ServiceName) == svcFQN {
			return true
		}
	}
	for _, rules := range ing.Spec.Rules {
		if rules.HTTP != nil {
			for _, svc := range rules.HTTP.Paths {
				if fqn(svc.Backend.ServiceName) == svcFQN {
					return true
				}
			}
		}
		for _, svc := range rules.TCP {
			if fqn(svc.Backend.ServiceName) == svcFQN {
				return true
			}
		}
	}
	return false
}
