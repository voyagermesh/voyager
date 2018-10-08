package v1beta1

import (
	"context"
	"fmt"
	"strings"

	"github.com/appscode/kutil"
	"github.com/appscode/kutil/discovery"
	watchtools "github.com/appscode/kutil/tools/watch"
	"github.com/evanphx/json-patch"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"k8s.io/api/admissionregistration/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	apiregistration "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	apireg_cs "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

func init() {
	pflag.BoolVar(&bypassValidatingWebhookXray, "bypass-validating-webhook-xray", bypassValidatingWebhookXray, "if true, bypasses validating webhook xray checks")
}

var bypassValidatingWebhookXray = false

var ErrMissingKind = errors.New("test object missing kind")
var ErrMissingVersion = errors.New("test object missing version")
var ErrWebhookNotActivated = errors.New("Admission webhooks are not activated. Enable it by configuring --enable-admission-plugins flag of kube-apiserver. For details, visit: https://appsco.de/kube-apiserver-webhooks")

type ValidatingWebhookXray struct {
	config         *rest.Config
	apiserviceName string
	webhookName    string
	testObj        runtime.Object
	op             v1beta1.OperationType
	transform      func(_ runtime.Object)
}

func NewCreateValidatingWebhookXray(config *rest.Config, apiserviceName, webhookName string, testObj runtime.Object) *ValidatingWebhookXray {
	return &ValidatingWebhookXray{
		config:         config,
		apiserviceName: apiserviceName,
		webhookName:    webhookName,
		testObj:        testObj,
		op:             v1beta1.Create,
		transform:      nil,
	}
}

func NewUpdateValidatingWebhookXray(config *rest.Config, apiserviceName, webhookName string, testObj runtime.Object, transform func(_ runtime.Object)) *ValidatingWebhookXray {
	return &ValidatingWebhookXray{
		config:         config,
		apiserviceName: apiserviceName,
		webhookName:    webhookName,
		testObj:        testObj,
		op:             v1beta1.Update,
		transform:      transform,
	}
}

func NewDeleteValidatingWebhookXray(config *rest.Config, apiserviceName, webhookName string, testObj runtime.Object, transform func(_ runtime.Object)) *ValidatingWebhookXray {
	return &ValidatingWebhookXray{
		config:         config,
		apiserviceName: apiserviceName,
		webhookName:    webhookName,
		testObj:        testObj,
		op:             v1beta1.Delete,
		transform:      transform,
	}
}

func (d ValidatingWebhookXray) IsActive() error {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, kutil.GCTimeout)
	defer cancel()

	err := rest.LoadTLSFiles(d.config)
	if err != nil {
		return err
	}

	kc := apireg_cs.NewForConfigOrDie(d.config)
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = fields.OneTermEqualSelector(kutil.ObjectNameField, d.apiserviceName).String()
			return kc.ApiregistrationV1beta1().APIServices().List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.FieldSelector = fields.OneTermEqualSelector(kutil.ObjectNameField, d.apiserviceName).String()
			return kc.ApiregistrationV1beta1().APIServices().Watch(options)
		},
	}

	_, err = watchtools.UntilWithSync(
		ctx,
		lw,
		&apiregistration.APIService{},
		nil,
		func(event watch.Event) (bool, error) {
			switch event.Type {
			case watch.Deleted:
				return false, nil
			case watch.Error:
				return false, errors.Wrap(err, "error watching")
			case watch.Added, watch.Modified:
				cur := event.Object.(*apiregistration.APIService)
				for _, cond := range cur.Status.Conditions {
					if cond.Type == apiregistration.Available && cond.Status == apiregistration.ConditionTrue {
						return d.check()
					}
				}
				return false, nil
			default:
				return false, fmt.Errorf("unexpected event type: %v", event.Type)
			}
		})
	return err
}

