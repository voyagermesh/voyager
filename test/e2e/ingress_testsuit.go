package e2e

import (
	"reflect"
	"strings"

	"time"

	"github.com/appscode/errors"
	"github.com/appscode/log"
	"k8s.io/kubernetes/pkg/api"
	k8serr "k8s.io/kubernetes/pkg/api/errors"
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
	if err := i.setUp(); err != nil {
		return err
	}
	defer i.cleanUp()

	if err := i.runTests(); err != nil {
		return err
	}
	log.Infoln("Ingress Test Passed")
	return nil
}

func (i *IngressTestSuit) setUp() error {
	_, err := i.t.KubeClient.Core().ReplicationControllers("default").Create(testServerRc)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return errors.New().WithCause(err).Internal()
	}

	_, err = i.t.KubeClient.Core().Services("default").Create(testServerSvc)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return errors.New().WithCause(err).Internal()
	}
	time.Sleep(time.Second * 30)
	log.Infoln("Ingress Test Setup Complete")
	return nil
}

func (i *IngressTestSuit) cleanUp() {
	if i.t.config.Cleanup {
		i.t.KubeClient.Core().Services("default").Delete(testServerRc.Name, &api.DeleteOptions{
			OrphanDependents: &i.t.config.Cleanup,
		})
		i.t.KubeClient.Core().ReplicationControllers("default").Delete(testServerSvc.Name, &api.DeleteOptions{})
	}
}

func (i *IngressTestSuit) runTests() error {
	ingType := reflect.ValueOf(i)
	serializedMethodName := make([]string, 0)
	for i := 0; i < ingType.NumMethod(); i++ {
		method := ingType.Type().Method(i)
		if strings.HasPrefix(method.Name, "TestIngress") {
			if strings.Contains(method.Name, "Ensure") {
				serializedMethodName = append([]string{method.Name}, serializedMethodName...)
			} else {
				serializedMethodName = append(serializedMethodName, method.Name)
			}
		}
	}

	for _, name := range serializedMethodName {
		log.Infoln("Running Test", name)
		shouldCall := ingType.MethodByName(name)
		if shouldCall.IsValid() {
			results := shouldCall.Call([]reflect.Value{})
			if len(results) == 1 {
				err, castOk := results[0].Interface().(error)
				if castOk {
					return err
				}
			}
		}
	}
	return nil
}
