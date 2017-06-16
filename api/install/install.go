package install

import (
	aci "github.com/appscode/voyager/api"
	"k8s.io/apimachinery/pkg/apimachinery/announced"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/pkg/api"
)

func init() {
	if err := announced.NewGroupMetaFactory(
		&announced.GroupMetaFactoryArgs{
			GroupName:                  aci.GroupName,
			VersionPreferenceOrder:     []string{aci.V1beta1SchemeGroupVersion.Version},
			ImportPrefix:               "github.com/appscode/voyager/api",
			RootScopedKinds:            sets.NewString("ThirdPartyResource"),
			AddInternalObjectsToScheme: aci.AddToScheme,
		},
		announced.VersionToSchemeFunc{
			aci.V1beta1SchemeGroupVersion.Version: aci.V1beta1AddToScheme,
		},
	).Announce(api.GroupFactoryRegistry).RegisterAndEnable(api.Registry, api.Scheme); err != nil {
		panic(err)
	}
}
