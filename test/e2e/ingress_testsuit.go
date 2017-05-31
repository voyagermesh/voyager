package e2e

import (
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/log"
	"k8s.io/kubernetes/pkg/api"
	k8serr "k8s.io/kubernetes/pkg/api/errors"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

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
		return errors.New().WithCause(err).Err()
	}

	_, err = i.t.KubeClient.Core().Services(testServerSvc.Namespace).Create(testServerSvc)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return errors.New().WithCause(err).Err()
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
		log.Infoln("Cleaning up Test Resources")
		i.t.KubeClient.Core().Services(testServerSvc.Namespace).Delete(testServerSvc.Name, &api.DeleteOptions{})
		rc, err := i.t.KubeClient.Core().ReplicationControllers(testServerRc.Namespace).Get(testServerRc.Name)
		if err == nil {
			rc.Spec.Replicas = 0
			i.t.KubeClient.Core().ReplicationControllers(testServerRc.Namespace).Update(rc)
			time.Sleep(time.Second * 5)
		}
		i.t.KubeClient.Core().ReplicationControllers(testServerRc.Namespace).Delete(testServerRc.Name, &api.DeleteOptions{})
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

	startTime := time.Now()

	errChan := make(chan error)
	var wg sync.WaitGroup
	limit := make(chan bool, i.t.Config.MaxConcurrentTest)
	for _, name := range serializedMethodName {
		shouldCall := ingType.MethodByName(name)
		if shouldCall.IsValid() {
			limit <- true
			wg.Add(1)
			// Run Test in separate goroutine
			go func(fun reflect.Value, n string) {
				defer func() {
					<-limit
					log.Infoln("Test", n, "FINISHED")
					wg.Done()
				}()

				log.Infoln("================== Running Test ====================", n)
				results := fun.Call([]reflect.Value{})
				if len(results) == 1 {
					err, castOk := results[0].Interface().(error)
					if castOk {
						if err != nil {
							log.Infoln("Test", n, "FAILED with Cause", err)
							errChan <- errors.FromErr(err).WithMessagef("Test Name %s", n).Err()
						}
					}
				}
			}(shouldCall, name)
		}
	}

	// ReadLoop
	errs := make([]error, 0)
	go func() {
		for err := range errChan {
			if err != nil {
				errs = append(errs, err)
			}
		}
	}()

	// Wait For All to Be DONE
	wg.Wait()

	log.Infoln("======================================")
	log.Infoln("TOTAL", len(serializedMethodName))
	log.Infoln("PASSED", len(serializedMethodName) - len(errs))
	log.Infoln("FAILED", len(errs))
	log.Infoln("Time Elapsed", time.Since(startTime).Minutes(), "minutes")
	log.Infoln("======================================")
	if len(errs) > 0 {
		for _, err := range errs {
			if err != nil {
				log.Infoln("Log\n", err)
			}
		}
		return errors.New().WithMessage("Test FAILED").WithCause(errs[0]).Err()
	}
	return nil
}
