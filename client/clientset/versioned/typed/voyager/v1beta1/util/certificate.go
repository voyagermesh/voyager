package util

import (
	"fmt"

	"github.com/appscode/kutil"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	cs "github.com/appscode/voyager/client/clientset/versioned/typed/voyager/v1beta1"
	"github.com/evanphx/json-patch"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

func CreateOrPatchCertificate(c cs.VoyagerV1beta1Interface, meta metav1.ObjectMeta, transform func(alert *api.Certificate) *api.Certificate) (*api.Certificate, kutil.VerbType, error) {
	cur, err := c.Certificates(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating Certificate %s/%s.", meta.Namespace, meta.Name)
		out, err := c.Certificates(meta.Namespace).Create(transform(&api.Certificate{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Certificate",
				APIVersion: api.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
		return out, kutil.VerbCreated, err
	} else if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return PatchCertificate(c, cur, transform)
}

func PatchCertificate(c cs.VoyagerV1beta1Interface, cur *api.Certificate, transform func(*api.Certificate) *api.Certificate) (*api.Certificate, kutil.VerbType, error) {
	return PatchCertificateObject(c, cur, transform(cur.DeepCopy()))
}

func PatchCertificateObject(c cs.VoyagerV1beta1Interface, cur, mod *api.Certificate) (*api.Certificate, kutil.VerbType, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	modJson, err := json.Marshal(mod)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	patch, err := jsonpatch.CreateMergePatch(curJson, modJson)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, kutil.VerbUnchanged, nil
	}
	glog.V(3).Infof("Patching Certificate %s/%s with %s.", cur.Namespace, cur.Name, string(patch))
	out, err := c.Certificates(cur.Namespace).Patch(cur.Name, types.MergePatchType, patch)
	return out, kutil.VerbPatched, err
}

func TryUpdateCertificate(c cs.VoyagerV1beta1Interface, meta metav1.ObjectMeta, transform func(*api.Certificate) *api.Certificate) (result *api.Certificate, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.Certificates(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.Certificates(cur.Namespace).Update(transform(cur.DeepCopy()))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update Certificate %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = errors.Errorf("failed to update Certificate %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func UpdateCertificateStatus(
	c cs.VoyagerV1beta1Interface,
	in *api.Certificate,
	transform func(*api.CertificateStatus) *api.CertificateStatus,
	useSubresource ...bool,
) (result *api.Certificate, err error) {
	if len(useSubresource) > 1 {
		return nil, errors.Errorf("invalid value passed for useSubresource: %v", useSubresource)
	}

	apply := func(x *api.Certificate, copy bool) *api.Certificate {
		out := &api.Certificate{
			TypeMeta:   x.TypeMeta,
			ObjectMeta: x.ObjectMeta,
			Spec:       x.Spec,
		}
		if copy {
			out.Status = *transform(in.Status.DeepCopy())
		} else {
			out.Status = *transform(&in.Status)
		}
		return out
	}

	if len(useSubresource) == 1 && useSubresource[0] {
		attempt := 0
		cur := in.DeepCopy()
		err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
			attempt++
			var e2 error
			result, e2 = c.Certificates(in.Namespace).UpdateStatus(apply(cur, false))
			if kerr.IsConflict(e2) {
				latest, e3 := c.Certificates(in.Namespace).Get(in.Name, metav1.GetOptions{})
				switch {
				case e3 == nil:
					cur = latest
					return false, nil
				case kutil.IsRequestRetryable(e3):
					return false, nil
				default:
					return false, e3
				}
			} else if err != nil && !kutil.IsRequestRetryable(e2) {
				return false, e2
			}
			return e2 == nil, nil
		})

		if err != nil {
			err = fmt.Errorf("failed to update status of Certificate %s/%s after %d attempts due to %v", in.Namespace, in.Name, attempt, err)
		}
		return
	}

	result, _, err = PatchCertificateObject(c, in, apply(in, true))
	return
}
