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
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	types "k8s.io/apimachinery/pkg/types"
	record "k8s.io/client-go/tools/record"
	l "log"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"

	platformv1beta1 "my.domain/platform/gk8soperator/api/v1beta1"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// APIEndpointReconciler reconciles a APIEndpoint object
type APIEndpointReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	APIController
	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=platform.my.domain,resources=apigateways,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=platform.my.domain,resources=apigateways/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=platform.my.domain,resources=apigateways/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the APIEndpoint object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *APIEndpointReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// get the API Endpoint resource
	var apiEndpoint platformv1beta1.APIEndpoint
	if err := r.Get(ctx, req.NamespacedName, &apiEndpoint); err != nil {
		//log.Error(err, "unable to fetch APIEndpoint")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	apiEndpointFinalizerName := "apiendpoint.platform.my.domain/finalizer"

	// examine DeletionTimestamp to determine if object is under deletion
	if apiEndpoint.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !containsString(apiEndpoint.GetFinalizers(), apiEndpointFinalizerName) {
			controllerutil.AddFinalizer(&apiEndpoint, apiEndpointFinalizerName)
			if err := r.Update(ctx, &apiEndpoint); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if containsString(apiEndpoint.GetFinalizers(), apiEndpointFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.DeleteAPI(&apiEndpoint); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}
			log.V(0).Info("api deleted", "ID", apiEndpoint.Status.ID)
			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&apiEndpoint, apiEndpointFinalizerName)
			if err := r.Update(ctx, &apiEndpoint); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	if apiEndpoint.Status.ID != "" {
		log.V(0).Info("api already exists", "ID", apiEndpoint.Status.ID)
		api, err := r.GetAPI(apiEndpoint.Status.ID)
		if err != nil {
			log.V(0).Info("api not configured", "ID", apiEndpoint.Status.ID)
			apiEndpoint.Status.ID = ""
			apiEndpoint.Status.UpdatedAt = 0

			r.recorder.Event(&apiEndpoint, v1.EventTypeNormal, "Ok", "API not found")

			err = r.UpdateCRD(&apiEndpoint, ctx)
			if err != nil {
				log.V(0).Info("error update CRD", "error", err)
			}
			return ctrl.Result{}, err
		}
		if apiEndpoint.Status.UpdatedGeneration < apiEndpoint.ObjectMeta.Generation || apiEndpoint.Status.UpdatedAt < api.UpdatedAt {
			log.V(0).Info("updating the api")

			target, err := r.GetAPITarget(&apiEndpoint, ctx)
			if err != nil {
				log.V(0).Info("error getting target for API", "error", err)
				r.recorder.Event(&apiEndpoint, v1.EventTypeNormal, "Error", "Error getting target for API")
				return ctrl.Result{}, err
			}
			if err = r.UpdateAPI(&apiEndpoint, target, ctx); err != nil {
				log.V(0).Info("error updating API", "error", err)
				r.recorder.Event(&apiEndpoint, v1.EventTypeNormal, "Error", "Error updating API")
				return ctrl.Result{}, err
			}
			r.recorder.Event(&apiEndpoint, v1.EventTypeNormal, "Ok", "Updated API")
			err = r.UpdateAPIPlans(&apiEndpoint)
			if err != nil {
				log.V(0).Info("error update plans", "error", err)
				r.recorder.Event(&apiEndpoint, v1.EventTypeNormal, "Error", "Error update API plans")
				return ctrl.Result{}, err
			}
			r.recorder.Event(&apiEndpoint, v1.EventTypeNormal, "Ok", "Updated API plans")

			if err = r.DeployAPI(api.ID); err != nil {
				log.V(0).Info("error deploying API", "error", err)
				r.recorder.Event(&apiEndpoint, v1.EventTypeNormal, "Error", "Error deploying API")
				return ctrl.Result{}, err
			}
			r.recorder.Event(&apiEndpoint, v1.EventTypeNormal, "Ok", "Deployed API")

			api, err := r.GetAPI(apiEndpoint.Status.ID)
			if err != nil {
				log.V(0).Info("error getting API", "error", err)
				r.recorder.Event(&apiEndpoint, v1.EventTypeNormal, "Error", "Error getting API")
				return ctrl.Result{}, err
			}
			apiEndpoint.Status.ID = api.ID
			apiEndpoint.Status.UpdatedAt = api.UpdatedAt
			apiEndpoint.Status.UpdatedGeneration = apiEndpoint.ObjectMeta.Generation

			err = r.UpdateCRD(&apiEndpoint, ctx)
			if err != nil {
				log.V(0).Info("error update CRD", "error", err)
				return ctrl.Result{}, err
			}

			log.V(0).Info("api crd updated")
			log.V(0).Info("api updated")
		}
	} else {
		log.V(0).Info("api not configured, creating it")
		api, err := r.CreateAPI(&apiEndpoint)
		if err != nil {
			log.V(0).Info("error creating API", "error", err)
			r.recorder.Event(&apiEndpoint, v1.EventTypeNormal, "Error", "Error getting API")
			return ctrl.Result{}, err
		}

		r.recorder.Event(&apiEndpoint, v1.EventTypeNormal, "Ok", "Create API")
		log.V(0).Info("updating the api after creation")

		target, err := r.GetAPITarget(&apiEndpoint, ctx)
		if err != nil {
			log.V(0).Info("error getting target for API", "error", err)
			r.recorder.Event(&apiEndpoint, v1.EventTypeNormal, "Error", "Error getting target for API")
			return ctrl.Result{}, err
		}
		apiEndpoint.Status.ID = api.Payload.ID

		if err = r.UpdateAPI(&apiEndpoint, target, ctx); err != nil {
			log.V(0).Info("error updating API", "error", err)
			r.recorder.Event(&apiEndpoint, v1.EventTypeNormal, "Error", "Error updating API")
			return ctrl.Result{}, err
		}
		r.recorder.Event(&apiEndpoint, v1.EventTypeNormal, "Ok", "Update API")

		err = r.UpdateAPIPlans(&apiEndpoint)
		if err != nil {
			log.V(0).Info("error update plans", "error", err)
			r.recorder.Event(&apiEndpoint, v1.EventTypeNormal, "Error", "Error updating API Plans")
			return ctrl.Result{}, err
		}
		r.recorder.Event(&apiEndpoint, v1.EventTypeNormal, "Ok", "Update API plans")

		if err = r.DeployAPI(apiEndpoint.Status.ID); err != nil {
			log.V(0).Info("error deploying API", "error", err)
			r.recorder.Event(&apiEndpoint, v1.EventTypeNormal, "Error", "Error deploying API")
			return ctrl.Result{}, err
		}
		r.recorder.Event(&apiEndpoint, v1.EventTypeNormal, "Ok", "Deploy API")

		api_updated, err := r.GetAPI(apiEndpoint.Status.ID)
		if err != nil {
			log.V(0).Info("error getting API", "error", err)
			r.recorder.Event(&apiEndpoint, v1.EventTypeNormal, "Error", "Error getting API")
			return ctrl.Result{}, err
		}

		apiEndpoint.Status.UpdatedAt = api_updated.UpdatedAt
		apiEndpoint.Status.UpdatedGeneration = apiEndpoint.ObjectMeta.Generation
		err = r.UpdateCRD(&apiEndpoint, ctx)
		if err != nil {
			log.V(0).Info("error update CRD", "error", err)
			return ctrl.Result{}, err
		}
		log.V(0).Info("api crd updated")
		log.V(0).Info("api created")
	}
	scheduledResult := ctrl.Result{RequeueAfter: time.Duration(r.config["reschedule_period"].(int)) * time.Second}
	return scheduledResult, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *APIEndpointReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Init()
	r.recorder = mgr.GetEventRecorderFor("APIEndpoint")
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1beta1.APIEndpoint{}).
		Complete(r)
}

