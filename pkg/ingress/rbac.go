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

	api "voyagermesh.dev/voyager/apis/voyager/v1beta1"
	"voyagermesh.dev/voyager/pkg/eventer"

	core "k8s.io/api/core/v1"
	extensions "k8s.io/api/networking/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kutil "kmodules.xyz/client-go"
	core_util "kmodules.xyz/client-go/core/v1"
	rbac_util "kmodules.xyz/client-go/rbac/v1"
)

func (c *controller) reconcileRBAC() error {
	if vt, err := c.reconcileServiceAccount(); err != nil {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonIngressRBACFailed,
			"Failed to reconcile ServiceAccount %s. Reason: %v",
			c.Ingress.OffshootName(),
			err,
		)
		return err
	} else if vt != kutil.VerbUnchanged {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonIngressRBACSuccessful,
			"Successfully %s ServiceAccount %s",
			vt,
			c.Ingress.OffshootName(),
		)
	}

	if vt, err := c.reconcileRoles(); err != nil {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonIngressRBACFailed,
			"Failed to reconcile Role %s. Reason: %v",
			c.Ingress.OffshootName(),
			err,
		)
		return err
	} else if vt != kutil.VerbUnchanged {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonIngressRBACSuccessful,
			"Successfully %s Role %s",
			vt,
			c.Ingress.OffshootName(),
		)
	}

	if vt, err := c.reconcileRoleBinding(); err != nil {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonIngressRBACFailed,
			"Failed to reconcile RoleBinding %s. Reason: %v",
			c.Ingress.OffshootName(),
			err,
		)
		return err
	} else if vt != kutil.VerbUnchanged {
		c.recorder.Eventf(
			c.Ingress.ObjectReference(),
			core.EventTypeNormal,
			eventer.EventReasonIngressRBACSuccessful,
			"Successfully %s RoleBinding %s",
			vt,
			c.Ingress.OffshootName(),
		)
	}
	return nil
}

func (c *controller) reconcileServiceAccount() (kutil.VerbType, error) {
	meta := metav1.ObjectMeta{
		Namespace: c.Ingress.Namespace,
		Name:      c.Ingress.OffshootName(),
	}
	_, vt, err := core_util.CreateOrPatchServiceAccount(context.TODO(), c.KubeClient, meta, func(in *core.ServiceAccount) *core.ServiceAccount {
		core_util.EnsureOwnerReference(in, metav1.NewControllerRef(c.Ingress, api.SchemeGroupVersion.WithKind(api.ResourceKindIngress)))

		in.Labels = c.Ingress.OffshootLabels()
		if in.Annotations == nil {
			in.Annotations = map[string]string{}
		}
		in.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		in.Annotations[api.OriginName] = c.Ingress.GetName()
		return in
	}, metav1.PatchOptions{})
	return vt, err
}

func (c *controller) reconcileRoles() (kutil.VerbType, error) {
	meta := metav1.ObjectMeta{
		Namespace: c.Ingress.Namespace,
		Name:      c.Ingress.OffshootName(),
	}
	_, vt, err := rbac_util.CreateOrPatchRole(context.TODO(), c.KubeClient, meta, func(in *rbac.Role) *rbac.Role {
		core_util.EnsureOwnerReference(in, metav1.NewControllerRef(c.Ingress, api.SchemeGroupVersion.WithKind(api.ResourceKindIngress)))

		in.Labels = c.Ingress.OffshootLabels()
		if in.Annotations == nil {
			in.Annotations = map[string]string{}
		}
		in.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		in.Annotations[api.OriginName] = c.Ingress.GetName()

		in.Rules = []rbac.PolicyRule{
			{
				APIGroups: []string{core.GroupName},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list", "watch"},
			},
			// We need to have those permission for secret mounter
			{
				APIGroups: []string{core.GroupName},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{api.SchemeGroupVersion.Group},
				Resources: []string{"ingresses", "certificates"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{extensions.GroupName},
				Resources: []string{"ingresses"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{core.GroupName},
				Resources: []string{"events"},
				Verbs:     []string{"create"},
			},
		}
		return in
	}, metav1.PatchOptions{})
	return vt, err
}

func (c *controller) reconcileRoleBinding() (kutil.VerbType, error) {
	meta := metav1.ObjectMeta{
		Namespace: c.Ingress.Namespace,
		Name:      c.Ingress.OffshootName(),
	}
	_, vt, err := rbac_util.CreateOrPatchRoleBinding(context.TODO(), c.KubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		core_util.EnsureOwnerReference(in, metav1.NewControllerRef(c.Ingress, api.SchemeGroupVersion.WithKind(api.ResourceKindIngress)))

		in.Labels = c.Ingress.OffshootLabels()
		if in.Annotations == nil {
			in.Annotations = map[string]string{}
		}
		in.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		in.Annotations[api.OriginName] = c.Ingress.GetName()

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "Role",
			Name:     c.Ingress.OffshootName(),
		}
		in.Subjects = []rbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      c.Ingress.OffshootName(),
				Namespace: c.Ingress.Namespace,
			},
		}
		return in
	}, metav1.PatchOptions{})
	return vt, err
}

func (c *controller) ensureRoleBindingDeleted() error {
	c.logger.Info("Deleting", "RoleBinding", c.Ingress.OffshootName())
	return c.KubeClient.RbacV1().
		RoleBindings(c.Ingress.Namespace).
		Delete(context.TODO(), c.Ingress.OffshootName(), metav1.DeleteOptions{})
}

func (c *controller) ensureRolesDeleted() error {
	c.logger.Info("Deleting", "Role", c.Ingress.OffshootName())
	return c.KubeClient.RbacV1().
		Roles(c.Ingress.Namespace).
		Delete(context.TODO(), c.Ingress.OffshootName(), metav1.DeleteOptions{})
}

func (c *controller) ensureServiceAccountDeleted() error {
	c.logger.Info("Deleting", "ServiceAccount", c.Ingress.OffshootName())
	return c.KubeClient.CoreV1().
		ServiceAccounts(c.Ingress.Namespace).
		Delete(context.TODO(), c.Ingress.OffshootName(), metav1.DeleteOptions{})
}
