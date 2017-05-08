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
	_, err := i.t.KubeClient.Core().ReplicationControllers(testServerRc.Namespace).Create(testServerRc)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return errors.New().WithCause(err).Internal()
	}

	_, err = i.t.KubeClient.Core().Services(testServerSvc.Namespace).Create(testServerSvc)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return errors.New().WithCause(err).Internal()
	}

	for it := 0; it < maxRetries; it++ {
		ep, err := i.t.KubeClient.Core().Endpoints(testServerSvc.Namespace).Get(testServerSvc.Name)
		if err == nil {
			if len(ep.Subsets) > 0 {
				if len(ep.Subsets[0].Addresses) > 0 {
					break
				}
			}
		}
		log.Infoln("Waiting for endpoint to be ready for testServer")
		time.Sleep(time.Second * 20)
	}

	log.Infoln("Ingress Test Setup Complete")
	return nil
}

func (i *IngressTestSuit) cleanUp() {
	if i.t.Config.Cleanup {
		i.t.KubeClient.Core().Services(testServerSvc.Namespace).Delete(testServerRc.Name, &api.DeleteOptions{
			OrphanDependents: &i.t.Config.Cleanup,
		})
		i.t.KubeClient.Core().ReplicationControllers(testServerSvc.Namespace).Delete(testServerSvc.Name, &api.DeleteOptions{
			OrphanDependents: &i.t.Config.Cleanup,
		})
	}
}

func (i *IngressTestSuit) runTests() error {
	ingType := reflect.ValueOf(i)
	serializedMethodName := make([]string, 0)
	if len(i.t.Config.RunOnly) > 0 {
		serializedMethodName = append(serializedMethodName, "TestIngress"+i.t.Config.RunOnly)
	} else {
		for it := 0; it < ingType.NumMethod(); it++ {
			method := ingType.Type().Method(it)
			if strings.HasPrefix(method.Name, "TestIngress") {
				if strings.Contains(method.Name, "Ensure") {
					serializedMethodName = append([]string{method.Name}, serializedMethodName...)
				} else {
					serializedMethodName = append(serializedMethodName, method.Name)
				}
			}
		}
	}

	for _, name := range serializedMethodName {
		log.Infoln("================== Running Test ====================", name)
		shouldCall := ingType.MethodByName(name)
		if shouldCall.IsValid() {
			results := shouldCall.Call([]reflect.Value{})
			if len(results) == 1 {
				err, castOk := results[0].Interface().(error)
				if castOk {
					return err
				}
			}
			log.Infoln("Wait a bit for things to be clean up inside cluster")
			time.Sleep(time.Second * 20)
		}
	}
	return nil
}
