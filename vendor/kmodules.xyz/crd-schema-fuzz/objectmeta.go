package crdfuzz

import (
	"reflect"

	fuzz "github.com/google/gofuzz"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

// SafeFuzzerFuncs will merge the given funcLists, replacing any fuzzer for metav1.ObjectMeta with
// a fuzzer safe to use as a CRD embedded ObjectMeta. An ObjectMeta embedded inside a CRD only supports the
// following 5 fields. So, all other fields are set to their zero value.
//   - Name
//   - Namespace
//   - Labels
//   - Annotations
//   - Finalizers
// ref: https://github.com/kubernetes-sigs/controller-tools/commit/adfbf775195bf1c2366286684cc77a97b04a8cb9
func SafeFuzzerFuncs(funcs ...fuzzer.FuzzerFuncs) fuzzer.FuzzerFuncs {
	return func(codecs serializer.CodecFactory) []interface{} {
		result := []interface{}{}
		for _, fns := range funcs {
			if fns == nil {
				continue
			}
			for _, f := range fns(codecs) {
				if !fuzzerForObjectMeta(f) {
					result = append(result, f)
				}
			}
		}
		result = append(result, func(j *metav1.ObjectMeta, c fuzz.Continue) {
			c.FuzzNoCustom(j)

			if len(j.Labels) == 0 {
				j.Labels = nil
			} else {
				delete(j.Labels, "")
			}
			if len(j.Annotations) == 0 {
				j.Annotations = nil
			} else {
				delete(j.Annotations, "")
			}
			if len(j.Finalizers) == 0 {
				j.Finalizers = nil
			}

			j.GenerateName = ""
			j.SelfLink = ""
			j.UID = ""
			j.ResourceVersion = ""
			j.Generation = 0
			j.CreationTimestamp = metav1.Time{}
			j.DeletionTimestamp = nil
			j.DeletionGracePeriodSeconds = nil
			j.OwnerReferences = nil
			j.ClusterName = ""
			j.ManagedFields = nil
		})
		return result
	}
}

func fuzzerForObjectMeta(f interface{}) bool {
	x := reflect.TypeOf(f)
	if x.Kind() != reflect.Func {
		return false
	}
	if x.IsVariadic() {
		return false
	}

	/*
		*v1.ObjectMeta ObjectMeta v1.ObjectMeta k8s.io/apimachinery/pkg/apis/meta/v1
		Continue fuzz.Continue github.com/google/gofuzz
	*/

	numIn := x.NumIn()   // count number of parameters
	numOut := x.NumOut() // count number of return values

	if numIn != 2 {
		return false
	}
	if numOut != 0 {
		return false
	}
	{
		inV := x.In(0)
		if inV.Kind() != reflect.Ptr {
			return false
		}
		inU := inV.Elem()
		if inU.PkgPath() != "k8s.io/apimachinery/pkg/apis/meta/v1" || inU.Name() != "ObjectMeta" {
			return false
		}
	}
	{
		inV := x.In(1)
		if inV.Kind() != reflect.Struct {
			return false
		}
		if inV.PkgPath() != "github.com/google/gofuzz" || inV.Name() != "Continue" {
			return false
		}
	}
	return true
}
