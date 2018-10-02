package clientcmd

import (
	"net"
	"os"

	"github.com/appscode/kutil/meta"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// BuildConfigFromFlags is a helper function that builds configs from a master
// url or a kubeconfig filepath. These are passed in as command line flags for cluster
// components. Warnings should reflect this usage. If neither masterUrl or kubeconfigPath
// are passed in we fallback to inClusterConfig. If inClusterConfig fails, we fallback
// to the default config.
func BuildConfigFromFlags(masterUrl, kubeconfigPath string) (*rest.Config, error) {
	return fix(clientcmd.BuildConfigFromFlags(masterUrl, kubeconfigPath))
}

// BuildConfigFromKubeconfigGetter is a helper function that builds configs from a master
// url and a kubeconfigGetter.
func BuildConfigFromKubeconfigGetter(masterUrl string, kubeconfigGetter clientcmd.KubeconfigGetter) (*rest.Config, error) {
	return fix(clientcmd.BuildConfigFromKubeconfigGetter(masterUrl, kubeconfigGetter))
}

func BuildConfigFromContext(kubeconfigPath, contextName string) (*rest.Config, error) {
	var loader clientcmd.ClientConfigLoader
	if kubeconfigPath == "" {
		if meta.PossiblyInCluster() {
			return rest.InClusterConfig()
		}
		rules := clientcmd.NewDefaultClientConfigLoadingRules()
		rules.DefaultClientConfig = &clientcmd.DefaultClientConfig
		loader = rules
	} else {
		loader = &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath}
	}
	overrides := &clientcmd.ConfigOverrides{
		CurrentContext: contextName,
	}
	return fix(clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides).ClientConfig())
}

func ClientFromContext(kubeconfigPath, contextName string) (kubernetes.Interface, error) {
	cfg, err := BuildConfigFromContext(kubeconfigPath, contextName)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}

func NamespaceFromContext(kubeconfigPath, contextName string) (string, error) {
	kConfig, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return "", err
	}
	ctx, found := kConfig.Contexts[contextName]
	if !found {
		return "", errors.Errorf("context %s not found in kubeconfig file %s", contextName, kubeconfigPath)
	}
	return ctx.Namespace, nil
}

func fix(cfg *rest.Config, err error) (*rest.Config, error) {
	return FixAKS(cfg), err
}

func FixAKS(cfg *rest.Config) *rest.Config {
	if cfg == nil {
		return nil
	}

	// ref: https://github.com/kubernetes/client-go/blob/kubernetes-1.11.3/rest/config.go#L309
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if len(host) == 0 || len(port) == 0 {
		return cfg
	}
	if cfg.Host != "https://"+net.JoinHostPort(host, port) &&
		cfg.Host != "https://kubernetes.default.svc" &&
		cfg.Host != "https://kubernetes.default.svc:443" {
		return cfg
	}

	if cert, err := meta.APIServerCertificate(cfg); err == nil {
		if host, err := meta.TestAKS(cert); err == nil {
			cfg.Host = "https://" + host
		}
	}
	return cfg
}
