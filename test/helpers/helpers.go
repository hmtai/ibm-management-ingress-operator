//
// Copyright 2020 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
package helpers

import (
	goctx "context"
	"fmt"
	"testing"

	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	olmclient "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilwait "k8s.io/apimachinery/pkg/util/wait"

	operator "github.com/IBM/ibm-management-ingress-operator/pkg/apis/operator/v1alpha1"
	"github.com/IBM/ibm-management-ingress-operator/test/config"
)

// CreateTest creates a ManagementIngressOperatorSet instance
func CreateTest(olmClient *olmclient.Clientset, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	// create.ManagementIngressOperatorConfig custom resource
	fmt.Println("--- CREATE:.ManagementIngressOperatorConfigConfig Instance")
	configInstance := newManagementIngressOperatorConfigCR(config.ConfigCrName, namespace)
	err = f.Client.Create(goctx.TODO(), configInstance, &framework.CleanupOptions{TestContext: ctx, Timeout: config.CleanupTimeout, RetryInterval: config.CleanupRetry})
	if err != nil {
		return err
	}

	// create ManagementIngressOperator custom resource
	fmt.Println("--- CREATE: ManagementIngressOperator Instance")
	metaOperatorInstance := newManagementIngressOperatorCR(config.ConfigCrName, namespace)
	err = f.Client.Create(goctx.TODO(), metaOperatorInstance, &framework.CleanupOptions{TestContext: ctx, Timeout: config.CleanupTimeout, RetryInterval: config.CleanupRetry})
	if err != nil {
		return err
	}

	// create ManagementIngressOperatorSet custom resource
	sets := []operator.SetService{}
	sets = append(sets, operator.SetService{
		Name:    "etcd",
		Channel: "singlenamespace-alpha",
		State:   "present",
	}, operator.SetService{
		Name:    "jenkins",
		Channel: "alpha",
		State:   "present",
	})

	setInstance := &operator.ManagementIngressOperatorSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.SetCrName,
			Namespace: namespace,
		},
		Spec: operator.ManagementIngressOperatorSetSpec{
			Services: sets,
		},
	}
	fmt.Println("--- CREATE: ManagementIngressOperatorSet Instance")
	// use TestCtx's create helper to create the object and add a cleanup function for the new object
	err = f.Client.Create(goctx.TODO(), setInstance, &framework.CleanupOptions{TestContext: ctx, Timeout: config.CleanupTimeout, RetryInterval: config.CleanupRetry})
	if err != nil {
		return err
	}
	// wait for all the csv ready
	optMap, err := GetOperators(f, namespace)
	if err != nil {
		return err
	}
	for _, s := range sets {
		opt := optMap[s.Name]
		err = WaitForSubCsvReady(olmClient, metav1.ObjectMeta{Name: opt.Name, Namespace: opt.Namespace})
		if err != nil {
			return err
		}
	}
	err = ValidateCustomeResource(f, namespace)
	if err != nil {
		return err
	}
	return nil
}

// UpdateTest updates a ManagementIngressOperatorSet instance
func UpdateTest(olmClient *olmclient.Clientset, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return err
	}
	setInstance := &operator.ManagementIngressOperatorSet{}
	fmt.Println("--- UPDATE: subscription")
	// Get ManagementIngressOperatorSet instance
	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: config.SetCrName, Namespace: namespace}, setInstance)
	if err != nil {
		return err
	}

	setInstance.Spec.Services[0].Channel = "clusterwide-alpha"
	err = f.Client.Update(goctx.TODO(), setInstance)
	if err != nil {
		return err
	}

	// wait for updated csv ready
	optMap, err := GetOperators(f, namespace)
	if err != nil {
		return err
	}
	opt := optMap[setInstance.Spec.Services[0].Name]
	err = WaitForSubCsvReady(olmClient, metav1.ObjectMeta{Name: opt.Name, Namespace: opt.Namespace})
	if err != nil {
		return err
	}
	return nil
}

