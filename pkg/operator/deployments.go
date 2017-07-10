package operator

import (
	"strings"

	acrt "github.com/appscode/go/runtime"
	"github.com/appscode/go/types"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (c *Operator) WatchDeployments() {
	defer acrt.HandleCrash()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.KubeClient.ExtensionsV1beta1().Deployments(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.KubeClient.ExtensionsV1beta1().Deployments(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&extensions.Deployment{},
		c.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				if deployment, ok := obj.(*extensions.Deployment); ok {
					log.Infof("Deployment %s@%s deleted", deployment.Name, deployment.Namespace)
					c.restoreDeploymentIfRequired(deployment)
				}
			},
		},
	)
	ctrl.Run(wait.NeverStop)
}

func (w *Operator) restoreDeploymentIfRequired(deployment *extensions.Deployment) error {
	if deployment.Annotations == nil {
		return nil
	}

	sourceName, sourceNameFound := deployment.Annotations[api.OriginName]
	sourceType, sourceTypeFound := deployment.Annotations[api.OriginAPISchema]
	noAnnotationResource := false
	if !sourceNameFound && !sourceTypeFound {
		// Lets Check if those are old Ingress resource
		if strings.HasPrefix(deployment.Name, api.VoyagerPrefix) {
			noAnnotationResource = true
			sourceName, sourceNameFound = deployment.Name[len(api.VoyagerPrefix):], true
		}
	}
	if !sourceNameFound {
		return nil
	}

	// deleted resource have source reference
	var ingressErr error
	var engress *api.Ingress
	if sourceType == api.APISchemaIngress {
		var resource *extensions.Ingress
		resource, ingressErr = w.KubeClient.ExtensionsV1beta1().Ingresses(deployment.Namespace).Get(sourceName, metav1.GetOptions{})
		if ingressErr == nil {
			engress, _ = api.NewEngressFromIngress(resource)
		}
	} else if sourceType == api.APISchemaEngress {
		engress, ingressErr = w.ExtClient.Ingresses(deployment.Namespace).Get(sourceName)
	} else if !sourceTypeFound {
		var resource *extensions.Ingress
		resource, ingressErr = w.KubeClient.ExtensionsV1beta1().Ingresses(deployment.Namespace).Get(sourceName, metav1.GetOptions{})
		if ingressErr == nil {
			engress, _ = api.NewEngressFromIngress(resource)
		} else {
			engress, ingressErr = w.ExtClient.Ingresses(deployment.Namespace).Get(sourceName)
		}
	}
	if ingressErr != nil {
		return ingressErr
	}

	// Ingress Still exists, restore resource
	log.Infof("Deployment %s@%s requires restoration", deployment.Name, deployment.Namespace)
	deployment.Spec.Paused = false
	if types.Int32(deployment.Spec.Replicas) < 1 {
		deployment.Spec.Replicas = types.Int32P(engress.Replicas())
	}
	deployment.SelfLink = ""
	deployment.ResourceVersion = ""
	if noAnnotationResource {
		// Old resource and annotations are missing so we need to add the annotations
		if deployment.Annotations == nil {
			deployment.Annotations = make(map[string]string)
		}
		deployment.Annotations[api.OriginAPISchema] = engress.APISchema()
		deployment.Annotations[api.OriginName] = sourceName
	}

	_, err := w.KubeClient.ExtensionsV1beta1().Deployments(deployment.Namespace).Create(deployment)
	return err
}
