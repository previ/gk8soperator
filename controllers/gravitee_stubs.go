package controllers

import (
	"context"
	"encoding/json"
	"errors"
	l "log"
	"os"
	"time"

	platformv1beta1 "my.domain/platform/gk8soperator/api/v1beta1"
	gravitee_apis "my.domain/platform/gk8soperator/pkg/gravitee/client/a_p_is"
	gravitee_plans "my.domain/platform/gk8soperator/pkg/gravitee/client/api_plans"
	gravitee_subs "my.domain/platform/gk8soperator/pkg/gravitee/client/application_subscriptions"
	gravitee_apps "my.domain/platform/gk8soperator/pkg/gravitee/client/applications"
	gravitee_models "my.domain/platform/gk8soperator/pkg/gravitee/models"
	log "sigs.k8s.io/controller-runtime/pkg/log"

	httpruntime "github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	yaml "gopkg.in/yaml.v2"
)

type APIController struct {
	config       map[interface{}]interface{}
	authInfo     httpruntime.ClientAuthInfoWriter
	client_apps  gravitee_apps.ClientService
	client_apis  gravitee_apis.ClientService
	client_plans gravitee_plans.ClientService
	client_subs  gravitee_subs.ClientService
}

func (c *APIController) Init() error {
	c.config = make(map[interface{}]interface{})
	config_file_name := os.Getenv("GRAVITEE_OPERATOR_CONFIG_FILE")
	config_raw, err := os.ReadFile(config_file_name)
	if err != nil {
		l.Panicf("unable to load configuration %v", err)
	}
	err = yaml.Unmarshal(config_raw, &c.config)
	if err != nil {
		l.Panicf("unable to read configuration %v", err)
	}

	var schemes []string
	schemes = append(schemes, c.config["schemes"].(string))
	transport := httptransport.New(c.config["host"].(string), c.config["path"].(string), schemes)
	c.authInfo = httptransport.BasicAuth(c.config["user"].(string), c.config["password"].(string))
	c.client_apis = gravitee_apis.New(transport, strfmt.Default)
	c.client_apps = gravitee_apps.New(transport, strfmt.Default)
	c.client_plans = gravitee_plans.New(transport, strfmt.Default)
	c.client_subs = gravitee_subs.New(transport, strfmt.Default)
	return nil
}

func (c *APIController) GetAPI(APIID string) (*gravitee_models.APIEntity, error) {
	get1Params := gravitee_apis.Get1Params{}
	get1Params.WithDefaults()
	get1Params.SetAPI(APIID)
	get1Params.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
	api, err := c.client_apis.Get1(
		&get1Params,
		c.authInfo,
	)
	return api.Payload, err
}

func (c *APIController) CreateAPI(apiEndpoint *platformv1beta1.APIEndpoint) error {
	createAPIParams := gravitee_apis.CreateAPIParams{}
	createAPIParams.WithAPI(&gravitee_models.NewAPIEntity{
		ContextPath: &apiEndpoint.Spec.ContextPath,
		Description: &apiEndpoint.Spec.Description,
		Name:        &apiEndpoint.Spec.Name,
		Version:     &apiEndpoint.Spec.Version,
		Endpoint:    &apiEndpoint.Spec.Target,
	})
	createAPIParams.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))

	_, err := c.client_apis.CreateAPI(
		&createAPIParams,
		c.authInfo,
	)
	return err
}

func (c *APIController) SearchAPI(ContextPath string) (*gravitee_models.APIListItem, error) {
	searchAPIsParams := gravitee_apis.SearchAPIsParams{}
	searchAPIsParams.WithDefaults()
	searchAPIsParams.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
	searchAPIsParams.SetQ("path " + ContextPath)
	apis, err := c.client_apis.SearchAPIs(&searchAPIsParams, c.authInfo)
	if err != nil {
		l.Panicf("unable to search APIs %s", err)
		return nil, err
	}
	if len(apis.Payload) == 1 {
		return apis.Payload[0], nil
	}
	return nil, nil
}

