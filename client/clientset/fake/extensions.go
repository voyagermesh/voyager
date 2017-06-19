package fake

import (
	"github.com/appscode/voyager/api"
	"github.com/appscode/voyager/client/clientset"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	kapi "k8s.io/client-go/pkg/api"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/testing"
)

type FakeExtensionClient struct {
	*testing.Fake
}

var _ clientset.ExtensionInterface = &FakeExtensionClient{}

func NewFakeExtensionClient(objects ...runtime.Object) *FakeExtensionClient {
	o := testing.NewObjectTracker(kapi.Registry, kapi.Scheme, kapi.Codecs.UniversalDecoder())
	for _, obj := range objects {
		if obj.GetObjectKind().GroupVersionKind().Group == api.GroupName {
			if err := o.Add(obj); err != nil {
				panic(err)
			}
		}
	}

	fakePtr := testing.Fake{}
	fakePtr.AddReactor("*", "*", testing.ObjectReaction(o, kapi.Registry.RESTMapper()))

	fakePtr.AddWatchReactor("*", testing.DefaultWatchReactor(watch.NewFake(), nil))

	return &FakeExtensionClient{&fakePtr}
}

func (c *FakeExtensionClient) Ingress(namespace string) clientset.IngressInterface {
	return &FakeIngress{c.Fake, namespace}
}

func (c *FakeExtensionClient) Certificate(ns string) clientset.CertificateInterface {
	return &FakeCertificate{c.Fake, ns}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeExtensionClient) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
