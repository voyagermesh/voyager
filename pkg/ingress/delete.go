package ingress

import (
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/go/types"
	"github.com/appscode/log"
	"github.com/appscode/voyager/api"
	"github.com/appscode/voyager/pkg/monitor"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
)

func (c *Controller) Delete() error {
	err := c.deleteLB()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	err = c.deleteConfigMap()
	if err != nil {
		return errors.FromErr(err).Err()
	}

	if c.Opt.EnableRBAC {
		if err := c.ensureRBACDeleted(); err != nil {
			return err
		}
	}

	if c.Parsed.Stats {
		c.ensureStatsServiceDeleted()
	}

	return nil
}

func (c *Controller) deleteLB() error {
	if c.Ingress.LBType() == api.LBTypeHostPort {
		err := c.deleteHostPortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	} else if c.Ingress.LBType() == api.LBTypeNodePort {
		err := c.deleteNodePortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	} else {
		// Ignore Error.
		c.deleteResidualPods()
		err := c.deleteNodePortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}

	monSpec, err := c.Ingress.MonitorSpec()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	if monSpec != nil && monSpec.Prometheus != nil {
		ctrl := monitor.NewPrometheusController(c.KubeClient, c.PromClient)
		ctrl.DeleteMonitor(c.Ingress, monSpec)
	}
	return c.deleteLBSvc()
}

func (c *Controller) deleteLBSvc() error {
	svc, err := c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
	if err == nil {
		// delete service
		err = c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Delete(c.Ingress.OffshootName(), &metav1.DeleteOptions{})
		if err != nil {
			return errors.FromErr(err).Err()
		}

		if (c.Ingress.LBType() == api.LBTypeHostPort) && c.CloudManager != nil {
			if fw, ok := c.CloudManager.Firewall(); ok {
				err = fw.EnsureFirewallDeleted(svc)
				if err != nil {
					return errors.FromErr(err).Err()
				}
			}
		}
	}
	return nil
}

func (c *Controller) deleteHostPortPods() error {
	d, err := c.KubeClient.ExtensionsV1beta1().DaemonSets(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
	if err != nil {
		return nil
	}
	err = c.KubeClient.ExtensionsV1beta1().DaemonSets(c.Ingress.Namespace).Delete(c.Ingress.OffshootName(), &metav1.DeleteOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	c.deletePodsForSelector(d.Spec.Selector.MatchLabels)
	return nil
}

func (c *Controller) deleteNodePortPods() error {
	d, err := c.KubeClient.ExtensionsV1beta1().Deployments(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	// resize the controller to zero (effectively deleting all pods) before deleting it.
	d.Spec.Replicas = types.Int32P(0)
	_, err = c.KubeClient.ExtensionsV1beta1().Deployments(c.Ingress.Namespace).Update(d)

	log.Debugln("Waiting before delete the RC")
	time.Sleep(time.Second * 5)
	// if update failed still trying to delete the controller.
	falseVar := false
	err = c.KubeClient.ExtensionsV1beta1().Deployments(c.Ingress.Namespace).Delete(c.Ingress.OffshootName(), &metav1.DeleteOptions{
		OrphanDependents: &falseVar,
	})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	c.deletePodsForSelector(d.Spec.Selector.MatchLabels)
	return nil
}

// Deprecated, creating pods using RC is now deprecated.
func (c *Controller) deleteResidualPods() error {
	rc, err := c.KubeClient.CoreV1().ReplicationControllers(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
	if err != nil {
		log.Warningln(err)
		return err
	}
	// resize the controller to zero (effectively deleting all pods) before deleting it.
	rc.Spec.Replicas = types.Int32P(0)
	_, err = c.KubeClient.CoreV1().ReplicationControllers(c.Ingress.Namespace).Update(rc)
	if err != nil {
		log.Warningln(err)
		return err
	}

	log.Debugln("Waiting before delete the RC")
	time.Sleep(time.Second * 5)
	// if update failed still trying to delete the controller.
	falseVar := false
	err = c.KubeClient.CoreV1().ReplicationControllers(c.Ingress.Namespace).Delete(c.Ingress.OffshootName(), &metav1.DeleteOptions{
		OrphanDependents: &falseVar,
	})
	if err != nil {
		log.Warningln(err)
		return err
	}
	c.deletePodsForSelector(rc.Spec.Selector)
	return nil
}

func (c *Controller) deleteConfigMap() error {
	err := c.KubeClient.CoreV1().ConfigMaps(c.Ingress.Namespace).Delete(c.Ingress.OffshootName(), &metav1.DeleteOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}

// Ensures deleting all pods if its still exits.
func (c *Controller) deletePodsForSelector(s map[string]string) {
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

func (c *Controller) ensureStatsServiceDeleted() error {
	err := c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Delete(
		c.Ingress.StatsServiceName(),
		&metav1.DeleteOptions{},
	)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}

func (c *Controller) ensureRBACDeleted() error {
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
