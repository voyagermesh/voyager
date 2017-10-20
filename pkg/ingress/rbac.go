package ingress

import (
	core_util "github.com/appscode/kutil/core/v1"
	rbac_util "github.com/appscode/kutil/rbac/v1beta1"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	rbac "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *controller) ensureServiceAccount() error {
	meta := metav1.ObjectMeta{
		Namespace: c.Ingress.Namespace,
		Name:      c.Ingress.OffshootName(),
	}
	_, err := core_util.CreateOrPatchServiceAccount(c.KubeClient, meta, func(in *apiv1.ServiceAccount) *apiv1.ServiceAccount {
		in.ObjectMeta = c.ensureOwnerReference(in.ObjectMeta)

		if in.Annotations == nil {
			in.Annotations = map[string]string{}
		}
		in.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		in.Annotations[api.OriginName] = c.Ingress.GetName()
		return in
	})
	return err
}

func (c *controller) ensureRoles() error {
	meta := metav1.ObjectMeta{
		Namespace: c.Ingress.Namespace,
		Name:      c.Ingress.OffshootName(),
	}
	_, err := rbac_util.CreateOrPatchRole(c.KubeClient, meta, func(in *rbac.Role) *rbac.Role {
		in.ObjectMeta = c.ensureOwnerReference(in.ObjectMeta)

		if in.Annotations == nil {
			in.Annotations = map[string]string{}
		}
		in.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		in.Annotations[api.OriginName] = c.Ingress.GetName()

		in.Rules = []rbac.PolicyRule{
			{
				APIGroups: []string{apiv1.GroupName},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list", "watch"},
			},
			// We need to have those permission for secret mounter
			{
				APIGroups: []string{apiv1.GroupName},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{api.SchemeGroupVersion.Group},
				Resources: []string{"certificates"},
				Verbs:     []string{"get", "list", "watch"},
			},
		}
		switch c.Ingress.APISchema() {
		case api.APISchemaEngress:
			in.Rules = append(in.Rules, rbac.PolicyRule{
				APIGroups: []string{api.SchemeGroupVersion.Group},
				Resources: []string{"ingresses"},
				Verbs:     []string{"get", "list", "watch"},
			})
		case api.APISchemaIngress:
			in.Rules = append(in.Rules, rbac.PolicyRule{
				APIGroups: []string{extensions.GroupName},
				Resources: []string{"ingresses"},
				Verbs:     []string{"get", "list", "watch"},
			})
		}
		return in
	})
	return err
}

func (c *controller) ensureRoleBinding() error {
	meta := metav1.ObjectMeta{
		Namespace: c.Ingress.Namespace,
		Name:      c.Ingress.OffshootName(),
	}
	_, err := rbac_util.CreateOrPatchRoleBinding(c.KubeClient, meta, func(in *rbac.RoleBinding) *rbac.RoleBinding {
		in.ObjectMeta = c.ensureOwnerReference(in.ObjectMeta)

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
	})
	return err
}

func (c *controller) ensureRoleBindingDeleted() error {
	c.logger.Infof("Deleting RoleBinding %s/%s", c.Ingress.Namespace, c.Ingress.OffshootName())
	return c.KubeClient.RbacV1beta1().
		RoleBindings(c.Ingress.Namespace).
		Delete(c.Ingress.OffshootName(), &metav1.DeleteOptions{})
}

func (c *controller) ensureRolesDeleted() error {
	c.logger.Infof("Deleting Role %s/%s", c.Ingress.Namespace, c.Ingress.OffshootName())
	return c.KubeClient.RbacV1beta1().
		Roles(c.Ingress.Namespace).
		Delete(c.Ingress.OffshootName(), &metav1.DeleteOptions{})
}

func (c *controller) ensureServiceAccountDeleted() error {
	c.logger.Infof("Deleting ServiceAccount %s/%s", c.Ingress.Namespace, c.Ingress.OffshootName())
	return c.KubeClient.CoreV1().
		ServiceAccounts(c.Ingress.Namespace).
		Delete(c.Ingress.OffshootName(), &metav1.DeleteOptions{})
}
