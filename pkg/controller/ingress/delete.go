package ingress

import (
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/log"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/selection"
	"k8s.io/kubernetes/pkg/util/sets"
)

func (lbc *EngressController) Delete() error {
	log.Infoln("Starting deleting lb. got engress with", lbc.Resource.ObjectMeta)
	err := lbc.parse()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	err = lbc.deleteLB()
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

func (lbc *EngressController) deleteLB() error {
	if lbc.Options.LBType == LBTypeDaemon || lbc.Options.LBType == LBTypeHostPort {
		err := lbc.deleteHostPortPods()
		if err != nil {
			return errors.FromErr(err).Err()
		}
	} else if lbc.Options.LBType == LBTypeNodePort {
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
	return lbc.deleteLBSvc()
}

func (lbc *EngressController) deleteLBSvc() error {
	svc, err := lbc.KubeClient.Core().Services(lbc.Resource.Namespace).Get(lbc.OffshootName())
	if err == nil {
		// delete service
		err = lbc.KubeClient.Core().Services(lbc.Resource.Namespace).Delete(VoyagerPrefix+lbc.Resource.Name, &kapi.DeleteOptions{})
		if err != nil {
			return errors.FromErr(err).Err()
		}

		if (lbc.Options.LBType == LBTypeDaemon || lbc.Options.LBType == LBTypeHostPort) && lbc.CloudManager != nil {
			if fw, ok := lbc.CloudManager.Firewall(); ok {
				convertedSvc := &kapi.Service{}
				kapi.Scheme.Convert(svc, convertedSvc, nil)
				err = fw.EnsureFirewallDeleted(convertedSvc)
				if err != nil {
					return errors.FromErr(err).Err()
				}
			}
		}

		if svc.Spec.Type == kapi.ServiceTypeNodePort && lbc.CloudManager != nil {
			if lb, ok := lbc.CloudManager.LoadBalancer(); ok {
				convertedSvc := &kapi.Service{}
				kapi.Scheme.Convert(svc, convertedSvc, nil)
				err = lb.EnsureLoadBalancerDeleted(lbc.Options.ClusterName, convertedSvc)
				if err != nil {
					return errors.FromErr(err).Err()
				}
			}
		}
	}
	return nil
}

func (lbc *EngressController) deleteHostPortPods() error {
	d, err := lbc.KubeClient.Extensions().DaemonSets(lbc.Resource.Namespace).Get(lbc.OffshootName())
	if err != nil {
		return nil
	}
	err = lbc.KubeClient.Extensions().DaemonSets(lbc.Resource.Namespace).Delete(VoyagerPrefix+lbc.Resource.Name, &kapi.DeleteOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	lbc.deletePodsForSelector(d.Spec.Selector.MatchLabels)
	return nil
}

func (lbc *EngressController) deleteNodePortPods() error {
	d, err := lbc.KubeClient.Extensions().Deployments(lbc.Resource.Namespace).Get(lbc.OffshootName())
	if err != nil {
		return errors.FromErr(err).Err()
	}
	// resize the controller to zero (effectively deleting all pods) before deleting it.
	d.Spec.Replicas = 0
	_, err = lbc.KubeClient.Extensions().Deployments(lbc.Resource.Namespace).Update(d)
	if err != nil {
		return errors.FromErr(err).Err()
	}

	log.Debugln("Waiting before delete the RC")
	time.Sleep(time.Second * 5)
	// if update failed still trying to delete the controller.
	falseVar := false
	err = lbc.KubeClient.Extensions().Deployments(lbc.Resource.Namespace).Delete(VoyagerPrefix+lbc.Resource.Name, &kapi.DeleteOptions{
		OrphanDependents: &falseVar,
	})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	lbc.deletePodsForSelector(d.Spec.Selector.MatchLabels)
	return nil
}

// Deprecated, creating pods using RC is now deprecated.
func (lbc *EngressController) deleteResidualPods() error {
	rc, err := lbc.KubeClient.Core().ReplicationControllers(lbc.Resource.Namespace).Get(lbc.OffshootName())
	if err != nil {
		log.Warningln(err)
		return err
	}
	// resize the controller to zero (effectively deleting all pods) before deleting it.
	rc.Spec.Replicas = 0
	_, err = lbc.KubeClient.Core().ReplicationControllers(lbc.Resource.Namespace).Update(rc)
	if err != nil {
		log.Warningln(err)
		return err
	}

	log.Debugln("Waiting before delete the RC")
	time.Sleep(time.Second * 5)
	// if update failed still trying to delete the controller.
	falseVar := false
	err = lbc.KubeClient.Core().ReplicationControllers(lbc.Resource.Namespace).Delete(VoyagerPrefix+lbc.Resource.Name, &kapi.DeleteOptions{
		OrphanDependents: &falseVar,
	})
	if err != nil {
		log.Warningln(err)
		return err
	}
	lbc.deletePodsForSelector(rc.Spec.Selector)
	return nil
}

func (lbc *EngressController) deleteConfigMap() error {
	err := lbc.KubeClient.Core().ConfigMaps(lbc.Resource.Namespace).Delete(VoyagerPrefix+lbc.Resource.Name, &kapi.DeleteOptions{})
	if err != nil {
		return errors.FromErr(err).Err()
	}
	return nil
}

// Ensures deleting all pods if its still exits.
func (lbc *EngressController) deletePodsForSelector(s map[string]string) {
	lb := labels.NewSelector()
	for key, value := range s {
		s := sets.NewString(value)
		ls, err := labels.NewRequirement(key, selection.Equals, s.List())
		if err != nil {
			log.Warningln(err)
		}
		lb = lb.Add(*ls)
	}
	pods, err := lbc.KubeClient.Core().Pods(lbc.Resource.Namespace).List(kapi.ListOptions{
		LabelSelector: lb,
	})

	if len(pods.Items) > 1 {
		log.Warningln("load balancer delete request, pods are greater than one.")
	}
	gracePeriods := int64(0)
	for _, pod := range pods.Items {
		err = lbc.KubeClient.Core().Pods(lbc.Resource.Namespace).Delete(pod.Name, &kapi.DeleteOptions{
			GracePeriodSeconds: &gracePeriods,
		})
		if err != nil {
			log.Warningln(err)
		}
	}
}

func (lbc *EngressController) ensureStatsServiceDeleted() {
	err := lbc.KubeClient.Core().
		Services(lbc.Resource.Namespace).
		Delete(
			lbc.Options.annotations.StatsServiceName(lbc.Resource.GetName()),
			&kapi.DeleteOptions{},
		)
	if err != nil {
		log.Errorln("Failed to delete Stats service", err)
	}
}
