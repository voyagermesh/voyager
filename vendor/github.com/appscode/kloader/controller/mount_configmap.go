package controller

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/appscode/go/ioutil"
	"github.com/appscode/go/log"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const maxRetries = 5

type configMapMounter struct {
	Source        *apiv1.ObjectReference
	mountLocation string
	cmdFile       string

	kubeConfig *rest.Config
	KubeClient clientset.Interface

	queue    workqueue.RateLimitingInterface
	informer cache.Controller
	indexer  cache.Indexer
}

func NewConfigMapMounter(kubeConfig *rest.Config, configMap, mountDir, cmd string, resyncPeriod time.Duration) *configMapMounter {
	configMapParts := strings.SplitN(strings.TrimSpace(configMap), ".", 2)
	source := &apiv1.ObjectReference{
		Name: configMapParts[0],
	}
	if len(configMapParts) == 2 {
		source.Namespace = configMapParts[1]
	} else {
		source.Namespace = namespace()
	}

	client := clientset.NewForConfigOrDie(kubeConfig)
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	handler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			incUpdateReceivedCounter()
			if key, err := cache.MetaNamespaceKeyFunc(obj); err == nil {
				log.Infoln("Queued Add event")
				queue.Add(key)
			} else {
				log.Infoln(err)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			incUpdateReceivedCounter()
			if oldMap, oldOK := old.(*apiv1.ConfigMap); oldOK {
				if newMap, newOK := new.(*apiv1.ConfigMap); newOK {
					if !reflect.DeepEqual(oldMap.Data, newMap.Data) {
						if key, err := cache.MetaNamespaceKeyFunc(new); err == nil {
							log.Infoln("Queued Update event", key)
							queue.Add(key)
						} else {
							log.Infoln(err)
						}
					}
				}
			}
		},
	}

	indexer, informer := cache.NewIndexerInformer(
		&cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
				return client.CoreV1().ConfigMaps(source.Namespace).List(metav1.ListOptions{
					FieldSelector: fields.OneTermEqualSelector("metadata.name", source.Name).String(),
				})
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return client.CoreV1().ConfigMaps(source.Namespace).Watch(metav1.ListOptions{
					FieldSelector: fields.OneTermEqualSelector("metadata.name", source.Name).String(),
				})
			},
		},
		&apiv1.ConfigMap{},
		resyncPeriod,
		handler,
		cache.Indexers{},
	)

	return &configMapMounter{
		Source:        source,
		mountLocation: strings.TrimSuffix(mountDir, "/"),
		cmdFile:       cmd,
		kubeConfig:    kubeConfig,
		KubeClient:    client,
		queue:         queue,
		informer:      informer,
		indexer:       indexer,
	}
}

func (c *configMapMounter) Run() {
	go c.informer.Run(wait.NeverStop)
	wait.Until(c.runWorker, time.Second, wait.NeverStop)
}

func (c *configMapMounter) runWorker() {
	for c.processNextItem() {
		// continue looping
	}
}

func (c *configMapMounter) processNextItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	err := c.processItem(key.(string))
	if err == nil {
		c.queue.Forget(key)
	} else if c.queue.NumRequeues(key) < maxRetries {
		log.Infof("Error processing %s (will retry): %v\n", key, err)
		c.queue.AddRateLimited(key)
	} else {
		log.Infof("Error processing %s (giving up): %v\n", key, err)
		c.queue.Forget(key)
	}

	return true
}

func (c *configMapMounter) processItem(key string) error {
	log.Infof("Processing change to ConfigMap %s\n", key)

	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		return fmt.Errorf("error fetching object with key %s from store: %v", key, err)
	}

	if !exists {
		log.Infof("Not exists: ConfigMap %s\n", key)
		return nil
	}

	// handle the event
	if obj.(*apiv1.ConfigMap) != nil {
		c.Mount(obj.(*apiv1.ConfigMap))
	}
	if len(c.cmdFile) > 0 {
		runCmd(c.cmdFile)
	}
	return nil
}

func (c *configMapMounter) Mount(configMap *apiv1.ConfigMap) {
	payload := make(map[string]ioutil.FileProjection)
	for k, v := range configMap.Data {
		payload[k] = ioutil.FileProjection{Mode: 0777, Data: []byte(v)}
	}

	writer, err := ioutil.NewAtomicWriter(c.mountLocation)
	if err != nil {
		log.Fatalln("Failed to Create atomic writer, Cause", err)
	}
	changed, err := writer.Write(payload)
	if err != nil {
		log.Fatalln("Failed to Mount ConfigMap, Cause", err)
	}
	if changed {
		incMountCounter()
	}
}
