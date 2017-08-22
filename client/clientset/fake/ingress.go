package fake

import (
	tapi "github.com/appscode/voyager/api"
	"github.com/appscode/voyager/client/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/testing"
)

type FakeIngress struct {
	Fake *testing.Fake
	ns   string
}

var ingressResource = tapi.V1beta1SchemeGroupVersion.WithResource(tapi.ResourceTypeIngress)

var _ clientset.IngressInterface = &FakeIngress{}

// Get returns the Ingress by name.
func (mock *FakeIngress) Get(name string) (*tapi.Ingress, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewGetAction(ingressResource, mock.ns, name), &tapi.Ingress{})

	if obj == nil {
		return nil, err
	}
	return obj.(*tapi.Ingress), err
}

// List returns the a of Ingresss.
func (mock *FakeIngress) List(opts metav1.ListOptions) (*tapi.IngressList, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewListAction(ingressResource, mock.ns, opts), &tapi.Ingress{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &tapi.IngressList{}
	for _, item := range obj.(*tapi.IngressList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Create creates a new Ingress.
func (mock *FakeIngress) Create(svc *tapi.Ingress) (*tapi.Ingress, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewCreateAction(ingressResource, mock.ns, svc), &tapi.Ingress{})

	if obj == nil {
		return nil, err
	}
	return obj.(*tapi.Ingress), err
}

// Update updates a Ingress.
func (mock *FakeIngress) Update(svc *tapi.Ingress) (*tapi.Ingress, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateAction(ingressResource, mock.ns, svc), &tapi.Ingress{})

	if obj == nil {
		return nil, err
	}
	return obj.(*tapi.Ingress), err
}

// Delete deletes a Ingress by name.
func (mock *FakeIngress) Delete(name string) error {
	_, err := mock.Fake.
		Invokes(testing.NewDeleteAction(ingressResource, mock.ns, name), &tapi.Ingress{})

	return err
}

func (mock *FakeIngress) UpdateStatus(srv *tapi.Ingress) (*tapi.Ingress, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateSubresourceAction(ingressResource, "status", mock.ns, srv), &tapi.Ingress{})

	if obj == nil {
		return nil, err
	}
	return obj.(*tapi.Ingress), err
}

func (mock *FakeIngress) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return mock.Fake.
		InvokesWatch(testing.NewWatchAction(ingressResource, mock.ns, opts))
}

func (mock *FakeIngress) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *tapi.Ingress, err error) {
	obj, err := mock.Fake.Invokes(testing.NewPatchSubresourceAction(ingressResource, mock.ns, name, data, subresources...), &tapi.Ingress{})
	if obj == nil {
		return nil, err
	}
	return obj.(*tapi.Ingress), err
}
