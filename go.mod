module voyagermesh.dev/voyager

go 1.12

require (
	cloud.google.com/go v0.58.0
	github.com/Azure/azure-sdk-for-go v43.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.10.2
	github.com/Azure/go-autorest/autorest/adal v0.8.3
	github.com/Azure/go-autorest/autorest/to v0.3.1-0.20191028180845-3492b2aff503
	github.com/JamesClonk/vultr v2.0.2+incompatible // indirect
	github.com/akamai/AkamaiOPEN-edgegrid-golang v1.0.0 // indirect
	github.com/appscode/hello-grpc v0.0.0-20190207041230-eea009cbf42e
	github.com/appscode/pat v0.0.0-20170521084856-48ff78925b79
	github.com/aws/aws-sdk-go v1.31.13
	github.com/benbjohnson/clock v1.0.2
	github.com/cloudflare/cloudflare-go v0.13.6 // indirect
	github.com/codeskyblue/go-sh v0.0.0-20200712050446-30169cf553fe
	github.com/dnsimple/dnsimple-go v0.0.0-00010101000000-000000000000 // indirect
	github.com/evanphx/json-patch v4.9.0+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/go-openapi/spec v0.19.8
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/google/go-cmp v0.4.1
	github.com/google/gofuzz v1.1.0
	github.com/hashicorp/vault/api v1.0.4
	github.com/imdario/mergo v0.3.6 // indirect
	github.com/json-iterator/go v1.1.10
	github.com/mitchellh/go-ps v0.0.0-20170309133038-4fdf99ab2936
	github.com/moul/http2curl v1.0.0
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/orcaman/concurrent-map v0.0.0-20190826125027-8c72a8bb44f6
	github.com/ovh/go-ovh v1.1.0 // indirect
	github.com/pires/go-proxyproto v0.1.3
	github.com/pkg/errors v0.9.1
	github.com/prometheus-operator/prometheus-operator v0.42.0
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.42.0
	github.com/prometheus/client_golang v1.6.0
	github.com/prometheus/common v0.10.0
	github.com/prometheus/haproxy_exporter v0.0.0-00010101000000-000000000000
	github.com/shirou/gopsutil v0.0.0-20180427012116-c95755e4bcd7
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	github.com/timewasted/linode v0.0.0-20160829202747-37e84520dcf7 // indirect
	github.com/tredoe/osutil v1.0.4
	github.com/xenolf/lego v0.0.0-00010101000000-000000000000
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	gomodules.xyz/blobfs v0.1.5
	gomodules.xyz/cert v1.2.0
	gomodules.xyz/pointer v0.0.0-20201105071923-daf60fa55209
	gomodules.xyz/runtime v0.0.0-20201104200926-d838b09dda8b
	gomodules.xyz/x v0.0.0-20201105065653-91c568df6331
	google.golang.org/api v0.26.0
	google.golang.org/grpc v1.29.1
	gopkg.in/gcfg.v1 v1.2.3
	k8s.io/api v0.18.9
	k8s.io/apiextensions-apiserver v0.18.9
	k8s.io/apimachinery v0.18.9
	k8s.io/apiserver v0.18.9
	k8s.io/client-go v0.18.9
	k8s.io/kube-openapi v0.0.0-20200410145947-61e04a5be9a6
	k8s.io/utils v0.0.0-20200414100711-2df71ebbae66
	kmodules.xyz/client-go v0.0.0-20201208053851-a1d7be95e006
	kmodules.xyz/crd-schema-fuzz v0.0.0-20200922204806-c1426cd7fcf4
	kmodules.xyz/monitoring-agent-api v0.0.0-20201105074044-be7a1044891a
	kmodules.xyz/webhook-runtime v0.0.0-20201105073856-2dc7382b88c6
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/dnsimple/dnsimple-go => github.com/dnsimple/dnsimple-go v0.0.0-20180703121714-35bcc6b47c20 // indirect
	github.com/grpc-ecosystem/grpc-gateway => github.com/gomodules/grpc-gateway v1.3.1-ac
	github.com/miekg/dns => github.com/miekg/dns v1.0.7
	github.com/prometheus/haproxy_exporter => github.com/appscode/haproxy_exporter v0.7.2-0.20190508003714-b4abf52090e2
	github.com/xenolf/lego => github.com/appscode/lego v1.2.2-0.20181215093553-e57a0a1b7259

)

