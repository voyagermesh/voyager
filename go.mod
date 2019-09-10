module github.com/appscode/voyager

go 1.12

require (
	cloud.google.com/go v0.39.0
	github.com/Azure/azure-pipeline-go v0.1.9 // indirect
	github.com/Azure/azure-sdk-for-go v31.1.0+incompatible
	github.com/Azure/azure-storage-blob-go v0.6.0 // indirect
	github.com/Azure/go-autorest/autorest v0.5.0
	github.com/Azure/go-autorest/autorest/adal v0.2.0
	github.com/Azure/go-autorest/autorest/azure/auth v0.1.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.2.0
	github.com/JamesClonk/vultr v2.0.0+incompatible // indirect
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/akamai/AkamaiOPEN-edgegrid-golang v0.8.0 // indirect
	github.com/appscode/go v0.0.0-20190808133642-1d4ef1f1c1e0
	github.com/appscode/hello-grpc v0.0.0-20190207041230-eea009cbf42e
	github.com/appscode/pat v0.0.0-20170521084856-48ff78925b79
	github.com/aws/aws-sdk-go v1.20.20
	github.com/benbjohnson/clock v0.0.0-20161215174838-7dc76406b6d3
	github.com/census-instrumentation/opencensus-proto v0.2.0 // indirect
	github.com/cloudflare/cloudflare-go v0.8.5 // indirect
	github.com/codeskyblue/go-sh v0.0.0-20190412065543-76bd3d59ff27
	github.com/coreos/prometheus-operator v0.31.1
	github.com/cpuguy83/go-md2man v1.0.10 // indirect
	github.com/dnsimple/dnsimple-go v0.0.0-20180703121714-35bcc6b47c20 // indirect
	github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/go-openapi/spec v0.19.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/google/go-cmp v0.3.0
	github.com/google/martian v2.1.1-0.20190517191504-25dcb96d9e51+incompatible // indirect
	github.com/gopherjs/gopherjs v0.0.0-20190430165422-3e4dfb77656c // indirect
	github.com/hashicorp/vault/api v1.0.1
	github.com/jackc/pgx v3.3.0+incompatible // indirect
	github.com/json-iterator/go v1.1.6
	github.com/juju/ratelimit v0.0.0-20151125201925-77ed1c8a0121 // indirect
	github.com/lib/pq v1.1.0 // indirect
	github.com/mitchellh/go-ps v0.0.0-20170309133038-4fdf99ab2936
	github.com/moul/http2curl v1.0.0
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/orcaman/concurrent-map v0.0.0-20190314100340-2693aad1ed75
	github.com/ovh/go-ovh v0.0.0-20181109152953-ba5adb4cf014 // indirect
	github.com/pires/go-proxyproto v0.0.0-20190111085350-4d51b51e3bfc
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.4
	github.com/prometheus/common v0.4.1
	github.com/prometheus/haproxy_exporter v0.10.1-0.20190712143102-4cebda7ab705
	github.com/shirou/gopsutil v2.18.12+incompatible
	github.com/smartystreets/assertions v0.0.0-20190401211740-f487f9de1cd3 // indirect
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/stretchr/testify v1.3.0
	github.com/timewasted/linode v0.0.0-20160829202747-37e84520dcf7 // indirect
	github.com/tredoe/osutil v0.0.0-20161130133508-7d3ee1afa71c
	github.com/xenolf/lego v2.5.0+incompatible
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	gomodules.xyz/cert v1.0.0
	google.golang.org/api v0.7.0
	google.golang.org/genproto v0.0.0-20190508193815-b515fa19cec8 // indirect
	google.golang.org/grpc v1.21.1
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/gcfg.v1 v1.2.3
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/warnings.v0 v0.1.1 // indirect
	k8s.io/api v0.0.0-20190503110853-61630f889b3c
	k8s.io/apiextensions-apiserver v0.0.0-20190516231611-bf6753f2aa24
	k8s.io/apimachinery v0.0.0-20190508063446-a3da69d3723c
	k8s.io/apiserver v0.0.0-20190516230822-f89599b3f645
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/kube-aggregator v0.0.0-20190508104018-6d3d96b06d29 // indirect
	k8s.io/kube-openapi v0.0.0-20190502190224-411b2483e503
	k8s.io/utils v0.0.0-20190506122338-8fab8cb257d5
	kmodules.xyz/client-go v0.0.0-20190802200916-043217632b6a
	kmodules.xyz/monitoring-agent-api v0.0.0-20190802203207-a87aa5b2e057
	kmodules.xyz/webhook-runtime v0.0.0-20190802202019-9e77ee949266
	sigs.k8s.io/yaml v1.1.0
)

replace (
	bitbucket.org/ww/goautoneg => gomodules.xyz/goautoneg v0.0.0-20120707110453-a547fc61f48d
	cloud.google.com/go => cloud.google.com/go v0.34.0
	contrib.go.opencensus.io/exporter/ocagent => contrib.go.opencensus.io/exporter/ocagent v0.3.0
	git.apache.org/thrift.git => github.com/apache/thrift v0.12.0
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.4.2+incompatible
	github.com/census-instrumentation/opencensus-proto => github.com/census-instrumentation/opencensus-proto v0.1.0
	github.com/golang/protobuf => github.com/golang/protobuf v1.2.0
	github.com/grpc-ecosystem/grpc-gateway => github.com/gomodules/grpc-gateway v1.3.1-ac
	github.com/miekg/dns => github.com/miekg/dns v1.0.7
	github.com/prometheus/haproxy_exporter => github.com/appscode/haproxy_exporter v0.7.2-0.20190508003714-b4abf52090e2
	github.com/xenolf/lego => github.com/appscode/lego v1.2.2-0.20181215093553-e57a0a1b7259
	go.opencensus.io => go.opencensus.io v0.21.0
	k8s.io/api => k8s.io/api v0.0.0-20190313235455-40a48860b5ab
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190315093550-53c4693659ed
	k8s.io/apimachinery => github.com/kmodules/apimachinery v0.0.0-20190508045248-a52a97a7a2bf
	k8s.io/apiserver => github.com/kmodules/apiserver v0.0.0-20190811223248-5a95b2df4348
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190314001948-2899ed30580f
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190314002645-c892ea32361a
	k8s.io/component-base => k8s.io/component-base v0.0.0-20190314000054-4a91899592f4
	k8s.io/klog => k8s.io/klog v0.3.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20190314000639-da8327669ac5
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20190228160746-b3a7cee44a30
	k8s.io/metrics => k8s.io/metrics v0.0.0-20190314001731-1bd6a4002213
	k8s.io/utils => k8s.io/utils v0.0.0-20190221042446-c2654d5206da
	sigs.k8s.io/structured-merge-diff => sigs.k8s.io/structured-merge-diff v0.0.0-20190302045857-e85c7b244fd2
)
