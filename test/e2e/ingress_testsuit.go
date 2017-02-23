package e2e

import (
	"fmt"

	"go/parser"
	"go/token"

	"github.com/appscode/errors"
	"github.com/appscode/log"
	"k8s.io/kubernetes/pkg/api"
)

type IngressTestSuit struct {
	t TestSuit
}

func NewIngressTestSuit(t TestSuit) *IngressTestSuit {
	return &IngressTestSuit{
		t: t,
	}
}

func (i *IngressTestSuit) Test() error {
	/*if err := i.setUp(); err != nil {
		return err
	}
	defer i.cleanUp()*/

	if err := i.runTests(); err != nil {
		return err
	}
	log.Infoln("Ingress Test Passed")
	return nil
}

func (i *IngressTestSuit) setUp() error {
	_, err := i.t.KubeClient.Core().ReplicationControllers("default").Create(testServerRc)
	if err == nil {
		return errors.New().WithCause(err).Internal()
	}

	_, err = i.t.KubeClient.Core().Services("default").Create(testServerSvc)
	if err == nil {
		return errors.New().WithCause(err).Internal()
	}

	log.Infoln("Ingress Test Setup Complete")
	return nil
}

func (i *IngressTestSuit) cleanUp() {
	i.t.KubeClient.Core().Services("default").Delete(testServerRc.Name, &api.DeleteOptions{})
	i.t.KubeClient.Core().ReplicationControllers("default").Delete(testServerSvc.Name, &api.DeleteOptions{})
}

func (i *IngressTestSuit) runTests() error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "ingress.go", nil, parser.SpuriousErrors)
	if err != nil {
		return err
	}

	for _, s := range f.Decls {
		fmt.Println(s.Pos().IsValid())
	}

	/*ingType := reflect.ValueOf(i)
	fmt.Println("=================================", ingType.NumMethod())
	for i := 0; i < ingType.NumMethod(); i++ {
		method := ingType.Type().Method(i)
		results := ingType.MethodByName(method.Name).Call([]reflect.Value{})
		if len(results) == 1 {
			return results[0].Interface().(error)
		}
	}*/
	return nil
}
