package ingress

import (
	"github.com/appscode/go/log"
	"github.com/appscode/kube-mon/agents"
	mona "github.com/appscode/kube-mon/api"
	"github.com/appscode/kutil"
	core_util "github.com/appscode/kutil/core/v1"
	meta_util "github.com/appscode/kutil/meta"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *controller) ensureMonitoringAgent(monSpec *mona.AgentSpec) (kutil.VerbType, error) {
	agent := agents.New(monSpec.Agent, c.KubeClient, c.CRDClient, c.PromClient)

	// if agent-type changed, delete old agent
	// do this before applying new agent-type annotation
	// ignore err here
	if err := c.ensureMonitoringAgentDeleted(agent); err != nil {
		log.Warningf("failed to delete old monitoring agent, reason: %s", err)
	}

	// create/update new agent
	// set agent-type annotation to stat-service
	vt, err := agent.CreateOrUpdate(c.Ingress.StatsAccessor(), monSpec)
	if err == nil {
		err = c.setNewAgentType(agent.GetType())
	}
	return vt, err
}

func (c *controller) ensureMonitoringAgentDeleted(newAgent mona.Agent) error {
	if oldAgent, err := c.getOldAgent(); err != nil {
		if kerr.IsNotFound(err) {
			return nil
		}
		return err
	} else if newAgent == nil || oldAgent.GetType() != newAgent.GetType() { // delete old agent
		if _, err := oldAgent.Delete(c.Ingress.StatsAccessor()); err != nil {
			return err
		}
	}
	return nil
}

func (c *controller) getOldAgent() (mona.Agent, error) {
	// get stat service
	svc, err := c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Get(c.Ingress.StatsServiceName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	agentType, err := meta_util.GetStringValue(svc.Annotations, mona.KeyAgent)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get agent type")
	}
	return agents.New(mona.AgentType(agentType), c.KubeClient, c.CRDClient, c.PromClient), nil
}

func (c *controller) setNewAgentType(agentType mona.AgentType) error {
	svc, err := c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Get(c.Ingress.StatsServiceName(), metav1.GetOptions{})
	if err != nil {
		return errors.Errorf("failed to get stat service %s, reason: %s", c.Ingress.StatsServiceName(), err.Error())
	}
	_, _, err = core_util.PatchService(c.KubeClient, svc, func(in *core.Service) *core.Service {
		in.Annotations = core_util.UpsertMap(in.Annotations, map[string]string{
			mona.KeyAgent: string(agentType),
		})
		return in
	})
	return err
}
