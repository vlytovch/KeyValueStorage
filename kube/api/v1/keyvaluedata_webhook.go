/*
Copyright 2022 vlytovch.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var keyvaluedatalog = logf.Log.WithName("keyvaluedata-resource")
var k8sClient client.Client

func (r *KeyValueData) SetupWebhookWithManager(mgr ctrl.Manager) error {
	k8sClient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-teamdev-com-v1-keyvaluedata,mutating=false,failurePolicy=fail,sideEffects=None,groups=teamdev.com,resources=keyvaluedata,verbs=create;update,versions=v1,name=vkeyvaluedata.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &KeyValueData{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *KeyValueData) ValidateCreate() error {
	keyvaluedatalog.Info("validate create", "name", r.Name)
	return r.validatePairsUniqueness()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *KeyValueData) ValidateUpdate(runtime.Object) error {
	keyvaluedatalog.Info("validate update", "name", r.Name)
	return r.validatePairsUniqueness()
}

func (r *KeyValueData) validatePairsUniqueness() error {
	var persistedKeyValueResources KeyValueDataList
	if err := k8sClient.List(context.TODO(), &persistedKeyValueResources, client.InNamespace(r.Namespace)); err != nil {
		keyvaluedatalog.Error(err, "error getting the KeyValueData object")
		return fmt.Errorf("error getting KeyValueData objects")
	}
	var modifiedResourceKeyValueData = r.Spec.Data
	for keys := range modifiedResourceKeyValueData {
		for _, item := range persistedKeyValueResources.Items {
			isIteratedItemModifiedResource := item.Name == r.Name
			if isIteratedItemModifiedResource {
				continue
			}
			if _, existingItemHasKey := item.Spec.Data[keys]; existingItemHasKey {
				return fmt.Errorf("KeyValueData resource containing %q already exists [%q]", keys, item.Name)
			}
		}
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *KeyValueData) ValidateDelete() error {
	keyvaluedatalog.Info("validate delete", "name", r.Name)
	return nil
}
