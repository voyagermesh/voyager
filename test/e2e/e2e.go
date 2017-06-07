package e2e

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/appscode/log"
	"github.com/appscode/voyager/client/clientset"
	acs "github.com/appscode/voyager/client/clientset"
	acw "github.com/appscode/voyager/pkg/watcher"
	"github.com/appscode/voyager/test/testframework"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
)

type TestSuit struct {
	Config     testframework.E2EConfig
	KubeClient internalclientset.Interface
	ExtClient  clientset.ExtensionInterface
	Voyager    *acw.Watcher
}

func init() {
	testframework.Initialize()
}

func NewE2ETestSuit() *TestSuit {
	ensureE2EConfigs()
	c, err := getKubeConfig()
	if err != nil {
		log.Fatalln("Failed to load Kube Config", err)
	}
	return &TestSuit{
		Config: testframework.TestContext.E2EConfigs,
		Voyager: &acw.Watcher{
			KubeClient:        internalclientset.NewForConfigOrDie(c),
			ExtClient:         acs.NewForConfigOrDie(c),
			SyncPeriod:        time.Minute * 5,
			ProviderName:      testframework.TestContext.E2EConfigs.ProviderName,
			ClusterName:       testframework.TestContext.E2EConfigs.ClusterName,
			LoadbalancerImage: testframework.TestContext.E2EConfigs.LoadbalancerImageName,
			IngressClass:      testframework.TestContext.E2EConfigs.IngressClass,
		},
		KubeClient: internalclientset.NewForConfigOrDie(c),
		ExtClient:  acs.NewForConfigOrDie(c),
	}
}

func (t *TestSuit) Run() error {
	if !t.Config.InCluster {
		go t.Voyager.Run()
	}
	defer time.Sleep(time.Second * 30)
	defer log.Flush()
	// Wait some time to initialize voyager watcher
	time.Sleep(time.Second * 10)
	ingTestSuit := NewIngressTestSuit(*t)
	if err := ingTestSuit.Test(); err != nil {
		return err
	}
	return nil
}

func ensureE2EConfigs() {
	if testframework.TestContext.E2EConfigs.ProviderName == "" ||
		testframework.TestContext.E2EConfigs.ClusterName == "" ||
		testframework.TestContext.E2EConfigs.LoadbalancerImageName == "" {
		log.Fatalln("Required flag not provided.")
	}
}

func getKubeConfig() (*restclient.Config, error) {
	if len(testframework.TestContext.E2EConfigs.KubeConfig) == 0 {
		if _, err := os.Stat(clientcmd.RecommendedHomeFile); err == nil {
			testframework.TestContext.E2EConfigs.KubeConfig = clientcmd.RecommendedHomeFile
		} else {
			k8sConfig := os.Getenv("TEST_KUBE_CONFIG")
			k8sConfigDir := os.TempDir() + "/.kube/config"
			err := ioutil.WriteFile(k8sConfigDir, []byte(k8sConfig), os.ModePerm)
			if err == nil {
				testframework.TestContext.E2EConfigs.KubeConfig = k8sConfigDir
			}
		}
	}

	return clientcmd.BuildConfigFromFlags(testframework.TestContext.E2EConfigs.Master, testframework.TestContext.E2EConfigs.KubeConfig)
}
