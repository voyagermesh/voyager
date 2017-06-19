package clientset

// These imports are the API groups the client will support.
import (
	"fmt"

	_ "github.com/appscode/voyager/api/install"
	kapi "k8s.io/client-go/pkg/api"
)

func init() {
	if missingVersions := kapi.Registry.ValidateEnvRequestedVersions(); len(missingVersions) != 0 {
		panic(fmt.Sprintf("KUBE_API_VERSIONS contains versions that are not installed: %q.", missingVersions))
	}
}
