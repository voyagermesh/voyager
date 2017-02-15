package app

import (
	"testing"

	acw "github.com/appscode/k8s-addons/pkg/watcher"
	"github.com/stretchr/testify/assert"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
	aci "github.com/appscode/k8s-addons/api"
)

func TestEnsureResource(t *testing.T) {
	w := &Watcher{
		Watcher: acw.Watcher{
			Client: clientset.NewSimpleClientset(
				&extensions.ThirdPartyResource{
					ObjectMeta: kapi.ObjectMeta{
						Name: "foo",
					},
					Versions: []extensions.APIVersion{
						{
							Name: "v1",
						},
					},
				},
			),
		},
	}
	w.ensureResource()

	data, err := w.Client.Extensions().ThirdPartyResources().List(kapi.ListOptions{})
	assert.Nil(t, err)
	if data == nil {
		t.Fatal("Item list should not be nil")
	}
	assert.Equal(t, 3, len(data.Items))

	_, err = w.Client.Extensions().ThirdPartyResources().Get("ingress." + aci.V1beta1SchemeGroupVersion.Group)
	assert.Nil(t, err)

	_, err = w.Client.Extensions().ThirdPartyResources().Get("certificate." + aci.V1beta1SchemeGroupVersion.Group)
	assert.Nil(t, err)
}