//DeleteTest delete a ManagementIngressOperatorSet instance
func DeleteTest(olmClient *olmclient.Clientset, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}
	setInstance := &operator.ManagementIngressOperatorSet{}
	fmt.Println("--- DELETE: subscription")
	// Get ManagementIngressOperatorSet instance
	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: config.SetCrName, Namespace: namespace}, setInstance)
	if err != nil {
		return err
	}
	// Mark first operator state as absent
	setInstance.Spec.Services[0].State = "absent"
	err = f.Client.Update(goctx.TODO(), setInstance)
	if err != nil {
		return err
	}

	optMap, err := GetOperators(f, namespace)
	if err != nil {
		return err
	}
	opt := optMap[setInstance.Spec.Services[0].Name]
	// Waiting for subscription deleted
	err = WaitForSubscriptionDelete(olmClient, metav1.ObjectMeta{Name: opt.Name, Namespace: opt.Namespace})
	if err != nil {
		return err
	}
	return nil
}

// UpdateConfigTest updates a ManagementIngressOperatorConfig instance
func UpdateConfigTest(olmClient *olmclient.Clientset, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return err
	}
	configInstance := &operator.ManagementIngressOperatorConfig{}
	fmt.Println("--- UPDATE: custom resource")
	// Get ManagementIngressOperatorSet instance
	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: config.ConfigCrName, Namespace: namespace}, configInstance)
	if err != nil {
		return err
	}

	configInstance.Spec.Services[0].Spec = map[string]runtime.RawExtension{
		"etcdCluster": {Raw: []byte(`{"size": 3}`)},
	}
	err = f.Client.Update(goctx.TODO(), configInstance)
	if err != nil {
		return err
	}

	err = ValidateCustomeResource(f, namespace)
	if err != nil {
		return err
	}

	return nil
}

// UpdateCatalogTest updates a ManagementIngressOperatorCatalog instance
func UpdateCatalogTest(olmClient *olmclient.Clientset, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return err
	}
	metaOperatorInstance := &operator.ManagementIngressOperatorCatalog{}
	fmt.Println("--- UPDATE: ManagementIngressOperator")

	// Get ManagementIngressOperatorCatalog instance
	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: config.CatalogCrName, Namespace: namespace}, metaOperatorInstance)
	if err != nil {
		return err
	}

	metaOperatorInstance.Spec.Operators[0].Channel = "clusterwide-alpha"
	err = f.Client.Update(goctx.TODO(), metaOperatorInstance)
	if err != nil {
		return err
	}

	// wait for updated csv ready
	optMap, err := GetOperators(f, namespace)
	if err != nil {
		return err
	}
	opt := optMap[metaOperatorInstance.Spec.Operators[0].Name]
	err = WaitForSubCsvReady(olmClient, metav1.ObjectMeta{Name: opt.Name, Namespace: opt.Namespace})
	if err != nil {
		return err
	}
	return nil
}

// GetOperators get a operator list waiting for being installed
func GetOperators(f *framework.Framework, namespace string) (map[string]operator.Operator, error) {
	moInstance := &operator.ManagementIngressOperatorCatalog{}
	lastReason := ""
	waitErr := utilwait.PollImmediate(config.WaitForRetry, config.APITimeout, func() (done bool, err error) {
		err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: config.CatalogCrName, Namespace: namespace}, moInstance)
		if err != nil {
			if errors.IsNotFound(err) {
				lastReason = fmt.Sprintf("Waiting on ManagementIngressOperator instance to be created [ibm-management-ingress-operator]")
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	if waitErr != nil {
		return nil, fmt.Errorf("%v: %s", waitErr, lastReason)
	}
	optMap := make(map[string]operator.Operator)
	for _, v := range moInstance.Spec.Operators {
		optMap[v.Name] = v
	}
	return optMap, nil
}

// WaitForSubCsvReady waits for the subscription and csv create success
func WaitForSubCsvReady(olmClient *olmclient.Clientset, opt metav1.ObjectMeta) error {
	lastReason := ""
	sub := &olmv1alpha1.Subscription{}
	fmt.Println("Waiting for Subscription created [" + opt.Name + "]")
	waitErr := utilwait.PollImmediate(config.WaitForRetry, config.APITimeout, func() (done bool, err error) {
		foundSub, err := olmClient.OperatorsV1alpha1().Subscriptions(opt.Namespace).Get(opt.Name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				lastReason = fmt.Sprintf("Waiting on subscription to be created" + ", Subscription.Name: " + opt.Name + " Subscription.Namespace: " + opt.Namespace)
				return false, nil
			}
			return false, err
		}
		if foundSub.Status.InstalledCSV == "" {
			lastReason = fmt.Sprintf("Waiting on CSV to be installed" + ", Subscription.Name: " + opt.Name + " Subscription.Namespace: " + opt.Namespace)
			return false, nil
		}
		sub = foundSub
		return true, nil
	})
	if waitErr != nil {
		return fmt.Errorf("%v: %s", waitErr, lastReason)
	}

	fmt.Println("Waiting for CSV status succeeded [" + sub.Status.InstalledCSV + "]")
	waitErr = utilwait.PollImmediate(config.WaitForRetry, config.APITimeout, func() (done bool, err error) {
		csv, err := olmClient.OperatorsV1alpha1().ClusterServiceVersions(opt.Namespace).Get(sub.Status.InstalledCSV, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				lastReason = fmt.Sprintf("Waiting on CSV to be created" + ", CSV.Name: " + sub.Status.InstalledCSV + " CSV.Namespace: " + opt.Namespace)
				return false, nil
			}
			return false, err
		}

		// New csv found and phase is succeeeded
		if sub.Status.InstalledCSV == csv.Name && csv.Status.Phase == "Succeeded" {
			return true, nil
		}
		lastReason = fmt.Sprintf("Waiting on CSV status succeeded" + ", CSV.Name: " + sub.Status.InstalledCSV + " CSV.Namespace: " + opt.Namespace)
		return false, nil
	})
	if waitErr != nil {
		return fmt.Errorf("%v: %s", waitErr, lastReason)
	}
	return nil
}

