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

type FakeCertificate struct {
	Fake *testing.Fake
	ns   string
}

var certResource = schema.GroupVersionResource{Group: api.GroupName, Version: "v1beta1", Resource: "certificates"}

var _ clientset.CertificateInterface = &FakeCertificate{}

// Get returns the Certificate by name.
func (mock *FakeCertificate) Get(name string) (*api.Certificate, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewGetAction(certResource, mock.ns, name), &api.Certificate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*api.Certificate), err
}

// List returns the a of Certificates.
func (mock *FakeCertificate) List(opts metav1.ListOptions) (*api.CertificateList, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewListAction(certResource, mock.ns, opts), &api.Certificate{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &api.CertificateList{}
	for _, item := range obj.(*api.CertificateList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Create creates a new Certificate.
func (mock *FakeCertificate) Create(svc *api.Certificate) (*api.Certificate, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewCreateAction(certResource, mock.ns, svc), &api.Certificate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*api.Certificate), err
}

// Update updates a Certificate.
func (mock *FakeCertificate) Update(svc *api.Certificate) (*api.Certificate, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateAction(certResource, mock.ns, svc), &api.Certificate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*api.Certificate), err
}

// Delete deletes a Certificate by name.
func (mock *FakeCertificate) Delete(name string) error {
	_, err := mock.Fake.
		Invokes(testing.NewDeleteAction(certResource, mock.ns, name), &api.Certificate{})

	return err
}

func (mock *FakeCertificate) UpdateStatus(srv *api.Certificate) (*api.Certificate, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateSubresourceAction(certResource, "status", mock.ns, srv), &api.Certificate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*api.Certificate), err
}

func (mock *FakeCertificate) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return mock.Fake.
		InvokesWatch(testing.NewWatchAction(certResource, mock.ns, opts))
}
