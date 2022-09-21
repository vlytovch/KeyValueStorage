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

package controllers

import (
	"bytes"
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/strings/slices"
	teamdevcomv1 "kube/api/v1"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var httpClient = &http.Client{}
var myFinalizerName = "teamdev.com.keyvaluedata/finalizer"
var pairsEndpointPattern = "http://%s/pairs"
var pairsByKeyEndpointPattern = pairsEndpointPattern + "/%s"

// KeyValueDataReconciler reconciles a KeyValueData object
type KeyValueDataReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=teamdev.com,resources=keyvaluedata,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=teamdev.com,resources=keyvaluedata/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=teamdev.com,resources=keyvaluedata/finalizers,verbs=update
//+kubebuilder:rbac:groups=*,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=*,resources=services/status,verbs=get

type pair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (r *KeyValueDataReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var (
		host             string
		port             = "8181"
		modifiedResource teamdevcomv1.KeyValueData
		service          v1.Service
	)
	if serviceGetErr := r.Client.Get(ctx, types.NamespacedName{Name: "storage", Namespace: "default"}, &service); serviceGetErr != nil {
		return ctrl.Result{}, serviceGetErr
	}
	if err := r.Client.Get(ctx, req.NamespacedName, &modifiedResource); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	host = service.Spec.ClusterIP + ":" + port

	isBeingDeleted := modifiedResource.ObjectMeta.DeletionTimestamp.IsZero()
	if isBeingDeleted {
		if result, err := r.registerFinalizer(ctx, modifiedResource); err != nil {
			return result, err
		}
	} else {
		return r.processDeletionAndRemoveFinalizer(ctx, modifiedResource, host)
	}

	r.Client.Get(ctx, req.NamespacedName, &modifiedResource)

	persistedKeys := r.getPersistedKeys(modifiedResource)
	persistedKeys = r.deleteKeysRemovedFromResource(persistedKeys, modifiedResource, host)
	persistedKeys = r.updateAndCreateKeysFromResource(persistedKeys, modifiedResource, host)
	var status = teamdevcomv1.KeyValueDataStatus{
		KeysInStorage: persistedKeys,
	}
	modifiedResource.Status = status
	r.Status().Update(ctx, &modifiedResource)
	return ctrl.Result{}, nil
}

func (r *KeyValueDataReconciler) updateAndCreateKeysFromResource(persistedKeys []string, modifiedResource teamdevcomv1.KeyValueData, host string) []string {
	for key, value := range modifiedResource.Spec.Data {
		var body, _ = json.Marshal(pair{Key: key, Value: value})
		var putRequest, _ = http.NewRequest("PUT", fmt.Sprintf(pairsEndpointPattern, host), bytes.NewReader(body))
		putRequest.Header.Set("Content-Type", "application/json; charset=utf-8")
		putResp, err := httpClient.Do(putRequest)
		if err == nil {
			if putResp.StatusCode == 201 {
				persistedKeys = append(persistedKeys, key)
			}
		}
	}
	return persistedKeys
}

func (r *KeyValueDataReconciler) deleteKeysRemovedFromResource(persistedKeys []string, modifiedResource teamdevcomv1.KeyValueData, host string) []string {
	for _, key := range persistedKeys {
		_, exists := modifiedResource.Spec.Data[key]
		if !exists {
			var deleteRequest, _ = http.NewRequest("DELETE", fmt.Sprintf(pairsByKeyEndpointPattern, host, key), nil)
			deleteResp, err := httpClient.Do(deleteRequest)
			if err == nil && deleteResp.StatusCode == 200 {
				persistedKeys = slices.Filter(nil, persistedKeys, func(s string) bool {
					return s != key
				})
			}
		}
	}
	return persistedKeys
}

func (r *KeyValueDataReconciler) getPersistedKeys(modifiedResource teamdevcomv1.KeyValueData) []string {
	var persistedKeys []string
	if modifiedResource.Status.KeysInStorage != nil {
		persistedKeys = modifiedResource.Status.KeysInStorage
	} else {
		persistedKeys = make([]string, 0)
	}
	return persistedKeys
}

func (r *KeyValueDataReconciler) processDeletionAndRemoveFinalizer(ctx context.Context, modifiedResource teamdevcomv1.KeyValueData, host string) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(&modifiedResource, myFinalizerName) {
		if err := r.DeleteAssociatedPairsFromServer(modifiedResource, host); err != nil {
			return ctrl.Result{}, err
		}
		controllerutil.RemoveFinalizer(&modifiedResource, myFinalizerName)
		if err := r.Update(ctx, &modifiedResource); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *KeyValueDataReconciler) registerFinalizer(ctx context.Context, modifiedResource teamdevcomv1.KeyValueData) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(&modifiedResource, myFinalizerName) {
		controllerutil.AddFinalizer(&modifiedResource, myFinalizerName)
		if err := r.Update(ctx, &modifiedResource); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *KeyValueDataReconciler) DeleteAssociatedPairsFromServer(data teamdevcomv1.KeyValueData, host string) error {
	for _, key := range data.Status.KeysInStorage {
		var deleteRequest, _ = http.NewRequest("DELETE", fmt.Sprintf(pairsByKeyEndpointPattern, host, key), nil)
		_, err := httpClient.Do(deleteRequest)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KeyValueDataReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&teamdevcomv1.KeyValueData{}).WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
