package events

import (
	"reflect"
	"strings"
	"time"

	"github.com/appscode/log"
	aci "github.com/appscode/voyager/api"
	"k8s.io/apimachinery/pkg/runtime"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

type EventType string

const (
	Added   EventType = "ADDED"
	Deleted EventType = "DELETED"
	Updated EventType = "UPDATETD"
	None    EventType = "NONE"
)

func (e EventType) String() string {
	return string(e)
}

func (e EventType) IsAdded() bool {
	if e == Added {
		return true
	}
	return false
}

func (e EventType) IsDeleted() bool {
	if e == Deleted {
		return true
	}
	return false
}

func (e EventType) IsUpdated() bool {
	if e == Updated {
		return true
	}
	return false
}

func (e EventType) IsNone() bool {
	if e == None {
		return true
	}
	return false
}

func (e EventType) Is(event string) bool {
	return strings.EqualFold(e.String(), event)
}

type EventReason string

const (
	EventReasonAlertAcknowledgement EventReason = "AlertAcknowledgement"
)

func (r EventReason) String() string {
	return string(r)
}

type ObjectKind string

const (
	ObjectKindAlert ObjectKind = "Alert"
)

func (o ObjectKind) String() string {
	return string(o)
}

type ObjectType string

const (
	Certificate     ObjectType = "certificates"
	ConfigMap       ObjectType = "configmaps"
	DaemonSet       ObjectType = "daemonsets"
	Endpoint        ObjectType = "endpoints"
	ExtendedIngress ObjectType = "extendedingresses"
	Ingress         ObjectType = "ingresses"
	Namespace       ObjectType = "namespaces"
	Node            ObjectType = "nodes"
	StatefulSet     ObjectType = "statefulsets"
	Pod             ObjectType = "pods"
	RC              ObjectType = "replicationcontrollers"
	ReplicaSet      ObjectType = "replicasets"
	Deployments     ObjectType = "deployments"
	Service         ObjectType = "services"
	Unknown         ObjectType = "unknown"
)

func (o ObjectType) String() string {
	return string(o)
}

func (o ObjectType) IsUnknown() bool {
	if o == Unknown {
		return true
	}
	return false
}

func (o ObjectType) Is(r string) bool {
	return strings.EqualFold(o.String(), r)
}

type Event struct {
	id           string
	EventType    EventType
	ResourceType ObjectType
	Timestamp    time.Time

	// real objects that created te event
	RuntimeObj []interface{}

	// kubernetes object metadata
	MetaData apiv1.ObjectMeta
}

func New(Type EventType, obj ...interface{}) *Event {
	if len(obj) <= 0 {
		return &Event{
			EventType: None,
		}
	}
	objType := detectObjectType(obj[0])
	metadata := objectMetadata(obj[0], objType)
	log.Debugln(objType, Type, "with name", metadata.Name)

	id := composeKey(objType, string(metadata.UID))
	return &Event{
		id:           id,
		EventType:    Type,
		ResourceType: objType,
		MetaData:     metadata,
		RuntimeObj:   obj,
		Timestamp:    time.Now(),
	}
}

func composeKey(t ObjectType, uid string) string {
	return string(t) + "@" + uid
}

func detectObjectType(o interface{}) ObjectType {
	log.V(7).Infoln("got object type", reflect.TypeOf(o))
	switch o.(type) {
	case apiv1.Pod, *apiv1.Pod:
		return Pod
	case apiv1.Namespace, *apiv1.Namespace:
		return Namespace
	case apiv1.Service, *apiv1.Service:
		return Service
	case apiv1.ReplicationController, *apiv1.ReplicationController:
		return RC
	case apiv1.Node, *apiv1.Node:
		return Node
	case extensions.Ingress, *extensions.Ingress:
		return Ingress
	case apiv1.ConfigMap, *apiv1.ConfigMap:
		return ConfigMap
	case apiv1.Endpoints, *apiv1.Endpoints:
		return Endpoint
	case aci.Ingress, *aci.Ingress:
		return ExtendedIngress
	case aci.Certificate, *aci.Certificate:
		return Certificate
	case extensions.ReplicaSet, *extensions.ReplicaSet:
		return ReplicaSet
	case apps.StatefulSet, *apps.StatefulSet:
		return StatefulSet
	case extensions.DaemonSet, *extensions.DaemonSet:
		return DaemonSet
	case extensions.Deployment, *extensions.Deployment:
		return Deployments
	}
	return Unknown
}

func objectMetadata(o interface{}, t ObjectType) apiv1.ObjectMeta {
	switch t {
	case Pod:
		return o.(*apiv1.Pod).ObjectMeta
	case Namespace:
		return o.(*apiv1.Namespace).ObjectMeta
	case Service:
		return o.(*apiv1.Service).ObjectMeta
	case RC:
		return o.(*apiv1.ReplicationController).ObjectMeta
	case Node:
		return o.(*apiv1.Node).ObjectMeta
	case Ingress:
		return o.(*extensions.Ingress).ObjectMeta
	case ExtendedIngress:
		return o.(*aci.Ingress).ObjectMeta
	case Certificate:
		return o.(*aci.Certificate).ObjectMeta
	case Endpoint:
		return o.(*apiv1.Endpoints).ObjectMeta
	case ReplicaSet:
		return o.(*extensions.ReplicaSet).ObjectMeta
	case StatefulSet:
		return o.(*apps.StatefulSet).ObjectMeta
	case DaemonSet:
		return o.(*extensions.DaemonSet).ObjectMeta
	case Deployments:
		return o.(*extensions.Deployment).ObjectMeta
	case ConfigMap:
		return o.(*apiv1.ConfigMap).ObjectMeta
	}
	return apiv1.ObjectMeta{}
}

func (e *Event) Ignorable() bool {
	if e.EventType == None {
		return true
	}

	if e.EventType == Updated {
		// updated called but only old object is present.
		if len(e.RuntimeObj) <= 1 {
			return true
		}

		// updated but both are equal. no changes
		if reflect.DeepEqual(e.RuntimeObj[0], e.RuntimeObj[1]) {
			return true
		}
	}
	return false
}

func (e *Event) GetRuntimeObject() (runtime.Object, bool) {
	if len(e.RuntimeObj) > 0 {
		obj := e.RuntimeObj[0]
		if ro, ok := obj.(runtime.Object); ok {
			return ro, true
		}
	}
	return nil, false
}
