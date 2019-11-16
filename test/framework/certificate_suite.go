/*
Copyright The Voyager Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	api_v1beta1 "github.com/appscode/voyager/apis/voyager/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (i *certificateInvocation) Create(cert *api_v1beta1.Certificate) error {
	_, err := i.VoyagerClient.VoyagerV1beta1().Certificates(i.Namespace()).Create(cert)
	if err != nil {
		return err
	}
	return nil
}

func (i *certificateInvocation) Get(cert *api_v1beta1.Certificate) (*api_v1beta1.Certificate, error) {
	return i.VoyagerClient.VoyagerV1beta1().Certificates(i.Namespace()).Get(cert.Name, metav1.GetOptions{})
}

func (i *certificateInvocation) Update(cert *api_v1beta1.Certificate) error {
	_, err := i.VoyagerClient.VoyagerV1beta1().Certificates(i.Namespace()).Update(cert)
	return err
}

func (i *certificateInvocation) Delete(cert *api_v1beta1.Certificate) error {
	return i.VoyagerClient.VoyagerV1beta1().Certificates(i.Namespace()).Delete(cert.Name, &metav1.DeleteOptions{})
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
