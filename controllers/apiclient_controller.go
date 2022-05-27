/*
Copyright 2021.

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
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	record "k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	log "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1beta1 "my.domain/platform/gk8soperator/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// APIClientReconciler reconciles a APIClient object
type APIClientReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	APIController
	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=platform.my.domain,resources=apiclients,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=platform.my.domain,resources=apiclients/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=platform.my.domain,resources=apiclients/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the APIClient object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *APIClientReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// get the API Endpoint resource
	var apiClient platformv1beta1.APIClient
	if err := r.Get(ctx, req.NamespacedName, &apiClient); err != nil {
		//log.Error(err, "unable to fetch APIClient")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	apiClientFinalizerName := "apiclient.platform.my.domain/finalizer"

	// examine DeletionTimestamp to determine if object is under deletion
	if apiClient.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !containsString(apiClient.GetFinalizers(), apiClientFinalizerName) {
			controllerutil.AddFinalizer(&apiClient, apiClientFinalizerName)
			if err := r.Update(ctx, &apiClient); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if containsString(apiClient.GetFinalizers(), apiClientFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.DeleteApplication(&apiClient); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&apiClient, apiClientFinalizerName)
			if err := r.Update(ctx, &apiClient); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	if apiClient.Status.ID != "" {
		// Application already exists
		app, err := r.GetApplication(apiClient.Status.ID)
		if err != nil {
			log.V(0).Info("unable to get Application", "error", err)
			r.recorder.Event(&apiClient, v1.EventTypeNormal, "Error", "Unable to get Application")
			return ctrl.Result{}, err
		}
		if apiClient.Status.UpdatedGeneration < apiClient.ObjectMeta.Generation || apiClient.Status.UpdatedAt < app.UpdatedAt {
			log.V(0).Info("updating the app")
			err := r.UpdateApplication(&apiClient)
			if err != nil {
				log.V(0).Info("nable to update Application", "error", err)
				r.recorder.Event(&apiClient, v1.EventTypeNormal, "Error", "Unable to update Application")
				return ctrl.Result{}, err
			}
			r.recorder.Event(&apiClient, v1.EventTypeNormal, "Ok", "Updated Application")
			r.UpdateCRD(&apiClient, ctx)
			r.UpdateAPISubscriptions(&apiClient, ctx)
		}
	} else {
		app, err := r.CreateApplication(&apiClient)
		if err != nil {
			log.V(0).Info("error creating Application", "error", err)
			r.recorder.Event(&apiClient, v1.EventTypeNormal, "Error", "Unable to create Application")
			return ctrl.Result{}, err
		}
		r.recorder.Event(&apiClient, v1.EventTypeNormal, "Ok", "Created Application")

		apiClient.Status.ID = app.ID
		r.UpdateCRD(&apiClient, ctx)
		r.UpdateAPISubscriptions(&apiClient, ctx)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *APIClientReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Init()
	r.recorder = mgr.GetEventRecorderFor("APIEndpoint")
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1beta1.APIClient{}).
		Complete(r)
}

func (r *APIClientReconciler) UpdateCRD(apiClient *platformv1beta1.APIClient, ctx context.Context) error {
	apiClient.Status.UpdatedGeneration = apiClient.ObjectMeta.Generation
	err := r.Status().Update(ctx, apiClient)
	return err
}
