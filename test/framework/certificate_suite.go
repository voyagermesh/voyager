package framework

import (
	api_v1beta1 "github.com/appscode/voyager/apis/voyager/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (i *certificateInvocation) Create(cert *api_v1beta1.Certificate) error {
	_, err := i.V1beta1Client.Certificates(i.Namespace()).Create(cert)
	if err != nil {
		return err
	}
	return nil
}

func (i *certificateInvocation) Get(cert *api_v1beta1.Certificate) (*api_v1beta1.Certificate, error) {
	return i.V1beta1Client.Certificates(i.Namespace()).Get(cert.Name, metav1.GetOptions{})
}

func (i *certificateInvocation) Update(cert *api_v1beta1.Certificate) error {
	_, err := i.V1beta1Client.Certificates(i.Namespace()).Update(cert)
	return err
}

func (i *certificateInvocation) Delete(cert *api_v1beta1.Certificate) error {
	return i.V1beta1Client.Certificates(i.Namespace()).Delete(cert.Name, &metav1.DeleteOptions{})
}

func (i *certificateInvocation) GetSkeleton() *api_v1beta1.Certificate {
	cert := &api_v1beta1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      i.UniqueName(),
			Namespace: i.Namespace(),
		},
		Spec: api_v1beta1.CertificateSpec{},
	}
	return cert
}
