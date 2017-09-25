package v1beta1

import (
	"encoding/json"
	"fmt"

	"github.com/appscode/kutil"
	"github.com/golang/glog"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	rbac "k8s.io/client-go/pkg/apis/rbac/v1beta1"
)

func EnsureRoleBinding(c clientset.Interface, meta metav1.ObjectMeta, transform func(*rbac.RoleBinding) *rbac.RoleBinding) (*rbac.RoleBinding, error) {
	return CreateOrPatchRoleBinding(c, meta, transform)
}

func CreateOrPatchRoleBinding(c clientset.Interface, meta metav1.ObjectMeta, transform func(*rbac.RoleBinding) *rbac.RoleBinding) (*rbac.RoleBinding, error) {
	cur, err := c.RbacV1beta1().RoleBindings(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return c.RbacV1beta1().RoleBindings(meta.Namespace).Create(transform(&rbac.RoleBinding{ObjectMeta: meta}))
	} else if err != nil {
		return nil, err
	}
	return PatchRoleBinding(c, cur, transform)
}

func PatchRoleBinding(c clientset.Interface, cur *rbac.RoleBinding, transform func(*rbac.RoleBinding) *rbac.RoleBinding) (*rbac.RoleBinding, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := json.Marshal(transform(cur))
	if err != nil {
		return nil, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, rbac.RoleBinding{})
	if err != nil {
		return nil, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, nil
	}
	glog.V(5).Infof("Patching RoleBinding %s@%s with %s.", cur.Name, cur.Namespace, string(patch))
	return c.RbacV1beta1().RoleBindings(cur.Namespace).Patch(cur.Name, types.StrategicMergePatchType, patch)
}

func TryPatchRoleBinding(c clientset.Interface, meta metav1.ObjectMeta, transform func(*rbac.RoleBinding) *rbac.RoleBinding) (result *rbac.RoleBinding, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.RbacV1beta1().RoleBindings(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = PatchRoleBinding(c, cur, transform)
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to patch RoleBinding %s@%s due to %v.", attempt, cur.Name, cur.Namespace, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to patch RoleBinding %s@%s after %d attempts due to %v", meta.Name, meta.Namespace, attempt, err)
	}
	return
}

func TryUpdateRoleBinding(c clientset.Interface, meta metav1.ObjectMeta, transform func(*rbac.RoleBinding) *rbac.RoleBinding) (result *rbac.RoleBinding, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.RbacV1beta1().RoleBindings(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.RbacV1beta1().RoleBindings(cur.Namespace).Update(transform(cur))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update RoleBinding %s@%s due to %v.", attempt, cur.Name, cur.Namespace, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update Role %s@%s after %d attempts due to %v", meta.Name, meta.Namespace, attempt, err)
	}
	return
}