// WaitForSubscriptionDelete waits for the subscription deleted
func WaitForSubscriptionDelete(olmClient *olmclient.Clientset, opt metav1.ObjectMeta) error {
	lastReason := ""
	fmt.Println("Waiting on subscription to be deleted [" + opt.Name + "]")
	waitErr := utilwait.PollImmediate(config.WaitForRetry, config.APITimeout, func() (done bool, err error) {
		_, err = olmClient.OperatorsV1alpha1().Subscriptions(opt.Namespace).Get(opt.Name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		lastReason = fmt.Sprintf("Waiting on subscription to be deleted" + ", Subscription.Name: " + opt.Name + " Subscription.Namespace: " + opt.Namespace)
		return false, nil
	})
	if waitErr != nil {
		return fmt.Errorf("%v: %s", waitErr, lastReason)
	}
	return nil
}

// ValidateCustomeResource check the result of the meta operator config
func ValidateCustomeResource(f *framework.Framework, namespace string) error {
	fmt.Println("Validating custome resources are ready")
	configInstance := &operator.ManagementIngressOperatorConfig{}
	// Get ManagementIngressOperatorSet instance
	err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: config.ConfigCrName, Namespace: namespace}, configInstance)
	if err != nil {
		return err
	}
	lastReason := ""
	waitErr := utilwait.PollImmediate(config.WaitForRetry, config.APITimeout, func() (done bool, err error) {
		for operatorName, operatorState := range configInstance.Status.ServiceStatus {
			for crName, crState := range operatorState.CrStatus {
				if crState == operator.ServiceRunning {
					continue
				} else {
					lastReason = fmt.Sprintf("Waiting on custome resource to be ready" + ", custome resource name: " + crName + " Operator name: " + operatorName)
					return false, nil
				}
			}
		}
		return true, nil
	})
	if waitErr != nil {
		return fmt.Errorf("%v: %s", waitErr, lastReason)
	}
	return nil
}

// AssertNoError confirms the error returned is nil
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// ManagementIngressOperator CR
func newManagementIngressOperatorCR(name, namespace string) *operator.ManagementIngressOperatorCatalog {
	return &operator.ManagementIngressOperatorCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: operator.ManagementIngressOperatorCatalogSpec{
			Operators: []operator.Operator{
				{
					Name:            "etcd",
					Namespace:       "etcd-operator",
					SourceName:      "community-operators",
					SourceNamespace: "openshift-marketplace",
					PackageName:     "etcd",
					Channel:         "singlenamespace-alpha",
					TargetNamespaces: []string{
						"etcd-operator",
					},
				},
				{
					Name:            "jenkins",
					Namespace:       "jenkins-operator",
					SourceName:      "community-operators",
					SourceNamespace: "openshift-marketplace",
					PackageName:     "jenkins-operator",
					Channel:         "alpha",
					TargetNamespaces: []string{
						"jenkins-operator",
					},
				},
			},
		},
	}
}
