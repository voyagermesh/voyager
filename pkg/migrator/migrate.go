package migrator

import (
	"errors"
	"fmt"
	"time"

	"github.com/appscode/go/log"
	apiext_util "github.com/appscode/kutil/apiextensions/v1beta1"
	"github.com/appscode/voyager/apis/voyager"
	api "github.com/appscode/voyager/apis/voyager/v1beta1"
	"github.com/hashicorp/go-version"
	extensions "k8s.io/api/extensions/v1beta1"
	kext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	kext_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type migrationState struct {
	tprRegDeleted bool
	crdCreated    bool
}

type migrator struct {
	kubeClient kubernetes.Interface
	crdClient  kext_cs.ApiextensionsV1beta1Interface

	migrationState *migrationState
}

func NewMigrator(kubeClient kubernetes.Interface, apiExtKubeClient kext_cs.ApiextensionsV1beta1Interface) *migrator {
	return &migrator{
		migrationState: &migrationState{},
		kubeClient:     kubeClient,
		crdClient:      apiExtKubeClient,
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
			voyager.ResourceNameIngress+"."+api.SchemeGroupVersion.Group,
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
			voyager.ResourceNameCertificate+"."+api.SchemeGroupVersion.Group,
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
		name := resourceName + "." + api.SchemeGroupVersion.Group
		if err := tprClient.Delete(name, &metav1.DeleteOptions{}); err != nil {
			return fmt.Errorf("failed to remove %s TPR", name)
		}
		return nil
	}

	if err := deleteTPR(voyager.ResourceNameCertificate); err != nil {
		return err
	}
	if err := deleteTPR(voyager.ResourceNameIngress); err != nil {
		return err
	}

	return nil
}

func (m *migrator) createCRDs() error {
	crds := []*kext.CustomResourceDefinition{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   voyager.ResourceTypeIngress + "." + api.SchemeGroupVersion.Group,
				Labels: map[string]string{"app": "voyager"},
			},
			Spec: kext.CustomResourceDefinitionSpec{
				Group:   voyager.GroupName,
				Version: api.SchemeGroupVersion.Version,
				Scope:   kext.NamespaceScoped,
				Names: kext.CustomResourceDefinitionNames{
					Singular:   voyager.ResourceNameIngress,
					Plural:     voyager.ResourceTypeIngress,
					Kind:       voyager.ResourceKindIngress,
					ShortNames: []string{"ing"},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   voyager.ResourceTypeCertificate + "." + api.SchemeGroupVersion.Group,
				Labels: map[string]string{"app": "voyager"},
			},
			Spec: kext.CustomResourceDefinitionSpec{
				Group:   voyager.GroupName,
				Version: api.SchemeGroupVersion.Version,
				Scope:   kext.NamespaceScoped,
				Names: kext.CustomResourceDefinitionNames{
					Singular:   voyager.ResourceNameCertificate,
					Plural:     voyager.ResourceTypeCertificate,
					Kind:       voyager.ResourceKindCertificate,
					ShortNames: []string{"cert"},
				},
			},
		},
	}
	for _, crd := range crds {
		_, err := m.crdClient.CustomResourceDefinitions().Get(crd.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			_, err = m.crdClient.CustomResourceDefinitions().Create(crd)
			if err != nil {
				return err
			}
		}
	}
	return apiext_util.WaitForCRDReady(m.kubeClient.CoreV1().RESTClient(), crds)
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
	deleteCRD := func(resourceType string) error {
		name := resourceType + "." + api.SchemeGroupVersion.Group
		err := m.crdClient.CustomResourceDefinitions().Delete(name, &metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf(`failed to delete CRD "%s""`, name)
		}
		return nil
	}

	if err := deleteCRD(voyager.ResourceTypeIngress); err != nil {
		return err
	}
	if err := deleteCRD(voyager.ResourceTypeCertificate); err != nil {
		return err
	}
	return nil
}

func (m *migrator) createTPRs() error {
	if err := m.createTPR(voyager.ResourceNameIngress); err != nil {
		return err
	}
	if err := m.createTPR(voyager.ResourceNameCertificate); err != nil {
		return err
	}
	return nil
}

func (m *migrator) createTPR(resourceName string) error {
	name := resourceName + "." + api.SchemeGroupVersion.Group
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
				Name: api.SchemeGroupVersion.Version,
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
