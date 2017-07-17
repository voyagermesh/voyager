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

func (lbc *Controller) Delete() error {
	err := lbc.deleteLB()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	err = lbc.deleteConfigMap()
	if err != nil {
		return errors.FromErr(err).Err()
	}

	if lbc.Parsed.Stats {
		lbc.ensureStatsServiceDeleted()
	}

	return nil
}

func (lbc *Controller) deleteLB() error {
	if lbc.Ingress.LBType() == api.LBTypeHostPort {
		err := lbc.deleteHostPortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	} else if lbc.Ingress.LBType() == api.LBTypeNodePort {
		err := lbc.deleteNodePortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	} else {
		// Ignore Error.
		lbc.deleteResidualPods()
		err := lbc.deleteNodePortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	}

	monSpec, err := lbc.Ingress.MonitorSpec()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	if monSpec != nil && monSpec.Prometheus != nil {
		ctrl := monitor.NewPrometheusController(lbc.KubeClient, lbc.PromClient)
		ctrl.DeleteMonitor(lbc.Ingress, monSpec)
	}
	return lbc.deleteLBSvc()
}

func (lbc *Controller) deleteLBSvc() error {
	svc, err := lbc.KubeClient.CoreV1().Services(lbc.Ingress.Namespace).Get(lbc.Ingress.OffshootName(), metav1.GetOptions{})
	if err == nil {
		// delete service
		err = lbc.KubeClient.CoreV1().Services(lbc.Ingress.Namespace).Delete(lbc.Ingress.OffshootName(), &metav1.DeleteOptions{})
		if err != nil {
			return errors.FromErr(err).Err()
		}

		if (lbc.Ingress.LBType() == api.LBTypeHostPort) && lbc.CloudManager != nil {
			if fw, ok := lbc.CloudManager.Firewall(); ok {
				err = fw.EnsureFirewallDeleted(svc)
				if err != nil {
					return errors.FromErr(err).Err()
				}
			}
		}
	}
	return nil
}

func (lbc *Controller) deleteHostPortPods() error {
	d, err := lbc.KubeClient.ExtensionsV1beta1().DaemonSets(lbc.Ingress.Namespace).Get(lbc.Ingress.OffshootName(), metav1.GetOptions{})
	if err != nil {
		return nil
	}
	err = lbc.KubeClient.ExtensionsV1beta1().DaemonSets(lbc.Ingress.Namespace).Delete(lbc.Ingress.OffshootName(), &metav1.DeleteOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	lbc.deletePodsForSelector(d.Spec.Selector.MatchLabels)
	return nil
}

func (lbc *Controller) deleteNodePortPods() error {
	d, err := lbc.KubeClient.ExtensionsV1beta1().Deployments(lbc.Ingress.Namespace).Get(lbc.Ingress.OffshootName(), metav1.GetOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	// resize the controller to zero (effectively deleting all pods) before deleting it.
	d.Spec.Replicas = types.Int32P(0)
	_, err = lbc.KubeClient.ExtensionsV1beta1().Deployments(lbc.Ingress.Namespace).Update(d)
	if err != nil {
		return errors.FromErr(err).Err()
	}

	log.Debugln("Waiting before delete the RC")
	time.Sleep(time.Second * 5)
	// if update failed still trying to delete the controller.
	falseVar := false
	err = lbc.KubeClient.ExtensionsV1beta1().Deployments(lbc.Ingress.Namespace).Delete(lbc.Ingress.OffshootName(), &metav1.DeleteOptions{
		OrphanDependents: &falseVar,
	})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	lbc.deletePodsForSelector(d.Spec.Selector.MatchLabels)
	return nil
}

// Deprecated, creating pods using RC is now deprecated.
func (lbc *Controller) deleteResidualPods() error {
	rc, err := lbc.KubeClient.CoreV1().ReplicationControllers(lbc.Ingress.Namespace).Get(lbc.Ingress.OffshootName(), metav1.GetOptions{})
	if err != nil {
		log.Warningln(err)
		return err
	}
	// resize the controller to zero (effectively deleting all pods) before deleting it.
	rc.Spec.Replicas = types.Int32P(0)
	_, err = lbc.KubeClient.CoreV1().ReplicationControllers(lbc.Ingress.Namespace).Update(rc)
	if err != nil {
		log.Warningln(err)
		return err
	}

	log.Debugln("Waiting before delete the RC")
	time.Sleep(time.Second * 5)
	// if update failed still trying to delete the controller.
	falseVar := false
	err = lbc.KubeClient.CoreV1().ReplicationControllers(lbc.Ingress.Namespace).Delete(lbc.Ingress.OffshootName(), &metav1.DeleteOptions{
		OrphanDependents: &falseVar,
	})
	if err != nil {
		log.Warningln(err)
		return err
	}
	lbc.deletePodsForSelector(rc.Spec.Selector)
	return nil
}

func (lbc *Controller) deleteConfigMap() error {
	err := lbc.KubeClient.CoreV1().ConfigMaps(lbc.Ingress.Namespace).Delete(lbc.Ingress.OffshootName(), &metav1.DeleteOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}

// Ensures deleting all pods if its still exits.
func (lbc *Controller) deletePodsForSelector(s map[string]string) {
	lb := labels.NewSelector()
	for key, value := range s {
		s := sets.NewString(value)
		ls, err := labels.NewRequirement(key, selection.Equals, s.List())
		if err != nil {
			log.Warningln(err)
		}
		lb = lb.Add(*ls)
	}
	pods, err := lbc.KubeClient.CoreV1().Pods(lbc.Ingress.Namespace).List(metav1.ListOptions{
		LabelSelector: lb.String(),
	})

	if len(pods.Items) > 1 {
		log.Warningln("load balancer delete request, pods are greater than one.")
	}
	gracePeriods := int64(0)
	for _, pod := range pods.Items {
		err = lbc.KubeClient.CoreV1().Pods(lbc.Ingress.Namespace).Delete(pod.Name, &metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriods,
		})
		if err != nil {
			log.Warningln(err)
		}
	}
}

func (lbc *Controller) ensureStatsServiceDeleted() error {
	err := lbc.KubeClient.CoreV1().Services(lbc.Ingress.Namespace).Delete(
		lbc.Ingress.StatsServiceName(),
		&metav1.DeleteOptions{},
	)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}
