package fake

import (
	"github.com/appscode/voyager/api"
	"github.com/appscode/voyager/client/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/testing"
)

type FakeIngress struct {
	Fake *testing.Fake
	ns   string
}

var ingressResource = schema.GroupVersionResource{Group: api.GroupName, Version: "v1beta1", Resource: "ingresses"}

var _ clientset.IngressInterface = &FakeIngress{}

// Get returns the Ingress by name.
func (mock *FakeIngress) Get(name string) (*api.Ingress, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewGetAction(ingressResource, mock.ns, name), &api.Ingress{})

	if obj == nil {
		return nil, err
	}
	return obj.(*api.Ingress), err
}

// List returns the a of Ingresss.
func (mock *FakeIngress) List(opts metav1.ListOptions) (*api.IngressList, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewListAction(ingressResource, mock.ns, opts), &api.Ingress{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &api.IngressList{}
	for _, item := range obj.(*api.IngressList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Create creates a new Ingress.
func (mock *FakeIngress) Create(svc *api.Ingress) (*api.Ingress, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewCreateAction(ingressResource, mock.ns, svc), &api.Ingress{})

	if obj == nil {
		return nil, err
	}
	return obj.(*api.Ingress), err
}

// Update updates a Ingress.
func (mock *FakeIngress) Update(svc *api.Ingress) (*api.Ingress, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateAction(ingressResource, mock.ns, svc), &api.Ingress{})

	if obj == nil {
		return nil, err
	}
	return obj.(*api.Ingress), err
}

// Delete deletes a Ingress by name.
func (mock *FakeIngress) Delete(name string) error {
	_, err := mock.Fake.
		Invokes(testing.NewDeleteAction(ingressResource, mock.ns, name), &api.Ingress{})

	return err
}

func (mock *FakeIngress) UpdateStatus(srv *api.Ingress) (*api.Ingress, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateSubresourceAction(ingressResource, "status", mock.ns, srv), &api.Ingress{})

	if obj == nil {
		return nil, err
	}
	return obj.(*api.Ingress), err
}

func (mock *FakeIngress) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return mock.Fake.
		InvokesWatch(testing.NewWatchAction(ingressResource, mock.ns, opts))
}
