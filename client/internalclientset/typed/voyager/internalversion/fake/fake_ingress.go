/*
Copyright 2017 The Stash Authors.

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

package fake

import (
	voyager "github.com/appscode/voyager/apis/voyager"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeIngresses implements IngressInterface
type FakeIngresses struct {
	Fake *FakeVoyager
	ns   string
}

var ingressesResource = schema.GroupVersionResource{Group: "voyager.appscode.com", Version: "", Resource: "ingresses"}

var ingressesKind = schema.GroupVersionKind{Group: "voyager.appscode.com", Version: "", Kind: "Ingress"}

func (c *FakeIngresses) Create(ingress *voyager.Ingress) (result *voyager.Ingress, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(ingressesResource, c.ns, ingress), &voyager.Ingress{})

	if obj == nil {
		return nil, err
	}
	return obj.(*voyager.Ingress), err
}

func (c *FakeIngresses) Update(ingress *voyager.Ingress) (result *voyager.Ingress, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(ingressesResource, c.ns, ingress), &voyager.Ingress{})

	if obj == nil {
		return nil, err
	}
	return obj.(*voyager.Ingress), err
}

func (c *FakeIngresses) UpdateStatus(ingress *voyager.Ingress) (*voyager.Ingress, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(ingressesResource, "status", c.ns, ingress), &voyager.Ingress{})

	if obj == nil {
		return nil, err
	}
	return obj.(*voyager.Ingress), err
}

func (c *FakeIngresses) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(ingressesResource, c.ns, name), &voyager.Ingress{})

	return err
}

func (c *FakeIngresses) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(ingressesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &voyager.IngressList{})
	return err
}

func (c *FakeIngresses) Get(name string, options v1.GetOptions) (result *voyager.Ingress, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(ingressesResource, c.ns, name), &voyager.Ingress{})

	if obj == nil {
		return nil, err
	}
	return obj.(*voyager.Ingress), err
}

func (c *FakeIngresses) List(opts v1.ListOptions) (result *voyager.IngressList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(ingressesResource, ingressesKind, c.ns, opts), &voyager.IngressList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &voyager.IngressList{}
	for _, item := range obj.(*voyager.IngressList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested ingresses.
func (c *FakeIngresses) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(ingressesResource, c.ns, opts))

}

// Patch applies the patch and returns the patched ingress.
func (c *FakeIngresses) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *voyager.Ingress, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(ingressesResource, c.ns, name, data, subresources...), &voyager.Ingress{})

	if obj == nil {
		return nil, err
	}
	return obj.(*voyager.Ingress), err
}
