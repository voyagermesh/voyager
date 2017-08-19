package operator

import (
	"encoding/json"
	"fmt"

	"github.com/appscode/errors"
	"github.com/appscode/log"
	tapi "github.com/appscode/voyager/api"
	_ "github.com/appscode/voyager/api/install"
	"github.com/appscode/voyager/pkg/ingress"
	_ "github.com/appscode/voyager/third_party/forked/cloudprovider/providers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	core_listers "k8s.io/client-go/listers/core/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (op *Operator) initServiceWatcher() cache.Controller {
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return op.KubeClient.CoreV1().Services(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return op.KubeClient.CoreV1().Services(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	indexer, informer := cache.NewIndexerInformer(lw,
		&apiv1.Service{},
		op.Opt.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if svc, ok := obj.(*apiv1.Service); ok {
					log.Infof("Service %s@%s added", svc.Name, svc.Namespace)
					op.updateHAProxyConfig(svc)
				}
			},
			DeleteFunc: func(obj interface{}) {
				if svc, ok := obj.(*apiv1.Service); ok {
					log.Infof("Service %s@%s deleted", svc.Name, svc.Namespace)

					if restored, err := op.restoreServiceIfRequired(svc); err == nil && restored {
						return
					}
					op.updateHAProxyConfig(svc)
				}
			},
		},
		cache.Indexers{},
	)
	op.ServiceLister = core_listers.NewServiceLister(indexer)
	return informer
}

func (op *Operator) restoreServiceIfRequired(svc *apiv1.Service) (bool, error) {
	if svc.Annotations == nil {
		return false, nil
	}

	// deleted resource have source reference
	engress, err := op.findOrigin(svc.ObjectMeta)
	if err != nil {
		return false, err
	}

	if svc.Name == engress.StatsServiceName() && !engress.Stats() {
		return false, nil
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
	svc.Annotations[tapi.OriginAPISchema] = engress.APISchema()
	svc.Annotations[tapi.OriginName] = engress.Name

	_, err = op.KubeClient.CoreV1().Services(svc.Namespace).Create(svc)
	return true, err
}

func (op *Operator) findOrigin(meta metav1.ObjectMeta) (*tapi.Ingress, error) {
	if meta.Annotations == nil {
		return nil, nil
	}

	sourceName, sourceNameFound := meta.Annotations[tapi.OriginName]
	sourceType, sourceTypeFound := meta.Annotations[tapi.OriginAPISchema]
	if !sourceNameFound && !sourceTypeFound {
		return nil, errors.New("No Types or Name found").Err()
	}

	if sourceType == tapi.APISchemaIngress {
		ingress, err := op.KubeClient.ExtensionsV1beta1().Ingresses(meta.Namespace).Get(sourceName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return tapi.NewEngressFromIngress(ingress)
	} else if sourceType == tapi.APISchemaEngress {
		return op.ExtClient.Ingresses(meta.Namespace).Get(sourceName)
	}
	return nil, fmt.Errorf("Unknown ingress type %s", sourceType)
}

func (op *Operator) updateHAProxyConfig(svc *apiv1.Service) error {
	ing, err := op.KubeClient.ExtensionsV1beta1().Ingresses(svc.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	eng, err := op.ExtClient.Ingresses(apiv1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	items := make([]tapi.Ingress, len(ing.Items))
	for i, item := range ing.Items {
		e, err := tapi.NewEngressFromIngress(item)
		if err != nil {
			continue
		}
		items[i] = *e
	}
	items = append(items, eng.Items...)

	log.Infoln("Updating All Ingress, got total", len(items))
	for i := range items {
		engress := &items[i]
		if engress.ShouldHandleIngress(op.Opt.IngressClass) {
			log.Infoln("Checking for service", svc, "to be used to load balance via ingress", engress.Name, engress.Namespace)
			if engress.HasBackendService(svc.Name, svc.Namespace) {
				ctrl := ingress.NewController(op.KubeClient, op.ExtClient, op.PromClient, op.ServiceLister, op.EndpointsLister, op.Opt, engress)
				if ctrl.IsExists() {
					// Loadbalancer resource for this ingress is found in its place,
					// so no need to create the resources. First trying to update
					// the configMap only for the rules.
					// In case of any failure in soft update we will make hard update
					// to the resource. If hard update encounters errors then we will
					// recreate the resource from scratch.
					log.Infoln("Loadbalancer is exists, trying to update")
					cfgErr := ctrl.Update(0, nil)
					if cfgErr != nil {
						log.Infoln("Loadbalancer update failed", cfgErr)
					}
				} else {
					// This LB should be there. If it is no there. we should create it
					log.Infoln("Loadbalancer is not found, recreating with a cleanup")
					ctrl.Create()
				}
				op.ensureEgressAnnotations(engress, svc)
			}
		}
	}
	return nil
}

func (op *Operator) ensureEgressAnnotations(r *tapi.Ingress, svc *apiv1.Service) {
	if svc.Annotations == nil {
		svc.Annotations = make(map[string]string)
	}
	if r.HasBackendService(svc.Name, svc.Namespace) {
		list := make([]tapi.IngressRef, 0)
		val, ok := svc.Annotations[tapi.EgressPoints]
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
					list = append(list, tapi.IngressRef{
						APISchema: r.APISchema(),
						Name:      r.Name,
						Namespace: r.Namespace,
					})
				}
			}
		} else {
			list = append(list, tapi.IngressRef{
				APISchema: r.APISchema(),
				Name:      r.Name,
				Namespace: r.Namespace,
			})
		}

		data, err := json.Marshal(list)
		if err == nil {
			svc.Annotations[tapi.EgressPoints] = string(data)
		}
		op.KubeClient.CoreV1().Services(svc.Namespace).Update(svc)
		return
	}
	// Lets check if service still have the annotation for this ingress.
	val, ok := svc.Annotations[tapi.EgressPoints]
	if ok {
		list := make([]tapi.IngressRef, 0)
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
				svc.Annotations[tapi.EgressPoints] = string(data)
			}
		}
		op.KubeClient.CoreV1().Services(svc.Namespace).Update(svc)
	}
}