func (c *APIController) UpdateAPI(apiEndpoint *platformv1beta1.APIEndpoint, target *string, ctx context.Context) error {
	update6Params := gravitee_apis.Update6Params{}
	update6Params.WithDefaults()
	update6Params.SetPathAPI(apiEndpoint.Status.ID)
	updateAPIEntity := gravitee_models.UpdateAPIEntity{}
	updateAPIEntity.Name = &apiEndpoint.Spec.Name
	updateAPIEntity.Version = &apiEndpoint.Spec.Version
	updateAPIEntity.Description = &apiEndpoint.Spec.Description
	var PRIVATE = string("private")
	updateAPIEntity.Visibility = &PRIVATE
	updateAPIEntity.Proxy = &gravitee_models.Proxy{}
	updateAPIEntity.Proxy.VirtualHosts = make([]*gravitee_models.VirtualHost, 1)
	updateAPIEntity.Proxy.VirtualHosts[0] = &gravitee_models.VirtualHost{}
	updateAPIEntity.Proxy.VirtualHosts[0].Path = apiEndpoint.Spec.ContextPath
	updateAPIEntity.Proxy.Groups = make([]*gravitee_models.EndpointGroup, 1)
	updateAPIEntity.Proxy.Groups[0] = &gravitee_models.EndpointGroup{}
	updateAPIEntity.Proxy.Groups[0].Name = "default-group"
	updateAPIEntity.Proxy.Groups[0].Endpoints = make([]*gravitee_models.Endpoint, 1)
	updateAPIEntity.Proxy.Groups[0].Endpoints[0] = &gravitee_models.Endpoint{}
	updateAPIEntity.Proxy.Groups[0].Endpoints[0].Name = "default"
	updateAPIEntity.Proxy.Groups[0].Endpoints[0].Target = *target
	updateAPIEntity.Proxy.Groups[0].Endpoints[0].Type = "HTTP"
	updateAPIEntity.Proxy.Cors = &gravitee_models.Cors{}
	updateAPIEntity.Proxy.Cors.Enabled = apiEndpoint.Spec.Cors.Enabled
	updateAPIEntity.Proxy.Cors.AllowCredentials = apiEndpoint.Spec.Cors.AllowCredentials
	updateAPIEntity.Proxy.Cors.AllowHeaders = apiEndpoint.Spec.Cors.AllowHeaders
	updateAPIEntity.Proxy.Cors.AllowMethods = apiEndpoint.Spec.Cors.AllowMethods
	updateAPIEntity.Proxy.Cors.AllowOrigin = apiEndpoint.Spec.Cors.AllowOrigin
	updateAPIEntity.Proxy.Cors.MaxAge = apiEndpoint.Spec.Cors.MaxAge
	updateAPIEntity.Proxy.Cors.ErrorStatusCode = apiEndpoint.Spec.Cors.ErrorStatusCode
	updateAPIEntity.Proxy.Cors.RunPolicies = apiEndpoint.Spec.Cors.RunPolicies
	update6Params.SetBodyAPI(&updateAPIEntity)
	update6Params.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
	//json_params, _ := json.Marshal(update6Params)
	//l.Printf("update6Params: %s", json_params)
	_, err := c.client_apis.Update6(
		&update6Params,
		c.authInfo,
	)
	return err
}

func (c *APIController) DeployAPI(apiID string) error {
	deployAPIParams := gravitee_apis.DeployAPIParams{
		API: apiID,
	}
	deployAPIParams.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
	_, err := c.client_apis.DeployAPI(
		&deployAPIParams,
		c.authInfo,
	)
	doLifecycleActionParams := gravitee_apis.DoLifecycleActionParams{
		API:    apiID,
		Action: "START",
	}
	doLifecycleActionParams.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
	_, err = c.client_apis.DoLifecycleAction(
		&doLifecycleActionParams,
		c.authInfo,
	)
	if err != nil {
		l.Printf("unable to DoLifecycleAction err: %s", err)
		err = nil // API was already started, non a real error
	}

	return err
}

