package operator

import (
	"github.com/appscode/go/log"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (op *Operator) PurgeOffshootsWithDeprecatedLabels() error {
	ingresses, err := op.KubeClient.ExtensionsV1beta1().Ingresses(op.Opt.WatchNamespace()).List(metav1.ListOptions{})
	if err == nil {
		for _, ing := range ingresses.Items {
			if getLBType(ing.Annotations) == api.LBTypeHostPort {
				op.KubeClient.ExtensionsV1beta1().DaemonSets(ing.Namespace).DeleteCollection(
					&metav1.DeleteOptions{},
					metav1.ListOptions{
						LabelSelector: labels.SelectorFromSet(deprecatedLabelsFor(ing.Name)).String(),
					})
			} else {
				op.KubeClient.ExtensionsV1beta1().Deployments(ing.Namespace).DeleteCollection(
					&metav1.DeleteOptions{},
					metav1.ListOptions{
						LabelSelector: labels.SelectorFromSet(deprecatedLabelsFor(ing.Name)).String(),
					})
			}

			op.KubeClient.CoreV1().Services(ing.Namespace).DeleteCollection(
				&metav1.DeleteOptions{},
				metav1.ListOptions{
					LabelSelector: labels.SelectorFromSet(deprecatedLabelsFor(ing.Name)).String(),
				})
		}
		return err
	}

	engresses, err := op.VoyagerClient.Ingresses(op.Opt.WatchNamespace()).List(metav1.ListOptions{})
	if err == nil {
		for _, ing := range engresses.Items {
			if getLBType(ing.Annotations) == api.LBTypeHostPort {
				op.KubeClient.ExtensionsV1beta1().DaemonSets(ing.Namespace).DeleteCollection(
					&metav1.DeleteOptions{},
					metav1.ListOptions{
						LabelSelector: labels.SelectorFromSet(deprecatedLabelsFor(ing.Name)).String(),
					})
			} else {
				op.KubeClient.ExtensionsV1beta1().Deployments(ing.Namespace).DeleteCollection(
					&metav1.DeleteOptions{},
					metav1.ListOptions{
						LabelSelector: labels.SelectorFromSet(deprecatedLabelsFor(ing.Name)).String(),
					})
			}

			op.KubeClient.CoreV1().Services(ing.Namespace).DeleteCollection(
				&metav1.DeleteOptions{},
				metav1.ListOptions{
					LabelSelector: labels.SelectorFromSet(deprecatedLabelsFor(ing.Name)).String(),
				})
		}
		return err
	}
	return nil
}

func getLBType(annotations map[string]string) string {
	if annotations == nil {
		return api.LBTypeLoadBalancer
	}
	if t, ok := annotations[api.LBType]; ok {
		return t
	}
	return api.LBTypeLoadBalancer
}

func deprecatedLabelsFor(name string) map[string]string {
	return map[string]string{
		"appType":     "ext-applbc-" + name,
		"type":        "ext-lbc-" + name,
		"target":      "eng-" + name,
		"meta":        "eng-" + name + "-applbc",
		"engressName": name,
	}
}

func (op *Operator) PurgeOffshootsDaemonSet() error {
	ingresses, err := op.KubeClient.ExtensionsV1beta1().Ingresses(op.Opt.WatchNamespace()).List(metav1.ListOptions{})
	if err == nil {
		for _, ing := range ingresses.Items {
			if getLBType(ing.Annotations) == api.LBTypeHostPort {
				name := api.VoyagerPrefix + ing.Name
				log.Infoln("Deleting DaemonSet %s/%s", ing.Namespace, name)
				op.KubeClient.ExtensionsV1beta1().DaemonSets(ing.Namespace).Delete(name, &metav1.DeleteOptions{})
			}
		}
		return err
	}

	engresses, err := op.VoyagerClient.Ingresses(op.Opt.WatchNamespace()).List(metav1.ListOptions{})
	if err == nil {
		for _, ing := range engresses.Items {
			if getLBType(ing.Annotations) == api.LBTypeHostPort {
				name := api.VoyagerPrefix + ing.Name
				if ds, err := op.KubeClient.ExtensionsV1beta1().DaemonSets(ing.Namespace).Get(name, metav1.GetOptions{}); err == nil {
					if ds.Spec.Template.Spec.Affinity != nil && ing.Spec.Affinity == nil {
						log.Infof("Updating Ingress %s/%s to add `spec.affinity`", ing.Namespace, ing.Name)
						ing.Spec.Affinity = ds.Spec.Template.Spec.Affinity
						_, err = op.VoyagerClient.Ingresses(ing.Namespace).Update(&ing)
						if err != nil {
							return err
						}
					}
					log.Infoln("Deleting DaemonSet %s/%s", ing.Namespace, name)
					op.KubeClient.ExtensionsV1beta1().DaemonSets(ing.Namespace).Delete(name, &metav1.DeleteOptions{})
				}
			}
		}
		return err
	}
	return nil
}
