package v1beta1

import (
	"reflect"

	"github.com/appscode/go/log"
	meta_util "github.com/appscode/kutil/meta"
	"github.com/golang/glog"
)

func (e *Ingress) AlreadyObserved(other *Ingress) bool {
	if e == nil {
		return other == nil
	}
	if other == nil { // && d != nil
		return false
	}
	if e == other {
		return true
	}

	var match bool

	if EnableStatusSubresource {
		match = e.Status.ObservedGeneration >= e.Generation
	} else {
		match = meta_util.Equal(e.Spec, other.Spec)
	}
	if match {
		match = reflect.DeepEqual(e.Labels, other.Labels)
	}
	if match {
		match = meta_util.EqualAnnotation(e.Annotations, other.Annotations)
	}

	if !match && bool(glog.V(log.LevelDebug)) {
		diff := meta_util.Diff(other, e)
		glog.V(log.LevelDebug).Infof("%s %s/%s has changed. Diff: %s", meta_util.GetKind(e), e.Namespace, e.Name, diff)
	}
	return match
}

func (e *Certificate) AlreadyObserved(other *Certificate) bool {
	if e == nil {
		return other == nil
	}
	if other == nil { // && d != nil
		return false
	}
	if e == other {
		return true
	}

	var match bool

	if EnableStatusSubresource {
		match = e.Status.ObservedGeneration >= e.Generation
	} else {
		match = meta_util.Equal(e.Spec, other.Spec)
	}
	if match {
		match = reflect.DeepEqual(e.Labels, other.Labels)
	}
	if match {
		match = meta_util.EqualAnnotation(e.Annotations, other.Annotations)
	}

	if !match && bool(glog.V(log.LevelDebug)) {
		diff := meta_util.Diff(other, e)
		glog.V(log.LevelDebug).Infof("%s %s/%s has changed. Diff: %s", meta_util.GetKind(e), e.Namespace, e.Name, diff)
	}
	return match
}
