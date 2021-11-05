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
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	log "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1beta1 "my.domain/platform/gk8soperator/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	gravitee_apps "my.domain/platform/gk8soperator/pkg/gravitee/client/applications"
	gravitee_models "my.domain/platform/gk8soperator/pkg/gravitee/models"
)

// APIClientReconciler reconciles a APIClient object
type APIClientReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	APIController
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
		getApplicationParams := gravitee_apps.GetApplicationParams{}
		getApplicationParams.Application = apiClient.Status.ID
		getApplicationParams.SetTimeout(time.Second * time.Duration(r.config["timeout"].(int)))
		app, err := r.client_apps.GetApplication(&getApplicationParams, r.authInfo)
		if err != nil {
			log.Error(err, "unable to get Application")
		}
		if apiClient.Status.UpdatedGeneration < apiClient.ObjectMeta.Generation || apiClient.Status.UpdatedAt < app.Payload.UpdatedAt {
			log.V(0).Info("updating the app")
			updateApplicationParams := gravitee_apps.UpdateApplicationParams{}
			updateApplicationParams.Application = apiClient.Status.ID
			updateApplicationParams.Body = &gravitee_models.UpdateApplicationEntity{}
			updateApplicationParams.Body.Name = &apiClient.Spec.Name
			updateApplicationParams.Body.Description = &apiClient.Spec.Description
			updateApplicationParams.Body.Type = apiClient.Spec.Type
			updateApplicationParams.Body.ClientID = apiClient.Spec.ClientID
			updateApplicationParams.Body.Settings = &gravitee_models.ApplicationSettings{
				App: &gravitee_models.SimpleApplicationSettings{
					ClientID: apiClient.Spec.ClientID,
					Type:     apiClient.Spec.Type,
				},
			}
			updateApplicationParams.SetTimeout(time.Second * time.Duration(r.config["timeout"].(int)))
			_, err = r.client_apps.UpdateApplication(&updateApplicationParams, r.authInfo)
			if err != nil {
				log.Error(err, "unable to update Application")
			}
			r.UpdateCRD(&apiClient, ctx)
			r.UpdateAPISubscriptions(&apiClient, ctx)
		}
	} else {
		createApplicationParams := gravitee_apps.CreateApplicationParams{}
		createApplicationParams.Application = &gravitee_models.NewApplicationEntity{}
		createApplicationParams.Application.Name = &apiClient.Spec.Name
		createApplicationParams.Application.Description = &apiClient.Spec.Description
		createApplicationParams.Application.Type = apiClient.Spec.Type
		createApplicationParams.Application.ClientID = apiClient.Spec.ClientID
		createApplicationParams.SetTimeout(time.Second * time.Duration(r.config["timeout"].(int)))
		//json_params, _ := json.Marshal(createApplicationParams)
		//l.Printf("createApplicationParams: %s", json_params)
		app, err := r.client_apps.CreateApplication(&createApplicationParams, r.authInfo)
		if err != nil {
			log.Error(err, "error creating Application")
		}
		apiClient.Status.ID = app.Payload.ID
		r.UpdateCRD(&apiClient, ctx)
		r.UpdateAPISubscriptions(&apiClient, ctx)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *APIClientReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Init()
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1beta1.APIClient{}).
		Complete(r)
}

func (r *APIClientReconciler) UpdateCRD(apiClient *platformv1beta1.APIClient, ctx context.Context) error {
	apiClient.Status.UpdatedGeneration = apiClient.ObjectMeta.Generation
	err := r.Status().Update(ctx, apiClient)
	return err
}
