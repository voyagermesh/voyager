package ingress

import (
	"reflect"

	"github.com/appscode/errors"
	"github.com/appscode/voyager/api"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	rbac "k8s.io/client-go/pkg/apis/rbac/v1beta1"
)

func (lbc *Controller) ensureServiceAccount() error {
	sa, err := lbc.KubeClient.CoreV1().ServiceAccounts(lbc.Ingress.Namespace).Get(lbc.Ingress.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		sa = &apiv1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      lbc.Ingress.OffshootName(),
				Namespace: lbc.Ingress.Namespace,
				Annotations: map[string]string{
					api.OriginAPISchema: lbc.Ingress.APISchema(),
					api.OriginName:      lbc.Ingress.GetName(),
				},
			},
		}
		_, err = lbc.KubeClient.CoreV1().ServiceAccounts(lbc.Ingress.Namespace).Create(sa)
		return err
	} else if err != nil {
		return errors.FromErr(err).Err()
	}

	needsUpdate := false
	if val, ok := lbc.ensureResourceAnnotations(sa.Annotations); ok {
		needsUpdate = true
		sa.Annotations = val
	}

	if needsUpdate {
		_, err = lbc.KubeClient.CoreV1().ServiceAccounts(lbc.Ingress.Namespace).Update(sa)
		return err
	}
	return nil
}

func (lbc *Controller) ensureRoles() error {
	defaultRole := &rbac.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lbc.Ingress.OffshootName(),
			Namespace: lbc.Ingress.Namespace,
			Annotations: map[string]string{
				api.OriginAPISchema: lbc.Ingress.APISchema(),
				api.OriginName:      lbc.Ingress.GetName(),
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

	switch lbc.Ingress.APISchema() {
	case api.APISchemaEngress:
		defaultRole.Rules = append(defaultRole.Rules, rbac.PolicyRule{
			APIGroups: []string{api.GroupName},
			Resources: []string{"ingresses"},
			Verbs:     []string{"get"},
		})
	case api.APISchemaIngress:
		defaultRole.Rules = append(defaultRole.Rules, rbac.PolicyRule{
			APIGroups: []string{extensions.GroupName},
			Resources: []string{"ingresses"},
			Verbs:     []string{"get"},
		})
	}

	if lbc.Ingress.Stats() && len(lbc.Ingress.StatsSecretName()) > 0 {
		defaultRole.Rules = append(defaultRole.Rules, rbac.PolicyRule{
			APIGroups:     []string{apiv1.GroupName},
			Resources:     []string{"secret"},
			ResourceNames: []string{lbc.Ingress.StatsSecretName()},
			Verbs:         []string{"get"},
		})
	}

	role, err := lbc.KubeClient.RbacV1beta1().Roles(lbc.Ingress.Namespace).Get(lbc.Ingress.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		_, err = lbc.KubeClient.RbacV1beta1().Roles(lbc.Ingress.Namespace).Create(defaultRole)
		return err
	} else if err != nil {
		return errors.FromErr(err).Err()
	}

	needsUpdate := false
	if val, ok := lbc.ensureResourceAnnotations(role.Annotations); ok {
		needsUpdate = true
		role.Annotations = val
	}

	if !reflect.DeepEqual(role.Rules, defaultRole.Rules) {
		needsUpdate = true
		role.Rules = defaultRole.Rules
	}

	if needsUpdate {
		_, err = lbc.KubeClient.RbacV1beta1().Roles(lbc.Ingress.Namespace).Update(role)
		return err
	}
	return nil
}

func (lbc *Controller) ensureRoleBinding() error {
	defaultRoleBinding := &rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lbc.Ingress.OffshootName(),
			Namespace: lbc.Ingress.Namespace,
			Annotations: map[string]string{
				api.OriginAPISchema: lbc.Ingress.APISchema(),
				api.OriginName:      lbc.Ingress.GetName(),
			},
		},
		RoleRef: rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "Role",
			Name:     lbc.Ingress.OffshootName(),
		},
		Subjects: []rbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      lbc.Ingress.OffshootName(),
				Namespace: lbc.Ingress.Namespace,
			},
		},
	}

	roleBinding, err := lbc.KubeClient.RbacV1beta1().RoleBindings(lbc.Ingress.Namespace).Get(lbc.Ingress.OffshootName(), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		_, err = lbc.KubeClient.RbacV1beta1().RoleBindings(lbc.Ingress.Namespace).Create(defaultRoleBinding)
		return err
	} else if err != nil {
		return errors.FromErr(err).Err()
	}

	needsUpdate := false
	if val, ok := lbc.ensureResourceAnnotations(roleBinding.Annotations); ok {
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
		_, err = lbc.KubeClient.RbacV1beta1().RoleBindings(lbc.Ingress.Namespace).Update(roleBinding)
		return err
	}
	return nil
}

func (lbc *Controller) ensureRoleBindingDeleted() error {
	return lbc.KubeClient.RbacV1beta1().
		RoleBindings(lbc.Ingress.Namespace).
		Delete(lbc.Ingress.OffshootName(), &metav1.DeleteOptions{})
}

func (lbc *Controller) ensureRolesDeleted() error {
	return lbc.KubeClient.RbacV1beta1().
		Roles(lbc.Ingress.Namespace).
		Delete(lbc.Ingress.OffshootName(), &metav1.DeleteOptions{})
}

func (lbc *Controller) ensureServiceAccountDeleted() error {
	return lbc.KubeClient.CoreV1().
		ServiceAccounts(lbc.Ingress.Namespace).
		Delete(lbc.Ingress.OffshootName(), &metav1.DeleteOptions{})
}
