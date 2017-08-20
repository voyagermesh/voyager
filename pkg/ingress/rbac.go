package ingress

import (
	core_util "github.com/appscode/kutil/core/v1"
	rbac_util "github.com/appscode/kutil/rbac/v1beta1"
	"github.com/appscode/voyager/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	rbac "k8s.io/client-go/pkg/apis/rbac/v1beta1"
)

func (c *controller) ensureServiceAccount() error {
	meta := metav1.ObjectMeta{
		Namespace: c.Ingress.Namespace,
		Name:      c.Ingress.OffshootName(),
	}
	_, err := core_util.CreateOrPatchServiceAccount(c.KubeClient, meta, func(obj *apiv1.ServiceAccount) *apiv1.ServiceAccount {
		if obj.Annotations == nil {
			obj.Annotations = map[string]string{}
		}
		obj.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		obj.Annotations[api.OriginName] = c.Ingress.GetName()
		return obj
	})
	return err
}

func (c *controller) ensureRoles() error {
	meta := metav1.ObjectMeta{
		Namespace: c.Ingress.Namespace,
		Name:      c.Ingress.OffshootName(),
	}
	_, err := rbac_util.CreateOrPatchRole(c.KubeClient, meta, func(obj *rbac.Role) *rbac.Role {
		if obj.Annotations == nil {
			obj.Annotations = map[string]string{}
		}
		obj.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		obj.Annotations[api.OriginName] = c.Ingress.GetName()

		obj.Rules = []rbac.PolicyRule{
			{
				APIGroups: []string{apiv1.GroupName},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list", "watch"},
			},
		}
		switch c.Ingress.APISchema() {
		case api.APISchemaEngress:
			obj.Rules = append(obj.Rules, rbac.PolicyRule{
				APIGroups:     []string{api.GroupName},
				Resources:     []string{"ingresses"},
				ResourceNames: []string{c.Ingress.Name},
				Verbs:         []string{"get"},
			})
		case api.APISchemaIngress:
			obj.Rules = append(obj.Rules, rbac.PolicyRule{
				APIGroups:     []string{extensions.GroupName},
				Resources:     []string{"ingresses"},
				ResourceNames: []string{c.Ingress.Name},
				Verbs:         []string{"get"},
			})
		}
		if c.Ingress.Stats() && len(c.Ingress.StatsSecretName()) > 0 {
			obj.Rules = append(obj.Rules, rbac.PolicyRule{
				APIGroups:     []string{apiv1.GroupName},
				Resources:     []string{"secret"},
				ResourceNames: []string{c.Ingress.StatsSecretName()},
				Verbs:         []string{"get"},
			})
		}
		return obj
	})
	return err
}

func (c *controller) ensureRoleBinding() error {
	meta := metav1.ObjectMeta{
		Namespace: c.Ingress.Namespace,
		Name:      c.Ingress.OffshootName(),
	}
	_, err := rbac_util.CreateOrPatchRoleBinding(c.KubeClient, meta, func(obj *rbac.RoleBinding) *rbac.RoleBinding {
		if obj.Annotations == nil {
			obj.Annotations = map[string]string{}
		}
		obj.Annotations[api.OriginAPISchema] = c.Ingress.APISchema()
		obj.Annotations[api.OriginName] = c.Ingress.GetName()

		obj.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "Role",
			Name:     c.Ingress.OffshootName(),
		}
		obj.Subjects = []rbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      c.Ingress.OffshootName(),
				Namespace: c.Ingress.Namespace,
			},
		}
		return obj
	})
	return err
}

func (c *controller) ensureRoleBindingDeleted() error {
	return c.KubeClient.RbacV1beta1().
		RoleBindings(c.Ingress.Namespace).
		Delete(c.Ingress.OffshootName(), &metav1.DeleteOptions{})
}

func (c *controller) ensureRolesDeleted() error {
	return c.KubeClient.RbacV1beta1().
		Roles(c.Ingress.Namespace).
		Delete(c.Ingress.OffshootName(), &metav1.DeleteOptions{})
}

func (c *controller) ensureServiceAccountDeleted() error {
	return c.KubeClient.CoreV1().
		ServiceAccounts(c.Ingress.Namespace).
		Delete(c.Ingress.OffshootName(), &metav1.DeleteOptions{})
}
