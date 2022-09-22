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
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KeyValueDataConditionType string

// These are valid conditions of KeyValueData.
const (
	// KeyValueDataAllow means KeyValueData was added
	KeyValueDataAdded KeyValueDataConditionType = "Added"
)

// KeyValueDataCondition indicates current condition of a deployed resource
type KeyValueDataCondition struct {
	// Type of ClientlessAccessGateway condition.
	Type KeyValueDataConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status core.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

// KeyValueDataConditions is a list of conditions inside of status object
type KeyValueDataConditions []KeyValueDataCondition

// KeyValueDataSpec defines the desired state of KeyValueData
type KeyValueDataSpec struct {
	Data map[string]string `json:"data"`
}

// KeyValueDataStatus defines the observed state of KeyValueData
type KeyValueDataStatus struct {
	KeysInStorage []string               `json:"keysInStorage"`
	Conditions    KeyValueDataConditions `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KeyValueData is the Schema for the keyvaluedata API
type KeyValueData struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeyValueDataSpec   `json:"spec,omitempty"`
	Status KeyValueDataStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KeyValueDataList contains a list of KeyValueData
type KeyValueDataList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeyValueData `json:"items"`
}

func (cs *KeyValueDataStatus) HasCondition(conditionType KeyValueDataConditionType) (bool, int) {
	for index, condition := range cs.Conditions {
		if condition.Type == conditionType {
			return true, index
		}
	}
	return false, -1
}

func (cs *KeyValueDataStatus) AddCondition(condition KeyValueDataCondition) {
	has, index := cs.HasCondition(condition.Type)
	if has {
		cs.Conditions[index] = condition
	} else {
		cs.Conditions = append(cs.Conditions, condition)
	}
}

func init() {
	SchemeBuilder.Register(&KeyValueData{}, &KeyValueDataList{})
}