func (c *APIController) UpdateAPIPlans(apiEndpoint *platformv1beta1.APIEndpoint) error {
	listPlansParams := gravitee_plans.ListPlansParams{}
	listPlansParams.WithDefaults()
	listPlansParams.API = apiEndpoint.Status.ID
	listPlansParams.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
	plans, err := c.client_plans.ListPlans(&listPlansParams, c.authInfo)
	if err != nil {
		l.Printf("ListPlans err: %s", err)
	}

	// update existing or crete new
	var plan_found bool
	for _, plan_new := range apiEndpoint.Spec.Plans {
		plan_found = false
		for _, plan_ext := range plans.Payload {
			if *plan_new.Name == plan_ext.Name {
				plan_found = true
				updatePlanParams := gravitee_plans.UpdatePlanParams{}
				updatePlanParams.WithDefaults()
				updatePlanParams.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
				updatePlanParams.SetAPI(apiEndpoint.Status.ID)
				updatePlanParams.SetPathPlan(plan_ext.ID)
				updatePlanParams.BodyPlan = &gravitee_models.UpdatePlanEntity{}
				updatePlanParams.BodyPlan.Description = &plan_new.Description
				updatePlanParams.BodyPlan.Name = plan_new.Name
				securityDefinition, _ := json.Marshal(plan_new.SecurityDefinition)
				updatePlanParams.BodyPlan.SecurityDefinition = string(securityDefinition)
				updatePlanParams.BodyPlan.Tags = plan_ext.Tags
				updatePlanParams.BodyPlan.Order = &plan_ext.Order
				updatePlanParams.BodyPlan.Validation = &plan_ext.Validation
				_, err := c.client_plans.UpdatePlan(&updatePlanParams, c.authInfo)
				if err != nil {
					l.Panicf("Error updating plan: %s", err)
				}
			}
		}
		if plan_found == false {
			createPlanParams := gravitee_plans.CreatePlanParams{}
			createPlanParams.WithDefaults()
			createPlanParams.SetAPI(apiEndpoint.Status.ID)
			createPlanParams.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
			createPlanParams.Plan = &gravitee_models.NewPlanEntity{}
			createPlanParams.Plan.API = apiEndpoint.Status.ID
			createPlanParams.Plan.Description = &plan_new.Description
			createPlanParams.Plan.Name = plan_new.Name
			createPlanParams.Plan.Security = plan_new.Security
			securityDefinition, _ := json.Marshal(plan_new.SecurityDefinition)
			createPlanParams.Plan.SecurityDefinition = string(securityDefinition)
			status := "PUBLISHED"
			createPlanParams.Plan.Status = &status
			typ := "API"
			createPlanParams.Plan.Type = &typ
			auto := "AUTO"
			createPlanParams.Plan.Validation = &auto
			_, err := c.client_plans.CreatePlan(&createPlanParams, c.authInfo)
			if err != nil {
				l.Panicf("Error creating plan: %s", err)
			}
		}
	}
	// delete retired plans
	for _, plan_ext := range plans.Payload {
		plan_found = false
		for _, plan_new := range apiEndpoint.Spec.Plans {
			if *plan_new.Name == plan_ext.Name {
				plan_found = true
			}
		}
		if plan_found == false {
			closePlanParams := gravitee_plans.ClosePlanParams{}
			closePlanParams.WithDefaults()
			closePlanParams.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
			closePlanParams.SetAPI(plan_ext.Apis[0])
			closePlanParams.SetPlan(plan_ext.ID)
			_, err := c.client_plans.ClosePlan(&closePlanParams, c.authInfo)
			if err != nil {
				l.Panicf("Error closing plan: %s", err)
			}
			deletePlanParams := gravitee_plans.DeletePlanParams{}
			deletePlanParams.WithDefaults()
			deletePlanParams.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
			deletePlanParams.SetAPI(plan_ext.Apis[0])
			deletePlanParams.SetPlan(plan_ext.ID)
			_, err = c.client_plans.DeletePlan(&deletePlanParams, c.authInfo)
			if err != nil {
				l.Panicf("Error closing plan: %s", err)
			}
		}
	}
	return nil
}

