package clientset

import (
	tapi "github.com/appscode/voyager/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

type CertificateGetter interface {
	Certificates(namespace string) CertificateInterface
}

type CertificateInterface interface {
	List(opts metav1.ListOptions) (*tapi.CertificateList, error)
	Get(name string) (*tapi.Certificate, error)
	Create(certificate *tapi.Certificate) (*tapi.Certificate, error)
	Update(certificate *tapi.Certificate) (*tapi.Certificate, error)
	Delete(name string) error
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	UpdateStatus(certificate *tapi.Certificate) (*tapi.Certificate, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (*tapi.Certificate, error)
}

type CertificateImpl struct {
	r  rest.Interface
	ns string
}

var _ CertificateInterface = &CertificateImpl{}

func newCertificate(c *ExtensionClient, namespace string) *CertificateImpl {
	return &CertificateImpl{c.restClient, namespace}
}

func (c *CertificateImpl) List(opts metav1.ListOptions) (result *tapi.CertificateList, err error) {
	result = &tapi.CertificateList{}
	err = c.r.Get().
		Namespace(c.ns).
		Resource(tapi.ResourceTypeCertificate).
		VersionedParams(&opts, ExtendedCodec).
		Do().
		Into(result)
	return
}

func (c *CertificateImpl) Get(name string) (result *tapi.Certificate, err error) {
	result = &tapi.Certificate{}
	err = c.r.Get().
		Namespace(c.ns).
		Resource(tapi.ResourceTypeCertificate).
		Name(name).
		Do().
		Into(result)
	return
}

func (c *CertificateImpl) Create(certificate *tapi.Certificate) (result *tapi.Certificate, err error) {
	result = &tapi.Certificate{}
	err = c.r.Post().
		Namespace(c.ns).
		Resource(tapi.ResourceTypeCertificate).
		Body(certificate).
		Do().
		Into(result)
	return
}

func (c *CertificateImpl) Update(certificate *tapi.Certificate) (result *tapi.Certificate, err error) {
	result = &tapi.Certificate{}
	err = c.r.Put().
		Namespace(c.ns).
		Resource(tapi.ResourceTypeCertificate).
		Name(certificate.Name).
		Body(certificate).
		Do().
		Into(result)
	return
}

func (c *CertificateImpl) Delete(name string) (err error) {
	return c.r.Delete().
		Namespace(c.ns).
		Resource(tapi.ResourceTypeCertificate).
		Name(name).
		Do().
		Error()
}

func (c *CertificateImpl) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return c.r.Get().
		Prefix("watch").
		Namespace(c.ns).
		Resource(tapi.ResourceTypeCertificate).
		VersionedParams(&opts, ExtendedCodec).
		Watch()
}

func (c *CertificateImpl) UpdateStatus(certificate *tapi.Certificate) (result *tapi.Certificate, err error) {
	result = &tapi.Certificate{}
	err = c.r.Put().
		Namespace(c.ns).
		Resource(tapi.ResourceTypeCertificate).
		Name(certificate.Name).
		SubResource("status").
		Body(certificate).
		Do().
		Into(result)
	return
}

func (c *CertificateImpl) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *tapi.Certificate, err error) {
	result = &tapi.Certificate{}
	err = c.r.Patch(pt).
		Namespace(c.ns).
		Resource(tapi.ResourceTypeCertificate).
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
