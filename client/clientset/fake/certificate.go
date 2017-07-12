package fake

import (
	tapi "github.com/appscode/voyager/api"
	"github.com/appscode/voyager/client/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/testing"
)

type FakeCertificate struct {
	Fake *testing.Fake
	ns   string
}

var certResource = tapi.V1beta1SchemeGroupVersion.WithResource(tapi.ResourceTypeCertificate)

var _ clientset.CertificateInterface = &FakeCertificate{}

// Get returns the Certificate by name.
func (mock *FakeCertificate) Get(name string) (*tapi.Certificate, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewGetAction(certResource, mock.ns, name), &tapi.Certificate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*tapi.Certificate), err
}

// List returns the a of Certificates.
func (mock *FakeCertificate) List(opts metav1.ListOptions) (*tapi.CertificateList, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewListAction(certResource, mock.ns, opts), &tapi.Certificate{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &tapi.CertificateList{}
	for _, item := range obj.(*tapi.CertificateList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Create creates a new Certificate.
func (mock *FakeCertificate) Create(svc *tapi.Certificate) (*tapi.Certificate, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewCreateAction(certResource, mock.ns, svc), &tapi.Certificate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*tapi.Certificate), err
}

// Update updates a Certificate.
func (mock *FakeCertificate) Update(svc *tapi.Certificate) (*tapi.Certificate, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateAction(certResource, mock.ns, svc), &tapi.Certificate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*tapi.Certificate), err
}

// Delete deletes a Certificate by name.
func (mock *FakeCertificate) Delete(name string) error {
	_, err := mock.Fake.
		Invokes(testing.NewDeleteAction(certResource, mock.ns, name), &tapi.Certificate{})

	return err
}

func (mock *FakeCertificate) UpdateStatus(srv *tapi.Certificate) (*tapi.Certificate, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateSubresourceAction(certResource, "status", mock.ns, srv), &tapi.Certificate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*tapi.Certificate), err
}

func (mock *FakeCertificate) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return mock.Fake.
		InvokesWatch(testing.NewWatchAction(certResource, mock.ns, opts))
}
