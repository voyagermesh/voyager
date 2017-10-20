package migrator

import (
	"errors"
	"fmt"
	"time"

	"github.com/appscode/go/log"
	"github.com/appscode/kutil"
	api "github.com/appscode/voyager/apis/voyager"
	api_v1beta1 "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/hashicorp/go-version"
	extensions "k8s.io/api/extensions/v1beta1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
)

type migrationState struct {
	tprRegDeleted bool
	crdCreated    bool
}

type migrator struct {
	kubeClient       clientset.Interface
	apiExtKubeClient apiextensionsclient.Interface

	migrationState *migrationState
}

func NewMigrator(kubeClient clientset.Interface, apiExtKubeClient apiextensionsclient.Interface) *migrator {
	return &migrator{
		migrationState:   &migrationState{},
		kubeClient:       kubeClient,
		apiExtKubeClient: apiExtKubeClient,
	}
}

func (m *migrator) isMigrationNeeded() (bool, error) {
	v, err := m.kubeClient.Discovery().ServerVersion()
	if err != nil {
		return false, err
	}

	ver, err := version.NewVersion(v.String())
	if err != nil {
		return false, err
	}

	mv := ver.Segments()[1]

	if mv == 7 {
		_, err := m.kubeClient.ExtensionsV1beta1().ThirdPartyResources().Get(
			api.ResourceNameIngress+"."+api_v1beta1.SchemeGroupVersion.Group,
			metav1.GetOptions{},
		)
		if err != nil {
			if !kerr.IsNotFound(err) {
				return false, err
			}
		} else {
			return true, nil
		}

		_, err = m.kubeClient.ExtensionsV1beta1().ThirdPartyResources().Get(
			api.ResourceNameCertificate+"."+api_v1beta1.SchemeGroupVersion.Group,
			metav1.GetOptions{},
		)
		if err != nil {
			if !kerr.IsNotFound(err) {
				return false, err
			}
		} else {
			return true, nil
		}
	}

	return false, nil
}

func (m *migrator) RunMigration() error {
	needed, err := m.isMigrationNeeded()
	if err != nil {
		return err
	}

	if needed {
		if err := m.migrateTPR2CRD(); err != nil {
			return m.rollback()
		}
	}

	return nil
}

func (m *migrator) migrateTPR2CRD() error {
	log.Debugln("Performing TPR to CRD migration.")

	log.Debugln("Deleting TPRs.")
	if err := m.deleteTPRs(); err != nil {
		return errors.New("Failed to Delete TPRs")
	}

	m.migrationState.tprRegDeleted = true

	log.Debugln("Creating CRDs.")
	if err := m.createCRDs(); err != nil {
		return errors.New("Failed to create CRDs")
	}

	m.migrationState.crdCreated = true
	return nil
}

func (m *migrator) deleteTPRs() error {
	tprClient := m.kubeClient.ExtensionsV1beta1().ThirdPartyResources()

	deleteTPR := func(resourceName string) error {
		name := resourceName + "." + api_v1beta1.SchemeGroupVersion.Group
		if err := tprClient.Delete(name, &metav1.DeleteOptions{}); err != nil {
			return fmt.Errorf("failed to remove %s TPR", name)
		}
		return nil
	}

	if err := deleteTPR(api.ResourceNameCertificate); err != nil {
		return err
	}
	if err := deleteTPR(api.ResourceNameIngress); err != nil {
		return err
	}

	return nil
}

