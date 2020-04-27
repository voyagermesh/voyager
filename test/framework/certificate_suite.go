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
	api_v1beta1 "voyagermesh.dev/voyager/apis/voyager/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (ci *certificateInvocation) Create(cert *api_v1beta1.Certificate) error {
	_, err := ci.VoyagerClient.VoyagerV1beta1().Certificates(ci.Namespace()).Create(cert)
	if err != nil {
		return err
	}
	return nil
}

func (ci *certificateInvocation) Get(cert *api_v1beta1.Certificate) (*api_v1beta1.Certificate, error) {
	return ci.VoyagerClient.VoyagerV1beta1().Certificates(ci.Namespace()).Get(cert.Name, metav1.GetOptions{})
}

func (ci *certificateInvocation) Update(cert *api_v1beta1.Certificate) error {
	_, err := ci.VoyagerClient.VoyagerV1beta1().Certificates(ci.Namespace()).Update(cert)
	return err
}

func (ci *certificateInvocation) Delete(cert *api_v1beta1.Certificate) error {
	return ci.VoyagerClient.VoyagerV1beta1().Certificates(ci.Namespace()).Delete(cert.Name, &metav1.DeleteOptions{})
}

func (ci *certificateInvocation) GetSkeleton() *api_v1beta1.Certificate {
	cert := &api_v1beta1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ci.UniqueName(),
			Namespace: ci.Namespace(),
		},
		Spec: api_v1beta1.CertificateSpec{},
	}
	return cert
}