func (r *APIEndpointReconciler) GetServiceByName(apiEndpoint *platformv1beta1.APIEndpoint, ctx context.Context) (*string, error) {
	service := v1.Service{}
	namespacedName := types.NamespacedName{
		Name:      apiEndpoint.Spec.TargetService,
		Namespace: apiEndpoint.ObjectMeta.Namespace,
	}
	if err := r.Get(ctx, namespacedName, &service); err != nil {
		l.Printf("unable to retrieve Service %s", err)
		return nil, err
	}
	var protocol string
	if service.Spec.Ports[0].AppProtocol != nil {
		protocol = *service.Spec.Ports[0].AppProtocol
	} else {
		protocol = r.config["service_default_protocol"].(string)
	}
	target := fmt.Sprintf("%s://%s.%s.svc.%s:%d/%s", protocol, namespacedName.Name, namespacedName.Namespace, r.config["service_default_domain"].(string), service.Spec.Ports[0].Port, apiEndpoint.Spec.Target)
	return &target, nil
}

func (r *APIEndpointReconciler) UpdateCRD(apiEndpoint *platformv1beta1.APIEndpoint, ctx context.Context) error {
	apiEndpoint.Status.UpdatedGeneration = apiEndpoint.ObjectMeta.Generation
	err := r.Status().Update(ctx, apiEndpoint)
	return err
}

func (r *APIEndpointReconciler) GetAPITarget(apiEndpoint *platformv1beta1.APIEndpoint, ctx context.Context) (*string, error) {
	var target *string
	var err error
	if apiEndpoint.Spec.TargetService != "" {
		target, err = r.GetServiceByName(apiEndpoint, ctx)
	} else {
		target = &apiEndpoint.Spec.Target
	}
	return target, err
}