func (m *migrator) createCRDs() error {
	crds := []*apiextensions.CustomResourceDefinition{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   api.ResourceTypeIngress + "." + api_v1beta1.SchemeGroupVersion.Group,
				Labels: map[string]string{"app": "voyager"},
			},
			Spec: apiextensions.CustomResourceDefinitionSpec{
				Group:   api.GroupName,
				Version: api_v1beta1.SchemeGroupVersion.Version,
				Scope:   apiextensions.NamespaceScoped,
				Names: apiextensions.CustomResourceDefinitionNames{
					Singular:   api.ResourceNameIngress,
					Plural:     api.ResourceTypeIngress,
					Kind:       api.ResourceKindIngress,
					ShortNames: []string{"ing"},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   api.ResourceTypeCertificate + "." + api_v1beta1.SchemeGroupVersion.Group,
				Labels: map[string]string{"app": "voyager"},
			},
			Spec: apiextensions.CustomResourceDefinitionSpec{
				Group:   api.GroupName,
				Version: api_v1beta1.SchemeGroupVersion.Version,
				Scope:   apiextensions.NamespaceScoped,
				Names: apiextensions.CustomResourceDefinitionNames{
					Singular:   api.ResourceNameCertificate,
					Plural:     api.ResourceTypeCertificate,
					Kind:       api.ResourceKindCertificate,
					ShortNames: []string{"cert"},
				},
			},
		},
	}
	for _, crd := range crds {
		_, err := m.apiExtKubeClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crd.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			_, err = m.apiExtKubeClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
			if err != nil {
				return err
			}
		}
	}
	return kutil.WaitForCRDReady(m.kubeClient.CoreV1().RESTClient(), crds)
}

func (m *migrator) rollback() error {
	log.Debugln("Rolling back migration.")

	ms := m.migrationState

	if ms.crdCreated {
		log.Debugln("Deleting CRDs.")
		err := m.deleteCRDs()
		if err != nil {
			return errors.New("Failed to delete CRDs")
		}
	}

	if ms.tprRegDeleted {
		log.Debugln("Creating TPRs.")
		err := m.createTPRs()
		if err != nil {
			return errors.New("Failed to recreate TPRs")
		}

		err = m.waitForTPRsReady()
		if err != nil {
			return errors.New("Failed to be ready TPRs")
		}
	}

	return nil
}

func (m *migrator) deleteCRDs() error {
	crdClient := m.apiExtKubeClient.ApiextensionsV1beta1().CustomResourceDefinitions()

	deleteCRD := func(resourceType string) error {
		name := resourceType + "." + api_v1beta1.SchemeGroupVersion.Group
		err := crdClient.Delete(name, &metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf(`Failed to delete CRD "%s""`, name)
		}
		return nil
	}

	if err := deleteCRD(api.ResourceTypeIngress); err != nil {
		return err
	}
	if err := deleteCRD(api.ResourceTypeCertificate); err != nil {
		return err
	}
	return nil
}

func (m *migrator) createTPRs() error {
	if err := m.createTPR(api.ResourceNameIngress); err != nil {
		return err
	}
	if err := m.createTPR(api.ResourceNameCertificate); err != nil {
		return err
	}
	return nil
}

func (m *migrator) createTPR(resourceName string) error {
	name := resourceName + "." + api_v1beta1.SchemeGroupVersion.Group
	_, err := m.kubeClient.ExtensionsV1beta1().ThirdPartyResources().Get(name, metav1.GetOptions{})
	if !kerr.IsNotFound(err) {
		return err
	}

	thirdPartyResource := &extensions.ThirdPartyResource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "ThirdPartyResource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": "voyager",
			},
		},
		Versions: []extensions.APIVersion{
			{
				Name: api_v1beta1.SchemeGroupVersion.Version,
			},
		},
	}

	_, err = m.kubeClient.ExtensionsV1beta1().ThirdPartyResources().Create(thirdPartyResource)
	return err
}

func (m *migrator) waitForTPRsReady() error {
	labelMap := map[string]string{
		"app": "voyager",
	}

	return wait.Poll(3*time.Second, 10*time.Minute, func() (bool, error) {
		crdList, err := m.kubeClient.ExtensionsV1beta1().ThirdPartyResources().List(metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(labelMap).String(),
		})
		if err != nil {
			return false, err
		}

		if len(crdList.Items) == 3 {
			return true, nil
		}

		return false, nil
	})
}