replace bitbucket.org/ww/goautoneg => gomodules.xyz/goautoneg v0.0.0-20120707110453-a547fc61f48d

replace cloud.google.com/go => cloud.google.com/go v0.49.0

replace git.apache.org/thrift.git => github.com/apache/thrift v0.13.0

replace github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v35.0.0+incompatible

replace github.com/Azure/go-ansiterm => github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.0.0+incompatible

replace github.com/Azure/go-autorest/autorest => github.com/Azure/go-autorest/autorest v0.9.0

replace github.com/Azure/go-autorest/autorest/adal => github.com/Azure/go-autorest/autorest/adal v0.5.0

replace github.com/Azure/go-autorest/autorest/azure/auth => github.com/Azure/go-autorest/autorest/azure/auth v0.2.0

replace github.com/Azure/go-autorest/autorest/date => github.com/Azure/go-autorest/autorest/date v0.1.0

replace github.com/Azure/go-autorest/autorest/mocks => github.com/Azure/go-autorest/autorest/mocks v0.2.0

replace github.com/Azure/go-autorest/autorest/to => github.com/Azure/go-autorest/autorest/to v0.2.0

replace github.com/Azure/go-autorest/autorest/validation => github.com/Azure/go-autorest/autorest/validation v0.1.0

replace github.com/Azure/go-autorest/logger => github.com/Azure/go-autorest/logger v0.1.0

replace github.com/Azure/go-autorest/tracing => github.com/Azure/go-autorest/tracing v0.5.0

replace github.com/go-openapi/analysis => github.com/go-openapi/analysis v0.19.5

replace github.com/go-openapi/errors => github.com/go-openapi/errors v0.19.2

replace github.com/go-openapi/jsonpointer => github.com/go-openapi/jsonpointer v0.19.3

replace github.com/go-openapi/jsonreference => github.com/go-openapi/jsonreference v0.19.3

replace github.com/go-openapi/loads => github.com/go-openapi/loads v0.19.4

replace github.com/go-openapi/runtime => github.com/go-openapi/runtime v0.19.4

replace github.com/go-openapi/spec => github.com/go-openapi/spec v0.19.3

replace github.com/go-openapi/strfmt => github.com/go-openapi/strfmt v0.19.3

replace github.com/go-openapi/swag => github.com/go-openapi/swag v0.19.5

replace github.com/go-openapi/validate => github.com/go-openapi/validate v0.19.5

replace github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.1

replace github.com/golang/protobuf => github.com/golang/protobuf v1.3.2

replace github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.3.1

replace github.com/imdario/mergo => github.com/imdario/mergo v0.3.5

replace github.com/prometheus-operator/prometheus-operator => github.com/prometheus-operator/prometheus-operator v0.42.0

replace github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring => github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.42.0

replace github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.7.1

replace go.etcd.io/etcd => go.etcd.io/etcd v0.0.0-20191023171146-3cf2f69b5738

replace google.golang.org/api => google.golang.org/api v0.14.0

replace google.golang.org/genproto => google.golang.org/genproto v0.0.0-20191115194625-c23dd37a84c9

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

replace k8s.io/api => github.com/kmodules/api v0.18.10-0.20200922195318-d60fe725dea0

replace k8s.io/apimachinery => github.com/kmodules/apimachinery v0.19.0-alpha.0.0.20200922195535-0c9a1b86beec

replace k8s.io/apiserver => github.com/kmodules/apiserver v0.18.10-0.20200922195747-1bd1cc8f00d1

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.18.9

replace k8s.io/client-go => github.com/kmodules/k8s-client-go v0.18.10-0.20200922201634-73fedf3d677e

replace k8s.io/component-base => k8s.io/component-base v0.18.9

replace k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200410145947-61e04a5be9a6

replace k8s.io/kubernetes => github.com/kmodules/kubernetes v1.19.0-alpha.0.0.20200922200158-8b13196d8dc4

replace k8s.io/utils => k8s.io/utils v0.0.0-20200324210504-a9aa75ae1b89