func (d ValidatingWebhookXray) check() (bool, error) {
	if bypassValidatingWebhookXray {
		return true, nil
	}

	kc, err := kubernetes.NewForConfig(d.config)
	if err != nil {
		return false, err
	}

	dc, err := dynamic.NewForConfig(d.config)
	if err != nil {
		return false, err
	}

	gvk := d.testObj.GetObjectKind().GroupVersionKind()
	if gvk.Version == "" {
		return false, ErrMissingVersion
	}
	if gvk.Kind == "" {
		return false, ErrMissingKind
	}
	glog.Infof("testing ValidatingWebhook %s using an object with GVK = %s", d.webhookName, gvk.String())

	gvr, err := discovery.ResourceForGVK(kc.Discovery(), gvk)
	if err != nil {
		return false, err
	}
	glog.Infof("testing ValidatingWebhook %s using an object with GVR = %s", d.webhookName, gvr.String())

	accessor, err := meta.Accessor(d.testObj)
	if err != nil {
		return false, err
	}

	var ri dynamic.ResourceInterface
	if accessor.GetNamespace() != "" {
		ri = dc.Resource(gvr).Namespace(accessor.GetNamespace())
	} else {
		ri = dc.Resource(gvr)
	}

	objJson, err := json.Marshal(d.testObj)
	if err != nil {
		return false, err
	}

	u := unstructured.Unstructured{}
	_, _, err = unstructured.UnstructuredJSONScheme.Decode(objJson, nil, &u)
	if err != nil {
		return false, err
	}

	if d.op == v1beta1.Create {
		_, err := ri.Create(&u)
		if kerr.IsForbidden(err) &&
			strings.HasPrefix(err.Error(), fmt.Sprintf(`admission webhook "%s" denied the request:`, d.webhookName)) {
			glog.Infof("failed to create invalid test object as expected with error: %s", err)
			return true, nil
		} else if kutil.IsRequestRetryable(err) {
			return false, nil
		} else if err != nil {
			return false, err
		}

		err = ri.Delete(accessor.GetName(), &metav1.DeleteOptions{})
		if kutil.IsRequestRetryable(err) {
			return false, nil
		} else if err != nil {
			return false, err
		}
		return false, ErrWebhookNotActivated
	} else if d.op == v1beta1.Update {
		_, err := ri.Create(&u)
		if kutil.IsRequestRetryable(err) {
			return false, nil
		} else if err != nil {
			return false, err
		}

		mod := d.testObj.DeepCopyObject()
		d.transform(mod)
		modJson, err := json.Marshal(mod)
		if err != nil {
			return false, err
		}

		patch, err := jsonpatch.CreateMergePatch(objJson, modJson)
		if err != nil {
			return false, err
		}

		_, err = ri.Patch(accessor.GetName(), types.MergePatchType, patch)
		defer ri.Delete(accessor.GetName(), &metav1.DeleteOptions{})

		if kerr.IsForbidden(err) &&
			strings.HasPrefix(err.Error(), fmt.Sprintf(`admission webhook "%s" denied the request:`, d.webhookName)) {
			glog.Infof("failed to update test object as expected with error: %s", err)
			return true, nil
		} else if kutil.IsRequestRetryable(err) {
			return false, nil
		} else if err != nil {
			return false, err
		}

		return false, ErrWebhookNotActivated
	} else if d.op == v1beta1.Delete {
		_, err := ri.Create(&u)
		if kutil.IsRequestRetryable(err) {
			return false, nil
		} else if err != nil {
			return false, err
		}

		err = ri.Delete(accessor.GetName(), &metav1.DeleteOptions{})
		if kerr.IsForbidden(err) &&
			strings.HasPrefix(err.Error(), fmt.Sprintf(`admission webhook "%s" denied the request:`, d.webhookName)) {
			defer func() {
				// update to make it valid
				mod := d.testObj.DeepCopyObject()
				d.transform(mod)
				modJson, err := json.Marshal(mod)
				if err != nil {
					return
				}

				patch, err := jsonpatch.CreateMergePatch(objJson, modJson)
				if err != nil {
					return
				}

				ri.Patch(accessor.GetName(), types.MergePatchType, patch)

				// delete
				ri.Delete(accessor.GetName(), &metav1.DeleteOptions{})
			}()

			glog.Infof("failed to delete test object as expected with error: %s", err)
			return true, nil
		} else if kutil.IsRequestRetryable(err) {
			return false, nil
		} else if err != nil {
			return false, err
		}
		return false, ErrWebhookNotActivated
	}

	return false, nil
}
