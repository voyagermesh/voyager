package ingress

import (
	"reflect"

	"github.com/appscode/go/errors"
	api "github.com/appscode/voyager/apis/voyager"
	api_v1beta1 "github.com/appscode/voyager/apis/voyager/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	rbac "k8s.io/client-go/pkg/apis/rbac/v1beta1"
)

func (c *controller) ensureServiceAccount() error {
	sa, err := c.KubeClient.CoreV1().ServiceAccounts(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		sa = &apiv1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      c.Ingress.OffshootName(),
				Namespace: c.Ingress.Namespace,
				Annotations: map[string]string{
					api_v1beta1.OriginAPISchema: c.Ingress.APISchema(),
					api_v1beta1.OriginName:      c.Ingress.GetName(),
				},
			},
		}
		_, err = c.KubeClient.CoreV1().ServiceAccounts(c.Ingress.Namespace).Create(sa)
		return err
	} else if err != nil {
		return errors.FromErr(err).Err()
	}

	needsUpdate := false
	if val, ok := c.ensureOriginAnnotations(sa.Annotations); ok {
		needsUpdate = true
		sa.Annotations = val
	}

	if needsUpdate {
		_, err = c.KubeClient.CoreV1().ServiceAccounts(c.Ingress.Namespace).Update(sa)
		return err
	}
	return nil
}

func (c *controller) ensureRoles() error {
	defaultRole := &rbac.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Ingress.OffshootName(),
			Namespace: c.Ingress.Namespace,
			Annotations: map[string]string{
				api_v1beta1.OriginAPISchema: c.Ingress.APISchema(),
				api_v1beta1.OriginName:      c.Ingress.GetName(),
			},
		},
		Rules: []rbac.PolicyRule{
			{
				APIGroups: []string{apiv1.GroupName},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}

	switch c.Ingress.APISchema() {
	case api_v1beta1.APISchemaEngress:
		defaultRole.Rules = append(defaultRole.Rules, rbac.PolicyRule{
			APIGroups:     []string{api.GroupName},
			Resources:     []string{"ingresses"},
			ResourceNames: []string{c.Ingress.Name},
			Verbs:         []string{"get"},
		})
	case api_v1beta1.APISchemaIngress:
		defaultRole.Rules = append(defaultRole.Rules, rbac.PolicyRule{
			APIGroups:     []string{extensions.GroupName},
			Resources:     []string{"ingresses"},
			ResourceNames: []string{c.Ingress.Name},
			Verbs:         []string{"get"},
		})
	}

	if c.Ingress.Stats() && len(c.Ingress.StatsSecretName()) > 0 {
		defaultRole.Rules = append(defaultRole.Rules, rbac.PolicyRule{
			APIGroups:     []string{apiv1.GroupName},
			Resources:     []string{"secret"},
			ResourceNames: []string{c.Ingress.StatsSecretName()},
			Verbs:         []string{"get"},
		})
	}

	role, err := c.KubeClient.RbacV1beta1().Roles(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		_, err = c.KubeClient.RbacV1beta1().Roles(c.Ingress.Namespace).Create(defaultRole)
		return err
	} else if err != nil {
		return errors.FromErr(err).Err()
	}

	needsUpdate := false
	if val, ok := c.ensureOriginAnnotations(role.Annotations); ok {
		needsUpdate = true
		role.Annotations = val
	}

	if !reflect.DeepEqual(role.Rules, defaultRole.Rules) {
		needsUpdate = true
		role.Rules = defaultRole.Rules
	}

	if needsUpdate {
		_, err = c.KubeClient.RbacV1beta1().Roles(c.Ingress.Namespace).Update(role)
		return err
	}
	return nil
}

func (c *controller) ensureRoleBinding() error {
	defaultRoleBinding := &rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Ingress.OffshootName(),
			Namespace: c.Ingress.Namespace,
			Annotations: map[string]string{
				api_v1beta1.OriginAPISchema: c.Ingress.APISchema(),
				api_v1beta1.OriginName:      c.Ingress.GetName(),
			},
		},
		RoleRef: rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "Role",
			Name:     c.Ingress.OffshootName(),
		},
		Subjects: []rbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      c.Ingress.OffshootName(),
				Namespace: c.Ingress.Namespace,
			},
		},
	}

	roleBinding, err := c.KubeClient.RbacV1beta1().RoleBindings(c.Ingress.Namespace).Get(c.Ingress.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		_, err = c.KubeClient.RbacV1beta1().RoleBindings(c.Ingress.Namespace).Create(defaultRoleBinding)
		return err
	} else if err != nil {
		return errors.FromErr(err).Err()
	}

	needsUpdate := false
	if val, ok := c.ensureOriginAnnotations(roleBinding.Annotations); ok {
		needsUpdate = true
		roleBinding.Annotations = val
	}

	if !reflect.DeepEqual(roleBinding.RoleRef, defaultRoleBinding.RoleRef) {
		needsUpdate = true
		roleBinding.RoleRef = defaultRoleBinding.RoleRef
	}

	if !reflect.DeepEqual(roleBinding.Subjects, defaultRoleBinding.Subjects) {
		needsUpdate = true
		roleBinding.Subjects = defaultRoleBinding.Subjects
	}

	if needsUpdate {
		_, err = c.KubeClient.RbacV1beta1().RoleBindings(c.Ingress.Namespace).Update(roleBinding)
		return err
	}
	return nil
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
