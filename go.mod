module voyagermesh.dev/voyager

go 1.12

require (
	cloud.google.com/go v0.54.0
	github.com/Azure/azure-sdk-for-go v32.5.0+incompatible
	github.com/Azure/go-autorest/autorest v0.9.2
	github.com/Azure/go-autorest/autorest/adal v0.7.0
	github.com/Azure/go-autorest/autorest/azure/auth v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.2.0
	github.com/JamesClonk/vultr v2.0.1+incompatible // indirect
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/akamai/AkamaiOPEN-edgegrid-golang v0.8.0 // indirect
	github.com/appscode/go v0.0.0-20200323182826-54e98e09185a
	github.com/appscode/hello-grpc v0.0.0-20190207041230-eea009cbf42e
	github.com/appscode/pat v0.0.0-20170521084856-48ff78925b79
	github.com/aws/aws-sdk-go v1.20.20
	github.com/benbjohnson/clock v1.0.0
	github.com/cloudflare/cloudflare-go v0.10.8 // indirect
	github.com/codeskyblue/go-sh v0.0.0-20190412065543-76bd3d59ff27
	github.com/coreos/prometheus-operator v0.34.0
	github.com/dnsimple/dnsimple-go v0.0.0-20180703121714-35bcc6b47c20 // indirect
	github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/go-openapi/spec v0.19.2
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/google/go-cmp v0.3.0
	github.com/google/gofuzz v1.0.0
	github.com/hashicorp/vault/api v1.0.4
	github.com/json-iterator/go v1.1.7
	github.com/mitchellh/go-ps v0.0.0-20190716172923-621e5597135b
	github.com/moul/http2curl v1.0.0
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/orcaman/concurrent-map v0.0.0-20190826125027-8c72a8bb44f6
	github.com/ovh/go-ovh v0.0.0-20181109152953-ba5adb4cf014 // indirect
	github.com/pires/go-proxyproto v0.0.0-20190615163442-2c19fd512994
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.1.0
	github.com/prometheus/common v0.3.0
	github.com/prometheus/haproxy_exporter v0.0.0-00010101000000-000000000000
	github.com/shirou/gopsutil v2.19.10+incompatible
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.3
	github.com/stretchr/testify v1.4.0
	github.com/timewasted/linode v0.0.0-20160829202747-37e84520dcf7 // indirect
	github.com/tredoe/osutil v0.0.0-20191018075336-e272fdda81c8
	github.com/xenolf/lego v0.0.0-00010101000000-000000000000
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sys v0.0.0-20190801041406-cbf593c0f2f3 // indirect
	gomodules.xyz/cert v1.0.2
	google.golang.org/api v0.6.1-0.20190607001116-5213b8090861
	google.golang.org/grpc v1.23.0
	gopkg.in/gcfg.v1 v1.2.0
	k8s.io/api v0.0.0-20191114100352-16d7abae0d2a
	k8s.io/apiextensions-apiserver v0.0.0-20191114105449-027877536833
	k8s.io/apimachinery v0.0.0-20191028221656-72ed19daf4bb
	k8s.io/apiserver v0.0.0-20191114103151-9ca1dc586682
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20190816220812-743ec37842bf
	k8s.io/utils v0.0.0-20190801114015-581e00157fb1
	kmodules.xyz/client-go v0.0.0-20200305091010-36ee6a593ed3
	kmodules.xyz/crd-schema-fuzz v0.0.0-20191129174258-81f984340891
	kmodules.xyz/monitoring-agent-api v0.0.0-20200125202117-d3b3e33ce41f
	kmodules.xyz/openshift v0.0.0-20191127145035-f6c48a90dbb7 // indirect
	kmodules.xyz/webhook-runtime v0.0.0-20191127075323-d4bfdee6974d
	sigs.k8s.io/yaml v1.1.0
)

replace (
	bitbucket.org/ww/goautoneg => gomodules.xyz/goautoneg v0.0.0-20120707110453-a547fc61f48d
	cloud.google.com/go => cloud.google.com/go v0.38.0
	git.apache.org/thrift.git => github.com/apache/thrift v0.12.0
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v32.5.0+incompatible
	github.com/Azure/go-ansiterm => github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.0.0+incompatible
	github.com/Azure/go-autorest/autorest => github.com/Azure/go-autorest/autorest v0.9.0
	github.com/Azure/go-autorest/autorest/adal => github.com/Azure/go-autorest/autorest/adal v0.5.0
	github.com/Azure/go-autorest/autorest/azure/auth => github.com/Azure/go-autorest/autorest/azure/auth v0.2.0
	github.com/Azure/go-autorest/autorest/date => github.com/Azure/go-autorest/autorest/date v0.1.0
	github.com/Azure/go-autorest/autorest/mocks => github.com/Azure/go-autorest/autorest/mocks v0.2.0
	github.com/Azure/go-autorest/autorest/to => github.com/Azure/go-autorest/autorest/to v0.2.0
	github.com/Azure/go-autorest/autorest/validation => github.com/Azure/go-autorest/autorest/validation v0.1.0
	github.com/Azure/go-autorest/logger => github.com/Azure/go-autorest/logger v0.1.0
	github.com/Azure/go-autorest/tracing => github.com/Azure/go-autorest/tracing v0.5.0
	github.com/grpc-ecosystem/grpc-gateway => github.com/gomodules/grpc-gateway v1.3.1-ac
	github.com/miekg/dns => github.com/miekg/dns v1.0.7
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.2
	github.com/prometheus/haproxy_exporter => github.com/appscode/haproxy_exporter v0.7.2-0.20190508003714-b4abf52090e2
	github.com/xenolf/lego => github.com/appscode/lego v1.2.2-0.20181215093553-e57a0a1b7259
	google.golang.org/api => google.golang.org/api v0.6.1-0.20190607001116-5213b8090861
	k8s.io/api => k8s.io/api v0.0.0-20191114100352-16d7abae0d2a
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191114105449-027877536833
	k8s.io/apimachinery => github.com/kmodules/apimachinery v0.0.0-20191119091232-0553326db082
	k8s.io/apiserver => github.com/kmodules/apiserver v0.0.0-20191119111000-36ac3646ae82
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191114110141-0a35778df828
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191114101535-6c5935290e33
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20191114112024-4bbba8331835
	k8s.io/component-base => k8s.io/component-base v0.0.0-20191114102325-35a9586014f7
	k8s.io/klog => k8s.io/klog v0.4.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20191114103820-f023614fb9ea
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20190816220812-743ec37842bf
	k8s.io/kubectl => k8s.io/kubectl v0.0.0-20191114113550-6123e1c827f7
	k8s.io/kubernetes => github.com/kmodules/kubernetes v1.17.0-alpha.0.0.20191127022853-9d027e3886fd
	k8s.io/metrics => k8s.io/metrics v0.0.0-20191114105837-a4a2842dc51b
	k8s.io/repo-infra => k8s.io/repo-infra v0.0.0-20181204233714-00fe14e3d1a3
	k8s.io/utils => k8s.io/utils v0.0.0-20190801114015-581e00157fb1
	sigs.k8s.io/kustomize => sigs.k8s.io/kustomize v2.0.3+incompatible
	sigs.k8s.io/structured-merge-diff => sigs.k8s.io/structured-merge-diff v0.0.0-20190817042607-6149e4549fca
	sigs.k8s.io/yaml => sigs.k8s.io/yaml v1.1.0
)