func (c *APIController) DeleteAPI(apiEndpoint *platformv1beta1.APIEndpoint) error {
	doLifecycleActionParams := gravitee_apis.DoLifecycleActionParams{
		API:    apiEndpoint.Status.ID,
		Action: "STOP",
	}
	doLifecycleActionParams.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
	_, err := c.client_apis.DoLifecycleAction(
		&doLifecycleActionParams,
		c.authInfo,
	)
	if err != nil {
		l.Printf("unable to DoLifecycleAction err: %s", err)
		err = nil // API was already started, non a real error
	}
	listPlansParams := gravitee_plans.ListPlansParams{}
	listPlansParams.WithDefaults()
	listPlansParams.API = apiEndpoint.Status.ID
	listPlansParams.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
	plans, err := c.client_plans.ListPlans(&listPlansParams, c.authInfo)
	if err != nil {
		l.Printf("ListPlans err: %s", err)
	}
	for _, plan := range plans.Payload {
		closePlanParams := gravitee_plans.ClosePlanParams{}
		closePlanParams.WithDefaults()
		closePlanParams.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
		closePlanParams.SetAPI(apiEndpoint.Status.ID)
		closePlanParams.SetPlan(plan.ID)
		_, err := c.client_plans.ClosePlan(&closePlanParams, c.authInfo)
		if err != nil {
			l.Panicf("Error closing plan: %s", err)
		}
		deletePlanParams := gravitee_plans.DeletePlanParams{}
		deletePlanParams.WithDefaults()
		deletePlanParams.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
		deletePlanParams.SetAPI(apiEndpoint.Status.ID)
		deletePlanParams.SetPlan(plan.ID)
		_, err = c.client_plans.DeletePlan(&deletePlanParams, c.authInfo)
		if err != nil {
			l.Panicf("Error deleting plan: %s", err)
		}
	}
	delete3Params := gravitee_apis.Delete3Params{}
	delete3Params.API = apiEndpoint.Status.ID
	delete3Params.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
	_, err = c.client_apis.Delete3(&delete3Params, c.authInfo)
	if err != nil {
		l.Printf("error Delete3 API: %v", err)
		return err
	}
	return nil
}

func (c *APIController) UpdateAPISubscriptions(apiClient *platformv1beta1.APIClient, ctx context.Context) error {
	log := log.FromContext(ctx)
	// list existing subscriptions and create a map <context_path-plan>:<subsciption>
	ListApplicationSubscriptionsParams := gravitee_subs.ListApplicationSubscriptionsParams{}
	ListApplicationSubscriptionsParams.Application = apiClient.Status.ID
	ListApplicationSubscriptionsParams.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
	subs, err := c.client_subs.ListApplicationSubscriptions(&ListApplicationSubscriptionsParams, c.authInfo)
	if err != nil {
		l.Panicf("unable to ListApplicationSubscriptions %s", err)
		return err
	}
	subs_ext := make(map[string]interface{})
	for _, sub_ext := range subs.Payload.Data {
		sub_ext_map := sub_ext.(map[string]interface{})
		get1Params := &gravitee_apis.Get1Params{
			API: sub_ext_map["api"].(string),
		}
		get1Params.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
		api, err := c.client_apis.Get1(get1Params, c.authInfo)
		if err != nil {
			l.Panicf("unable to get API %s", err)
			return err
		}
		plan, err := c.GetPlan(sub_ext_map["api"].(string), sub_ext_map["plan"].(string))
		if err != nil {
			l.Panicf("unable to get Plan %s", err)
			return err
		}
		subs_ext[api.Payload.ContextPath+"-"+plan.Name] = plan
	}
	// check for new subscriptions and create them
	for _, sub_new := range apiClient.Spec.APISubscriptions {
		if _, ok := subs_ext[sub_new.APIContextPath+"-"+sub_new.APIPlanName]; ok == false {
			api_list_item, err := c.SearchAPI(sub_new.APIContextPath)
			if err != nil {
				l.Printf("unable to get API by ContextPath %s", err)
				return err
			}
			plan, err := c.GetPlanByName(api_list_item.ID, sub_new.APIPlanName)
			if err != nil {
				l.Printf("unable to get Plan by name %s", err)
				return err
			}
			createSubscription1Params := gravitee_subs.CreateSubscription1Params{
				Application: apiClient.Status.ID,
				Plan:        plan.ID,
			}
			createSubscription1Params.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
			c.client_subs.CreateSubscription1(&createSubscription1Params, c.authInfo)
			log.V(0).Info("creating subscription", "subscription", sub_new.APIContextPath+"-"+sub_new.APIPlanName)
		}
	}
	// check for obsolete subscriptions and close them
	var sub_found bool
	for sub_key, sub_ext := range subs_ext {
		sub_found = false
		for _, sub_new := range apiClient.Spec.APISubscriptions {
			if sub_key == sub_new.APIContextPath+"-"+sub_new.APIPlanName {
				sub_found = true
			}
		}
		if sub_found == false {
			var sub_id string
			for _, sub := range subs.Payload.Data {
				sub_map := sub.(map[string]interface{})
				if sub_map["plan"].(string) == sub_ext.(*gravitee_models.PlanEntity).ID {
					sub_id = sub_map["id"].(string)
				}

			}
			closeSubscriptionParams := gravitee_subs.CloseSubscriptionParams{
				Application:  apiClient.Status.ID,
				Subscription: sub_id,
			}
			//l.Printf("closeSubscriptionParams %v", closeSubscriptionParams)
			closeSubscriptionParams.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
			_, err = c.client_subs.CloseSubscription(&closeSubscriptionParams, c.authInfo)
			if err != nil {
				l.Printf("unable to close Plan %s", err)
				return err
			}
			log.V(0).Info("closing subscription", "subscription", sub_key)
		}
	}
	return err
}

