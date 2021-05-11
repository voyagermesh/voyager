/*
Copyright AppsCode Inc. and Contributors

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

package server

import (
	"io"
	"net"

	"voyagermesh.dev/voyager/pkg/operator"
	"voyagermesh.dev/voyager/pkg/server"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	license "go.bytebuilders.dev/license-verifier/kubernetes"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"kmodules.xyz/client-go/tools/clientcmd"
)

const defaultEtcdPathPrefix = "/registry/voyager.appscode.com"

type VoyagerOptions struct {
	RecommendedOptions *genericoptions.RecommendedOptions
	OperatorOptions    *OperatorOptions

	StdOut io.Writer
	StdErr io.Writer
}

func NewVoyagerOptions(out, errOut io.Writer) *VoyagerOptions {
	o := &VoyagerOptions{
		// TODO we will nil out the etcd storage options.  This requires a later level of k8s.io/apiserver
		RecommendedOptions: genericoptions.NewRecommendedOptions(
			defaultEtcdPathPrefix,
			server.Codecs.LegacyCodec(admissionv1beta1.SchemeGroupVersion),
		),
		OperatorOptions: NewOperatorOptions(),
		StdOut:          out,
		StdErr:          errOut,
	}
	o.RecommendedOptions.Etcd = nil
	o.RecommendedOptions.Admission = nil

	return o
}

func (o *VoyagerOptions) AddFlags(fs *pflag.FlagSet) {
	o.RecommendedOptions.AddFlags(fs)
	o.OperatorOptions.AddFlags(fs)
}

func (o VoyagerOptions) Validate(args []string) error {
	return nil
}

func (o *VoyagerOptions) Complete() error {
	return nil
}

func (o VoyagerOptions) Config() (*server.VoyagerConfig, error) {
	// TODO have a "real" external address
	if err := o.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		return nil, errors.Errorf("error creating self-signed certificates: %v", err)
	}

	serverConfig := genericapiserver.NewRecommendedConfig(server.Codecs)
	if err := o.RecommendedOptions.ApplyTo(serverConfig); err != nil {
		return nil, err
	}
	// Fixes https://github.com/Azure/AKS/issues/522
	clientcmd.Fix(serverConfig.ClientConfig)

	operatorConfig := operator.NewOperatorConfig(serverConfig.ClientConfig)
	if err := o.OperatorOptions.ApplyTo(operatorConfig); err != nil {
		return nil, err
	}

	config := &server.VoyagerConfig{
		GenericConfig:  serverConfig,
		OperatorConfig: operatorConfig,
	}
	return config, nil
}

func (o VoyagerOptions) Run(stopCh <-chan struct{}) error {
	config, err := o.Config()
	if err != nil {
		return err
	}

	s, err := config.Complete().New()
	if err != nil {
		return err
	}

	// Start periodic license verification
	//nolint:errcheck
	go license.VerifyLicensePeriodically(config.OperatorConfig.ClientConfig, o.OperatorOptions.LicenseFile, stopCh)

	return s.Run(stopCh)
}
