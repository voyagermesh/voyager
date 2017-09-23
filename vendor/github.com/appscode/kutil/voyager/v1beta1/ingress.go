package v1beta1

import (
	"encoding/json"
	"fmt"

	"github.com/appscode/kutil"
	aci "github.com/appscode/voyager/apis/voyager/v1beta1"
	tcs "github.com/appscode/voyager/client/typed/voyager/v1beta1"
	"github.com/golang/glog"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/wait"
)

func EnsureIngress(c tcs.VoyagerV1beta1Interface, meta metav1.ObjectMeta, transform func(alert *aci.Ingress) *aci.Ingress) (*aci.Ingress, error) {
	return CreateOrPatchIngress(c, meta, transform)
}

func CreateOrPatchIngress(c tcs.VoyagerV1beta1Interface, meta metav1.ObjectMeta, transform func(alert *aci.Ingress) *aci.Ingress) (*aci.Ingress, error) {
	cur, err := c.Ingresses(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return c.Ingresses(meta.Namespace).Create(transform(&aci.Ingress{ObjectMeta: meta}))
	} else if err != nil {
		return nil, err
	}
	return PatchIngress(c, cur, transform)
}

func PatchIngress(c tcs.VoyagerV1beta1Interface, cur *aci.Ingress, transform func(*aci.Ingress) *aci.Ingress) (*aci.Ingress, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := json.Marshal(transform(cur))
	if err != nil {
		return nil, err
	}

	patch, err := jsonmergepatch.CreateThreeWayJSONMergePatch(curJson, modJson, curJson)
	if err != nil {
		return nil, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, nil
	}
	glog.V(5).Infof("Patching Ingress %s@%s with %s.", cur.Name, cur.Namespace, string(patch))
	result, err := c.Ingresses(cur.Namespace).Patch(cur.Name, types.MergePatchType, patch)
	return result, err
}

func TryPatchIngress(c tcs.VoyagerV1beta1Interface, meta metav1.ObjectMeta, transform func(*aci.Ingress) *aci.Ingress) (result *aci.Ingress, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.Ingresses(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = PatchIngress(c, cur, transform)
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to patch Ingress %s@%s due to %v.", attempt, cur.Name, cur.Namespace, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to patch Ingress %s@%s after %d attempts due to %v", meta.Name, meta.Namespace, attempt, err)
	}
	return
}

func TryUpdateIngress(c tcs.VoyagerV1beta1Interface, meta metav1.ObjectMeta, transform func(*aci.Ingress) *aci.Ingress) (result *aci.Ingress, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.Ingresses(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.Ingresses(cur.Namespace).Update(transform(cur))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update Ingress %s@%s due to %v.", attempt, cur.Name, cur.Namespace, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update Ingress %s@%s after %d attempts due to %v", meta.Name, meta.Namespace, attempt, err)
	}
	return
}
