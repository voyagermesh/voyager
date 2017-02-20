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
	log.Infoln("Starting deleting lb. got engress with", lbc.Config.ObjectMeta)
	err := lbc.parse()
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}
	err = lbc.deleteLB()
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}
	err = lbc.deleteConfigMap()
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}
	return nil
}

func (lbc *EngressController) deleteLB() error {
	var err error
	if lbc.Options.LBType == LBDaemon {
		err = lbc.deleteDaemonSets()
	} else {
		err = lbc.deleteRc()
	}
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}
	svc, err := lbc.KubeClient.Core().Services(lbc.Config.Namespace).Get(VoyagerPrefix + lbc.Config.Name)
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}

	// delete service
	err = lbc.KubeClient.Core().Services(lbc.Config.Namespace).Delete(VoyagerPrefix+lbc.Config.Name, &kapi.DeleteOptions{})
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}

	if lbc.Options.LBType == LBDaemon && lbc.CloudManager != nil {
		if fw, ok := lbc.CloudManager.Firewall(); ok {
			convertedSvc := &kapi.Service{}
			kapi.Scheme.Convert(svc, convertedSvc, nil)
			err = fw.EnsureFirewallDeleted(convertedSvc)
			if err != nil {
				return errors.New().WithCause(err).Internal()
			}
		}
	}

	if svc.Spec.Type == kapi.ServiceTypeNodePort && lbc.CloudManager != nil {
		if lb, ok := lbc.CloudManager.LoadBalancer(); ok {
			convertedSvc := &kapi.Service{}
			kapi.Scheme.Convert(svc, convertedSvc, nil)
			err = lb.EnsureLoadBalancerDeleted(lbc.Options.ClusterName, convertedSvc)
			if err != nil {
				return errors.New().WithCause(err).Internal()
			}
		}
	}
	return nil
}

func (lbc *EngressController) deleteDaemonSets() error {
	d, err := lbc.KubeClient.Extensions().DaemonSets(lbc.Config.Namespace).Get(VoyagerPrefix + lbc.Config.Name)
	if err != nil {
		return nil
	}
	err = lbc.KubeClient.Extensions().DaemonSets(lbc.Config.Namespace).Delete(VoyagerPrefix+lbc.Config.Name, &kapi.DeleteOptions{})
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}

	lb := labels.NewSelector()
	for key, value := range d.Spec.Selector.MatchLabels {
		s := sets.NewString(value)
		ls, err := labels.NewRequirement(key, selection.Equals, s.List())
		if err != nil {
			return errors.New().WithCause(err).Internal()
		}
		lb = lb.Add(*ls)
	}
	pods, err := lbc.KubeClient.Core().Pods(lbc.Config.Namespace).List(kapi.ListOptions{
		LabelSelector: lb,
	})

	if len(pods.Items) > 1 {
		log.Warningln("load balancer delete request, pods are gretter than one.")
	}
	for _, pod := range pods.Items {
		err = lbc.KubeClient.Core().Pods(lbc.Config.Namespace).Delete(pod.Name, &kapi.DeleteOptions{})
		if err != nil {
			return errors.New().WithCause(err).Internal()
		}
	}
	return nil
}

func (lbc *EngressController) deleteRc() error {
	rc, err := lbc.KubeClient.Core().ReplicationControllers(lbc.Config.Namespace).Get(VoyagerPrefix + lbc.Config.Name)
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}
	// resize the controller to zero (effectively deleting all pods) before deleting it.
	rc.Spec.Replicas = 0
	_, err = lbc.KubeClient.Core().ReplicationControllers(lbc.Config.Namespace).Update(rc)
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}

	log.Debugln("Waiting before delete the RC")
	time.Sleep(time.Second * 10)
	// if update failed still trying to delete the controller.
	falseVar := false
	err = lbc.KubeClient.Core().ReplicationControllers(lbc.Config.Namespace).Delete(VoyagerPrefix+lbc.Config.Name, &kapi.DeleteOptions{
		OrphanDependents: &falseVar,
	})
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}
	return nil
}

func (lbc *EngressController) deleteConfigMap() error {
	err := lbc.KubeClient.Core().ConfigMaps(lbc.Config.Namespace).Delete(VoyagerPrefix+lbc.Config.Name, &kapi.DeleteOptions{})
	if err != nil {
		return errors.New().WithCause(err).Internal()
	}
	return nil
}
