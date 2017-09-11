package operator

import (
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func (op *Operator) PurgeOffshootsWithDeprecatedLabels() error {
	ingresses, err := op.KubeClient.ExtensionsV1beta1().Ingresses(apiv1.NamespaceAll).List(metav1.ListOptions{})
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

	engresses, err := op.ExtClient.Ingresses(apiv1.NamespaceAll).List(metav1.ListOptions{})
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
