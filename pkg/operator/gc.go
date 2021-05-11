/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package operator

import (
	"context"

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
)

func (op *Operator) PurgeOffshootsWithDeprecatedLabels() error {
	ingresses, err := op.KubeClient.NetworkingV1beta1().Ingresses(op.WatchNamespace).List(context.TODO(), metav1.ListOptions{})
	if err == nil {
		for _, ing := range ingresses.Items {
			if getLBType(ing.Annotations) == api.LBTypeHostPort {
				err = op.KubeClient.AppsV1().DaemonSets(ing.Namespace).DeleteCollection(
					context.TODO(),
					metav1.DeleteOptions{},
					metav1.ListOptions{
						LabelSelector: labels.SelectorFromSet(deprecatedLabelsFor(ing.Name)).String(),
					})
				if err != nil {
					return err
				}
			} else {
				err = op.KubeClient.AppsV1().Deployments(ing.Namespace).DeleteCollection(
					context.TODO(),
					metav1.DeleteOptions{},
					metav1.ListOptions{
						LabelSelector: labels.SelectorFromSet(deprecatedLabelsFor(ing.Name)).String(),
					})
				if err != nil {
					return err
				}
			}

			if services, err := op.KubeClient.CoreV1().Services(ing.Namespace).List(context.TODO(), metav1.ListOptions{
				LabelSelector: labels.SelectorFromSet(deprecatedLabelsFor(ing.Name)).String(),
			}); err == nil {
				for _, svc := range services.Items {
					err = op.KubeClient.CoreV1().Services(ing.Namespace).Delete(context.TODO(), svc.Name, metav1.DeleteOptions{})
					if err != nil {
						return err
					}
				}
			}
		}
		return err
	}

	engresses, err := op.VoyagerClient.VoyagerV1beta1().Ingresses(op.WatchNamespace).List(context.TODO(), metav1.ListOptions{})
	if err == nil {
		for _, ing := range engresses.Items {
			if getLBType(ing.Annotations) == api.LBTypeHostPort {
				err = op.KubeClient.AppsV1().DaemonSets(ing.Namespace).DeleteCollection(
					context.TODO(),
					metav1.DeleteOptions{},
					metav1.ListOptions{
						LabelSelector: labels.SelectorFromSet(deprecatedLabelsFor(ing.Name)).String(),
					})
				if err != nil {
					return err
				}
			} else {
				err = op.KubeClient.AppsV1().Deployments(ing.Namespace).DeleteCollection(
					context.TODO(),
					metav1.DeleteOptions{},
					metav1.ListOptions{
						LabelSelector: labels.SelectorFromSet(deprecatedLabelsFor(ing.Name)).String(),
					})
				if err != nil {
					return err
				}
			}

			if services, err := op.KubeClient.CoreV1().Services(ing.Namespace).List(context.TODO(), metav1.ListOptions{
				LabelSelector: labels.SelectorFromSet(deprecatedLabelsFor(ing.Name)).String(),
			}); err == nil {
				for _, svc := range services.Items {
					err = op.KubeClient.CoreV1().Services(ing.Namespace).Delete(context.TODO(), svc.Name, metav1.DeleteOptions{})
					if err != nil {
						return err
					}
				}
			}
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
	ingresses, err := op.KubeClient.NetworkingV1beta1().Ingresses(op.WatchNamespace).List(context.TODO(), metav1.ListOptions{})
	if err == nil {
		for _, ing := range ingresses.Items {
			if getLBType(ing.Annotations) == api.LBTypeHostPort {
				name := api.VoyagerPrefix + ing.Name
				klog.Infof("Deleting DaemonSet %s/%s", ing.Namespace, name)
				err = op.KubeClient.AppsV1().DaemonSets(ing.Namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
				if err != nil {
					return err
				}
			}
		}
		return err
	}

	engresses, err := op.VoyagerClient.VoyagerV1beta1().Ingresses(op.WatchNamespace).List(context.TODO(), metav1.ListOptions{})
	if err == nil {
		for _, ing := range engresses.Items {
			if getLBType(ing.Annotations) == api.LBTypeHostPort {
				name := api.VoyagerPrefix + ing.Name
				if ds, err := op.KubeClient.AppsV1().DaemonSets(ing.Namespace).Get(context.TODO(), name, metav1.GetOptions{}); err == nil {
					if ds.Spec.Template.Spec.Affinity != nil && ing.Spec.Affinity == nil {
						klog.Infof("Updating Ingress %s/%s to add `spec.affinity`", ing.Namespace, ing.Name)
						ing.Spec.Affinity = ds.Spec.Template.Spec.Affinity
						_, err = op.VoyagerClient.VoyagerV1beta1().Ingresses(ing.Namespace).Update(context.TODO(), &ing, metav1.UpdateOptions{})
						if err != nil {
							return err
						}
					}
					klog.Infof("Deleting DaemonSet %s/%s", ing.Namespace, name)
					err = op.KubeClient.AppsV1().DaemonSets(ing.Namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
					if err != nil {
						return err
					}
				}
			}
		}
		return err
	}
	return nil
}
