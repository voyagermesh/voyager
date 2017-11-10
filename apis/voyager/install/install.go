package install

import (
	"github.com/appscode/voyager/apis/voyager"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"k8s.io/apimachinery/pkg/apimachinery/announced"
	"k8s.io/apimachinery/pkg/apimachinery/registered"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Install registers the API group and adds types to a scheme
func Install(groupFactoryRegistry announced.APIGroupFactoryRegistry, registry *registered.APIRegistrationManager, scheme *runtime.Scheme) {
	if err := announced.NewGroupMetaFactory(
		&announced.GroupMetaFactoryArgs{
			GroupName:                  voyager.GroupName,
			VersionPreferenceOrder:     []string{api.SchemeGroupVersion.Version},
			RootScopedKinds:            sets.NewString("CustomResourceDefinition"),
			AddInternalObjectsToScheme: voyager.AddToScheme,
		},
		announced.VersionToSchemeFunc{
			api.SchemeGroupVersion.Version: api.AddToScheme,
		},
	).Announce(groupFactoryRegistry).RegisterAndEnable(registry, scheme); err != nil {
		panic(err)
	}
}
