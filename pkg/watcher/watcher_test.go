package watcher

import (
	"testing"

	aci "github.com/appscode/voyager/api"
	"github.com/appscode/voyager/test/testframework"
	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
)

func init() {
	testframework.Initialize()
}

func TestEnsureResource(t *testing.T) {
	w := &Watcher{
		KubeClient: clientset.NewSimpleClientset(
			&extensions.ThirdPartyResource{
				ObjectMeta: apiv1.ObjectMeta{
					Name: "foo",
				},
				Versions: []extensions.APIVersion{
					{
						Name: "v1",
					},
				},
			},
		),
	}
	w.ensureResource()

	data, err := w.KubeClient.Extensions().ThirdPartyResources().List(apiv1.ListOptions{})
	assert.Nil(t, err)
	if data == nil {
		t.Fatal("Item list should not be nil")
	}
	assert.Equal(t, 3, len(data.Items))

	_, err = w.KubeClient.Extensions().ThirdPartyResources().Get("ingress." + aci.V1beta1SchemeGroupVersion.Group)
	assert.Nil(t, err)

	_, err = w.KubeClient.Extensions().ThirdPartyResources().Get("certificate." + aci.V1beta1SchemeGroupVersion.Group)
	assert.Nil(t, err)
}
