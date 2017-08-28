package v1beta1

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/appscode/jsonpatch"
	"github.com/appscode/kutil"
	"github.com/golang/glog"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	rbac "k8s.io/client-go/pkg/apis/rbac/v1beta1"
)

func EnsureRole(c clientset.Interface, meta metav1.ObjectMeta, transform func(*rbac.Role) *rbac.Role) (*rbac.Role, error) {
	return CreateOrPatchRole(c, meta, transform)
}

func CreateOrPatchRole(c clientset.Interface, meta metav1.ObjectMeta, transform func(*rbac.Role) *rbac.Role) (*rbac.Role, error) {
	cur, err := c.RbacV1beta1().Roles(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return c.RbacV1beta1().Roles(meta.Namespace).Create(transform(&rbac.Role{ObjectMeta: meta}))
	} else if err != nil {
		return nil, err
	}
	return PatchRole(c, cur, transform)
}

func PatchRole(c clientset.Interface, cur *rbac.Role, transform func(*rbac.Role) *rbac.Role) (*rbac.Role, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := json.Marshal(transform(cur))
	if err != nil {
		return nil, err
	}

	patch, err := jsonpatch.CreatePatch(curJson, modJson)
	if err != nil {
		return nil, err
	}
	if len(patch) == 0 {
		return cur, nil
	}
	pb, err := json.MarshalIndent(patch, "", "  ")
	if err != nil {
		return nil, err
	}
	glog.V(5).Infof("Patching Role %s@%s with %s.", cur.Name, cur.Namespace, string(pb))
	return c.RbacV1beta1().Roles(cur.Namespace).Patch(cur.Name, types.JSONPatchType, pb)
}

func TryPatchRole(c clientset.Interface, meta metav1.ObjectMeta, transform func(*rbac.Role) *rbac.Role) (*rbac.Role, error) {
	attempt := 0
	for ; attempt < kutil.MaxAttempts; attempt = attempt + 1 {
		cur, err := c.RbacV1beta1().Roles(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return cur, err
		} else if err == nil {
			return PatchRole(c, cur, transform)
		}
		glog.Errorf("Attempt %d failed to patch Role %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(kutil.RetryInterval)
	}
	return nil, fmt.Errorf("Failed to patch Role %s@%s after %d attempts.", meta.Name, meta.Namespace, attempt)
}

func TryUpdateRole(c clientset.Interface, meta metav1.ObjectMeta, transform func(*rbac.Role) *rbac.Role) (*rbac.Role, error) {
	attempt := 0
	for ; attempt < kutil.MaxAttempts; attempt = attempt + 1 {
		cur, err := c.RbacV1beta1().Roles(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return cur, err
		} else if err == nil {
			return c.RbacV1beta1().Roles(cur.Namespace).Update(transform(cur))
		}
		glog.Errorf("Attempt %d failed to update Role %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(kutil.RetryInterval)
	}
	return nil, fmt.Errorf("Failed to update Role %s@%s after %d attempts.", meta.Name, meta.Namespace, attempt)
}
