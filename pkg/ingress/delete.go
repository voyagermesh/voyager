package ingress

import (
	"github.com/appscode/errors"
	"github.com/appscode/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
)

func (c *controller) deleteConfigMap() error {
	err := c.KubeClient.CoreV1().ConfigMaps(c.Ingress.Namespace).Delete(c.Ingress.OffshootName(), &metav1.DeleteOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}

// Ensures deleting all pods if its still exits.
func (c *controller) deletePodsForSelector(s map[string]string) {
	lb := labels.NewSelector()
	for key, value := range s {
		s := sets.NewString(value)
		ls, err := labels.NewRequirement(key, selection.Equals, s.List())
		if err != nil {
			log.Warningln(err)
		}
		lb = lb.Add(*ls)
	}
	pods, err := c.KubeClient.CoreV1().Pods(c.Ingress.Namespace).List(metav1.ListOptions{
		LabelSelector: lb.String(),
	})

	if len(pods.Items) > 1 {
		log.Warningln("load balancer delete request, pods are greater than one.")
	}
	gracePeriods := int64(0)
	for _, pod := range pods.Items {
		err = c.KubeClient.CoreV1().Pods(c.Ingress.Namespace).Delete(pod.Name, &metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriods,
		})
		if err != nil {
			log.Warningln(err)
		}
	}
}

func (c *controller) ensureStatsServiceDeleted() error {
	err := c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Delete(
		c.Ingress.StatsServiceName(),
		&metav1.DeleteOptions{},
	)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}

func (c *controller) ensureRBACDeleted() error {
	if err := c.ensureRoleBindingDeleted(); err != nil {
		return errors.FromErr(err).Err()
	}

	if err := c.ensureRolesDeleted(); err != nil {
		return errors.FromErr(err).Err()
	}

	if err := c.ensureServiceAccountDeleted(); err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}
