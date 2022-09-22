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
	"io"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
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

type serverError struct {
	Key     string `json:"key"`
	Message string `json:"message"`
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

	deleteServerErrors, err := r.deleteKeysRemovedFromResource(modifiedResource.Status.KeysInStorage, modifiedResource, host)
	if err != nil {
		return ctrl.Result{}, err
	}
	putServerErrors, err := r.updateAndCreateKeysFromResource(modifiedResource, host)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := r.UpdateStatus(ctx, modifiedResource, append(putServerErrors, deleteServerErrors...), host); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *KeyValueDataReconciler) UpdateStatus(ctx context.Context, modifiedResource teamdevcomv1.KeyValueData, serverErrors []serverError, host string) error {
	var condition teamdevcomv1.KeyValueDataCondition

	var persistedKeys []string
	for key := range modifiedResource.Spec.Data {
		resp, err := httpClient.Get(fmt.Sprintf(pairsByKeyEndpointPattern, host, key))
		if err == nil && resp.StatusCode == 200 {
			persistedKeys = append(persistedKeys, key)
		}
	}

	if len(serverErrors) != 0 {
		errMessages, err := json.Marshal(serverErrors)
		if err != nil {
			return err
		}
		condition = teamdevcomv1.KeyValueDataCondition{
			Type:           teamdevcomv1.KeyValueDataAdded,
			Status:         v1.ConditionFalse,
			LastUpdateTime: metav1.Now(),
			Reason:         "BadServerResponse",
			Message:        string(errMessages),
		}
	} else {
		condition = teamdevcomv1.KeyValueDataCondition{
			Type:           teamdevcomv1.KeyValueDataAdded,
			Status:         v1.ConditionTrue,
			LastUpdateTime: metav1.Now(),
		}
	}
	modifiedResource.Status.AddCondition(condition)
	modifiedResource.Status.KeysInStorage = persistedKeys
	if err := r.Client.Status().Update(ctx, &modifiedResource); err != nil {
		return fmt.Errorf("failed to update object with name %s: %w", modifiedResource.Name, err)
	}
	return nil
}

func (r *KeyValueDataReconciler) updateAndCreateKeysFromResource(modifiedResource teamdevcomv1.KeyValueData, host string) ([]serverError, error) {
	var serverErrors []serverError
	for key, value := range modifiedResource.Spec.Data {
		var body, _ = json.Marshal(pair{Key: key, Value: value})
		var putRequest, _ = http.NewRequest("PUT", fmt.Sprintf(pairsEndpointPattern, host), bytes.NewReader(body))
		putRequest.Header.Set("Content-Type", "application/json; charset=utf-8")
		putResp, err := httpClient.Do(putRequest)
		if err != nil {
			return serverErrors, err
		}
		if putResp.StatusCode != 201 && putResp.StatusCode != 200 {
			var body, _ = io.ReadAll(putResp.Body)
			serverErrors = append(serverErrors, serverError{Key: key, Message: string(body)})
		}
	}
	return serverErrors, nil
}

func (r *KeyValueDataReconciler) deleteKeysRemovedFromResource(persistedKeys []string, modifiedResource teamdevcomv1.KeyValueData, host string) ([]serverError, error) {
	var serverErrors []serverError
	for _, key := range persistedKeys {
		_, exists := modifiedResource.Spec.Data[key]
		if !exists {
			var deleteRequest, _ = http.NewRequest("DELETE", fmt.Sprintf(pairsByKeyEndpointPattern, host, key), nil)
			resp, err := httpClient.Do(deleteRequest)
			if err != nil {
				return serverErrors, err
			}
			if resp.StatusCode != 200 {
				var body, _ = io.ReadAll(resp.Body)
				serverErrors = append(serverErrors, serverError{Key: key, Message: string(body)})
			}
		}
	}
	return serverErrors, nil
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