func (c *APIController) GetPlan(APIID string, PlanID string) (*gravitee_models.PlanEntity, error) {
	getPlanParams := gravitee_plans.GetPlanParams{
		API:  APIID,
		Plan: PlanID,
	}
	getPlanParams.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
	plan, err := c.client_plans.GetPlan(&getPlanParams, c.authInfo)
	return plan.Payload, err
}

func (c *APIController) GetPlanByName(APIID string, PlanName string) (*gravitee_models.PlanEntity, error) {
	listPlanParams := gravitee_plans.ListPlansParams{
		API: APIID,
	}
	listPlanParams.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
	plans, err := c.client_plans.ListPlans(&listPlanParams, c.authInfo)
	if err != nil {
		l.Printf("unable to search Plans %s", err)
		return nil, err
	}
	for _, plan := range plans.Payload {
		if plan.Name == PlanName {
			return plan, nil
		}
	}
	return nil, errors.New("unable to find exact one plan")
}

func (c *APIController) DeleteApplication(apiClient *platformv1beta1.APIClient) error {
	ListApplicationSubscriptionsParams := gravitee_subs.ListApplicationSubscriptionsParams{}
	ListApplicationSubscriptionsParams.Application = apiClient.Status.ID
	ListApplicationSubscriptionsParams.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
	subs, err := c.client_subs.ListApplicationSubscriptions(&ListApplicationSubscriptionsParams, c.authInfo)
	if err != nil {
		l.Panicf("unable to ListApplicationSubscriptions %s", err)
		return err
	}
	for _, sub_ext := range subs.Payload.Data {
		sub_ext_map := sub_ext.(map[string]interface{})
		closeSubscriptionParams := gravitee_subs.CloseSubscriptionParams{
			Application:  apiClient.Status.ID,
			Subscription: sub_ext_map["id"].(string),
		}
		closeSubscriptionParams.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
		_, err = c.client_subs.CloseSubscription(&closeSubscriptionParams, c.authInfo)
		if err != nil {
			l.Printf("unable to close Plan %s", err)
			return err
		}
	}
	deleteApplicationParams := gravitee_apps.DeleteApplicationParams{
		Application: apiClient.Status.ID,
	}
	deleteApplicationParams.SetTimeout(time.Second * time.Duration(c.config["timeout"].(int)))
	_, err = c.client_apps.DeleteApplication(&deleteApplicationParams, c.authInfo)
	if err != nil {
		l.Printf("unable to delete Application %s", err)
		return err
	}
	return nil
}

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
