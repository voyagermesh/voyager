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

package ingress

import (
	"context"

	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	kutil "kmodules.xyz/client-go"
	core_util "kmodules.xyz/client-go/core/v1"
	meta_util "kmodules.xyz/client-go/meta"
	"kmodules.xyz/monitoring-agent-api/agents"
	mona "kmodules.xyz/monitoring-agent-api/api/v1"
)

func (c *controller) ensureMonitoringAgent(monSpec *mona.AgentSpec) (kutil.VerbType, error) {
	agent := agents.New(monSpec.Agent, c.KubeClient, c.PromClient)

	// if agent-type changed, delete old agent
	// do this before applying new agent-type annotation
	// ignore err here
	if err := c.ensureMonitoringAgentDeleted(agent); err != nil {
		klog.Warningf("failed to delete old monitoring agent, reason: %s", err)
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
	svc, err := c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Get(context.TODO(), c.Ingress.StatsServiceName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	agentType, err := meta_util.GetStringValue(svc.Annotations, mona.KeyAgent)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get agent type")
	}
	return agents.New(mona.AgentType(agentType), c.KubeClient, c.PromClient), nil
}

func (c *controller) setNewAgentType(agentType mona.AgentType) error {
	svc, err := c.KubeClient.CoreV1().Services(c.Ingress.Namespace).Get(context.TODO(), c.Ingress.StatsServiceName(), metav1.GetOptions{})
	if err != nil {
		return errors.Errorf("failed to get stat service %s, reason: %s", c.Ingress.StatsServiceName(), err.Error())
	}
	_, _, err = core_util.PatchService(context.TODO(), c.KubeClient, svc, func(in *core.Service) *core.Service {
		in.Annotations = core_util.UpsertMap(in.Annotations, map[string]string{
			mona.KeyAgent: string(agentType),
		})
		return in
	}, metav1.PatchOptions{})
	return err
}
