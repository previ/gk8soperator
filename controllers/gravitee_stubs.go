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
	gravitee_analytics "my.domain/platform/gk8soperator/pkg/gravitee/client/api_analytics"
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
	config           map[interface{}]interface{}
	authInfo         httpruntime.ClientAuthInfoWriter
	client_apps      gravitee_apps.ClientService
	client_apis      gravitee_apis.ClientService
	client_plans     gravitee_plans.ClientService
	client_subs      gravitee_subs.ClientService
	client_analytics gravitee_analytics.ClientService
	Timeout          int
	OrgID            string
	EnvID            string
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
	if c.config["user"] != nil && len(c.config["user"].(string)) > 0 && c.config["password"] != nil && len(c.config["password"].(string)) > 0 {
		c.authInfo = httptransport.BasicAuth(c.config["user"].(string), c.config["password"].(string))
	} else if c.config["token"] != nil && len(c.config["token"].(string)) > 0 {
		c.authInfo = httptransport.BearerToken(c.config["token"].(string))
	}
	c.Timeout = c.config["timeout"].(int)
	c.OrgID = c.config["organization"].(string)
	c.EnvID = c.config["environment"].(string)
	c.client_apis = gravitee_apis.New(transport, strfmt.Default)
	c.client_apps = gravitee_apps.New(transport, strfmt.Default)
	c.client_plans = gravitee_plans.New(transport, strfmt.Default)
	c.client_subs = gravitee_subs.New(transport, strfmt.Default)
	c.client_analytics = gravitee_analytics.New(transport, strfmt.Default)
	return nil
}

func (c *APIController) GetAPI(APIID string) (*gravitee_models.APIEntity, error) {
	getAPIParams := gravitee_apis.GetAPIParams{}
	getAPIParams.WithDefaults()
	getAPIParams.SetAPI(APIID)
	getAPIParams.SetOrgID(c.OrgID)
	getAPIParams.SetEnvID(c.EnvID)
	getAPIParams.SetTimeout(time.Second * time.Duration(c.Timeout))
	api, err := c.client_apis.GetAPI(
		&getAPIParams,
		c.authInfo,
	)
	if err != nil {
		l.Printf("unable to get API %s", err)
		return nil, err
	}
	return api.Payload, err
}

func (c *APIController) CreateAPI(apiEndpoint *platformv1beta1.APIEndpoint) (*gravitee_apis.CreateAPICreated, error) {
	createAPIParams := gravitee_apis.CreateAPIParams{}
	createAPIParams.WithAPI(&gravitee_models.NewAPIEntity{
		ContextPath: &apiEndpoint.Spec.ContextPath,
		Description: &apiEndpoint.Spec.Description,
		Name:        &apiEndpoint.Spec.Name,
		Version:     &apiEndpoint.Spec.Version,
		Endpoint:    &apiEndpoint.Spec.Target,
	})
	createAPIParams.SetTimeout(time.Second * time.Duration(c.Timeout))
	createAPIParams.SetOrgID(c.OrgID)
	createAPIParams.SetEnvID(c.EnvID)
	// json_params, _ := json.Marshal(createAPIParams)
	// l.Printf("createAPIParams: %s", json_params)

	api, err := c.client_apis.CreateAPI(
		&createAPIParams,
		c.authInfo,
	)
	if err != nil {
		l.Printf("unable to create API %s", err)
		return nil, err
	}

	return api, err
}

func (c *APIController) SearchAPI(ContextPath string) (*gravitee_models.APIListItem, error) {
	searchApisParams := gravitee_apis.SearchApisParams{}
	searchApisParams.WithDefaults()
	searchApisParams.SetTimeout(time.Second * time.Duration(c.Timeout))
	searchApisParams.SetQ("virtual_hosts.path " + ContextPath)
	searchApisParams.SetOrgID(c.OrgID)
	searchApisParams.SetEnvID(c.EnvID)
	// json_params, _ := json.Marshal(searchApisParams)
	// l.Printf("SearchApisParams: %s", json_params)
	apis, err := c.client_apis.SearchApis(&searchApisParams, c.authInfo)
	if err != nil {
		l.Panicf("unable to search APIs %s", err)
		return nil, err
	}
	l.Printf("searchAPIsResults: %v", apis.Payload)
	if len(apis.Payload) == 1 {
		return apis.Payload[0], nil
	} else {
		return nil, nil
	}
}

func (c *APIController) UpdateAPI(apiEndpoint *platformv1beta1.APIEndpoint, target *string, ctx context.Context) error {
	updateAPIParams := gravitee_apis.UpdateAPIParams{}
	updateAPIParams.WithDefaults()
	updateAPIParams.SetPathAPI(apiEndpoint.Status.ID)
	updateAPIEntity := gravitee_models.UpdateAPIEntity{}
	updateAPIEntity.Name = &apiEndpoint.Spec.Name
	updateAPIEntity.Version = &apiEndpoint.Spec.Version
	updateAPIEntity.Description = &apiEndpoint.Spec.Description
	updateAPIEntity.Categories = make([]string, 0)
	updateAPIEntity.Flows = make([]*gravitee_models.Flow, 0)
	updateAPIEntity.Groups = make([]string, 0)
	updateAPIEntity.Labels = make([]string, 0)
	updateAPIEntity.Metadata = make([]*gravitee_models.APIMetadataEntity, 0)
	updateAPIEntity.PathMappings = make([]string, 0)
	updateAPIEntity.Properties = make([]*gravitee_models.PropertyEntity, 0)
	updateAPIEntity.Resources = make([]*gravitee_models.Resource, 0)
	var PRIVATE = string("private")
	updateAPIEntity.Visibility = &PRIVATE
	updateAPIEntity.Tags = apiEndpoint.Spec.Tags
	updateAPIEntity.Proxy = &gravitee_models.Proxy{}
	updateAPIEntity.Proxy.VirtualHosts = make([]*gravitee_models.VirtualHost, 1)
	updateAPIEntity.Proxy.VirtualHosts[0] = &gravitee_models.VirtualHost{}
	updateAPIEntity.Proxy.VirtualHosts[0].Path = apiEndpoint.Spec.ContextPath
	updateAPIEntity.Proxy.Groups = make([]*gravitee_models.EndpointGroup, 1)
	updateAPIEntity.Proxy.Groups[0] = &gravitee_models.EndpointGroup{}
	updateAPIEntity.Proxy.Groups[0].Name = "default-group"
	updateAPIEntity.Proxy.Groups[0].Headers = make([]*gravitee_models.HTTPHeader, 0)
	updateAPIEntity.Proxy.Groups[0].Endpoints = make([]*gravitee_models.Endpoint, 1)
	updateAPIEntity.Proxy.Groups[0].Endpoints[0] = &gravitee_models.Endpoint{}
	updateAPIEntity.Proxy.Groups[0].Endpoints[0].Name = "default"
	updateAPIEntity.Proxy.Groups[0].Endpoints[0].Target = *target
	updateAPIEntity.Proxy.Groups[0].Endpoints[0].Type = "http"
	updateAPIEntity.Proxy.Groups[0].Endpoints[0].Tenants = make([]string, 0)
	updateAPIEntity.Proxy.Cors = &gravitee_models.Cors{}
	updateAPIEntity.Proxy.Cors.Enabled = apiEndpoint.Spec.Cors.Enabled
	updateAPIEntity.Proxy.Cors.AllowCredentials = apiEndpoint.Spec.Cors.AllowCredentials
	updateAPIEntity.Proxy.Cors.AllowHeaders = apiEndpoint.Spec.Cors.AllowHeaders
	updateAPIEntity.Proxy.Cors.AllowMethods = apiEndpoint.Spec.Cors.AllowMethods
	updateAPIEntity.Proxy.Cors.AllowOrigin = apiEndpoint.Spec.Cors.AllowOrigin
	updateAPIEntity.Proxy.Cors.MaxAge = apiEndpoint.Spec.Cors.MaxAge
	//updateAPIEntity.Proxy.Cors.ErrorStatusCode = apiEndpoint.Spec.Cors.ErrorStatusCode
	updateAPIEntity.Proxy.Cors.RunPolicies = apiEndpoint.Spec.Cors.RunPolicies
	updateAPIParams.SetBodyAPI(&updateAPIEntity)
	updateAPIParams.SetTimeout(time.Second * time.Duration(c.Timeout))
	updateAPIParams.SetOrgID(c.OrgID)
	updateAPIParams.SetEnvID(c.EnvID)
	_, err := c.client_apis.UpdateAPI(
		&updateAPIParams,
		c.authInfo,
	)
	if err != nil {
		json_params, _ := json.Marshal(updateAPIParams)
		l.Printf("updateAPIParams: %s", json_params)
		l.Printf("error UpdateAPI: %s", err)
	}
	return err
}

func (c *APIController) DeployAPI(apiID string) error {
	deployAPIParams := gravitee_apis.DeployAPIParams{
		API: apiID,
	}
	deployAPIParams.SetTimeout(time.Second * time.Duration(c.Timeout))
	deployAPIParams.SetOrgID(c.OrgID)
	deployAPIParams.SetEnvID(c.EnvID)
	_, err := c.client_apis.DeployAPI(
		&deployAPIParams,
		c.authInfo,
	)
	doAPILifecycleActionParams := gravitee_apis.DoAPILifecycleActionParams{
		API:    apiID,
		Action: "START",
	}
	doAPILifecycleActionParams.SetTimeout(time.Second * time.Duration(c.Timeout))
	doAPILifecycleActionParams.SetOrgID(c.OrgID)
	doAPILifecycleActionParams.SetEnvID(c.EnvID)
	_, err = c.client_apis.DoAPILifecycleAction(
		&doAPILifecycleActionParams,
		c.authInfo,
	)
	if err != nil {
		l.Printf("unable to DoLifecycleAction err: %s", err)
		err = nil // API was already started, non a real error
	}

	return err
}

func (c *APIController) UpdateAPIPlans(apiEndpoint *platformv1beta1.APIEndpoint) error {
	getAPIPlansParams := gravitee_plans.GetAPIPlansParams{}
	getAPIPlansParams.WithDefaults()
	getAPIPlansParams.API = apiEndpoint.Status.ID
	getAPIPlansParams.SetTimeout(time.Second * time.Duration(c.Timeout))
	getAPIPlansParams.SetOrgID(c.OrgID)
	getAPIPlansParams.SetEnvID(c.EnvID)
	plans, err := c.client_plans.GetAPIPlans(&getAPIPlansParams, c.authInfo)
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
				updateAPIPlanParams := gravitee_plans.UpdateAPIPlanParams{}
				updateAPIPlanParams.WithDefaults()
				updateAPIPlanParams.SetTimeout(time.Second * time.Duration(c.Timeout))
				updateAPIPlanParams.SetAPI(apiEndpoint.Status.ID)
				updateAPIPlanParams.SetPathPlan(plan_ext.ID)
				updateAPIPlanParams.BodyPlan = &gravitee_models.UpdatePlanEntity{}
				updateAPIPlanParams.BodyPlan.Description = &plan_new.Description
				updateAPIPlanParams.BodyPlan.Name = plan_new.Name
				securityDefinition, _ := json.Marshal(plan_new.SecurityDefinition)
				updateAPIPlanParams.BodyPlan.SecurityDefinition = string(securityDefinition)
				updateAPIPlanParams.BodyPlan.Tags = plan_ext.Tags
				updateAPIPlanParams.BodyPlan.Order = &plan_ext.Order
				updateAPIPlanParams.BodyPlan.Validation = &plan_ext.Validation
				updateAPIPlanParams.SetOrgID(c.OrgID)
				updateAPIPlanParams.SetEnvID(c.EnvID)
				_, err := c.client_plans.UpdateAPIPlan(&updateAPIPlanParams, c.authInfo)
				if err != nil {
					l.Panicf("Error updating plan: %s", err)
				}
			}
		}
		if plan_found == false {
			createAPIPlanParams := gravitee_plans.CreateAPIPlanParams{}
			createAPIPlanParams.WithDefaults()
			createAPIPlanParams.SetAPI(apiEndpoint.Status.ID)
			createAPIPlanParams.SetTimeout(time.Second * time.Duration(c.Timeout))
			createAPIPlanParams.SetOrgID(c.OrgID)
			createAPIPlanParams.SetEnvID(c.EnvID)
			createAPIPlanParams.Plan = &gravitee_models.NewPlanEntity{}
			createAPIPlanParams.Plan.API = apiEndpoint.Status.ID
			createAPIPlanParams.Plan.Description = &plan_new.Description
			createAPIPlanParams.Plan.Name = plan_new.Name
			createAPIPlanParams.Plan.Security = plan_new.Security
			securityDefinition, _ := json.Marshal(plan_new.SecurityDefinition)
			createAPIPlanParams.Plan.SecurityDefinition = string(securityDefinition)
			status := "PUBLISHED"
			createAPIPlanParams.Plan.Status = &status
			typ := "API"
			createAPIPlanParams.Plan.Type = &typ
			auto := "AUTO"
			createAPIPlanParams.Plan.Validation = &auto
			_, err := c.client_plans.CreateAPIPlan(&createAPIPlanParams, c.authInfo)
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
			closeAPIPlanParams := gravitee_plans.CloseAPIPlanParams{}
			closeAPIPlanParams.WithDefaults()
			closeAPIPlanParams.SetTimeout(time.Second * time.Duration(c.Timeout))
			closeAPIPlanParams.SetOrgID(c.OrgID)
			closeAPIPlanParams.SetEnvID(c.EnvID)
			closeAPIPlanParams.SetAPI(plan_ext.API)
			closeAPIPlanParams.SetPlan(plan_ext.ID)
			_, _, err := c.client_plans.CloseAPIPlan(&closeAPIPlanParams, c.authInfo)
			if err != nil {
				l.Panicf("Error closing plan: %s", err)
			}
			deleteAPIPlanParams := gravitee_plans.DeleteAPIPlanParams{}
			deleteAPIPlanParams.WithDefaults()
			deleteAPIPlanParams.SetTimeout(time.Second * time.Duration(c.Timeout))
			deleteAPIPlanParams.SetOrgID(c.OrgID)
			deleteAPIPlanParams.SetEnvID(c.EnvID)
			deleteAPIPlanParams.SetAPI(plan_ext.API)
			deleteAPIPlanParams.SetPlan(plan_ext.ID)
			_, err = c.client_plans.DeleteAPIPlan(&deleteAPIPlanParams, c.authInfo)
			if err != nil {
				l.Panicf("Error deleting plan: %s", err)
			}
		}
	}
	return nil
}

func (c *APIController) DeleteAPI(apiEndpoint *platformv1beta1.APIEndpoint) error {
	doAPILifecycleActionParams := gravitee_apis.DoAPILifecycleActionParams{
		API:    apiEndpoint.Status.ID,
		Action: "STOP",
	}
	doAPILifecycleActionParams.SetTimeout(time.Second * time.Duration(c.Timeout))
	doAPILifecycleActionParams.SetOrgID(c.OrgID)
	doAPILifecycleActionParams.SetEnvID(c.EnvID)
	_, err := c.client_apis.DoAPILifecycleAction(
		&doAPILifecycleActionParams,
		c.authInfo,
	)
	if err != nil {
		l.Printf("unable to DoLifecycleAction err: %s", err)
		err = nil // API was already started, non a real error
	}
	getAPIPlansParams := gravitee_plans.GetAPIPlansParams{}
	getAPIPlansParams.WithDefaults()
	getAPIPlansParams.API = apiEndpoint.Status.ID
	getAPIPlansParams.SetTimeout(time.Second * time.Duration(c.Timeout))
	getAPIPlansParams.SetOrgID(c.OrgID)
	getAPIPlansParams.SetEnvID(c.EnvID)
	plans, err := c.client_plans.GetAPIPlans(&getAPIPlansParams, c.authInfo)
	if err != nil {
		l.Printf("ListPlans err: %s", err)
		return err
	}
	for _, plan := range plans.Payload {
		closeAPIPlanParams := gravitee_plans.CloseAPIPlanParams{}
		closeAPIPlanParams.WithDefaults()
		closeAPIPlanParams.SetTimeout(time.Second * time.Duration(c.Timeout))
		closeAPIPlanParams.SetOrgID(c.OrgID)
		closeAPIPlanParams.SetEnvID(c.EnvID)
		closeAPIPlanParams.SetAPI(apiEndpoint.Status.ID)
		closeAPIPlanParams.SetPlan(plan.ID)
		_, _, err := c.client_plans.CloseAPIPlan(&closeAPIPlanParams, c.authInfo)
		if err != nil {
			l.Panicf("Error closing plan: %s", err)
			continue
		}
		deleteAPIPlanParams := gravitee_plans.DeleteAPIPlanParams{}
		deleteAPIPlanParams.WithDefaults()
		deleteAPIPlanParams.SetTimeout(time.Second * time.Duration(c.Timeout))
		deleteAPIPlanParams.SetOrgID(c.OrgID)
		deleteAPIPlanParams.SetEnvID(c.EnvID)
		deleteAPIPlanParams.SetAPI(apiEndpoint.Status.ID)
		deleteAPIPlanParams.SetPlan(plan.ID)
		_, err = c.client_plans.DeleteAPIPlan(&deleteAPIPlanParams, c.authInfo)
		if err != nil {
			l.Panicf("Error deleting plan: %s", err)
			continue
		}
	}
	deleteAPIParams := gravitee_apis.DeleteAPIParams{}
	deleteAPIParams.API = apiEndpoint.Status.ID
	deleteAPIParams.SetTimeout(time.Second * time.Duration(c.Timeout))
	deleteAPIParams.SetOrgID(c.OrgID)
	deleteAPIParams.SetEnvID(c.EnvID)
	_, err = c.client_apis.DeleteAPI(&deleteAPIParams, c.authInfo)
	if err != nil {
		l.Printf("error DeleteAPI API: %v", err)
		return err
	}
	return nil
}

func (c *APIController) UpdateAPISubscriptions(apiClient *platformv1beta1.APIClient, ctx context.Context) error {
	log := log.FromContext(ctx)
	// list existing subscriptions and create a map <context_path-plan>:<subsciption>
	getApplicationSubscriptionsParams := gravitee_subs.GetApplicationSubscriptionsParams{}
	getApplicationSubscriptionsParams.Application = apiClient.Status.ID
	getApplicationSubscriptionsParams.SetTimeout(time.Second * time.Duration(c.Timeout))
	getApplicationSubscriptionsParams.SetOrgID(c.OrgID)
	getApplicationSubscriptionsParams.SetEnvID(c.EnvID)
	subs, err := c.client_subs.GetApplicationSubscriptions(&getApplicationSubscriptionsParams, c.authInfo)
	if err != nil {
		l.Panicf("unable to GetApplicationSubscriptions %s", err)
		return err
	}

	subs_ext := make(map[string]interface{})
	for _, sub_ext := range subs.Payload.Data {
		sub_ext_map := sub_ext.(map[string]interface{})
		getAPIParams := &gravitee_apis.GetAPIParams{
			API: sub_ext_map["api"].(string),
		}
		getAPIParams.SetTimeout(time.Second * time.Duration(c.Timeout))
		getAPIParams.SetOrgID(c.OrgID)
		getAPIParams.SetEnvID(c.EnvID)
		api, err := c.client_apis.GetAPI(getAPIParams, c.authInfo)
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
			createSubscriptionWithApplicationParams := gravitee_subs.CreateSubscriptionWithApplicationParams{
				Application: apiClient.Status.ID,
				Plan:        plan.ID,
			}
			createSubscriptionWithApplicationParams.SetTimeout(time.Second * time.Duration(c.Timeout))
			createSubscriptionWithApplicationParams.SetOrgID(c.OrgID)
			createSubscriptionWithApplicationParams.SetEnvID(c.EnvID)
			c.client_subs.CreateSubscriptionWithApplication(&createSubscriptionWithApplicationParams, c.authInfo)
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
			closeApplicationSubscriptionParams := gravitee_subs.CloseApplicationSubscriptionParams{
				Application:  apiClient.Status.ID,
				Subscription: sub_id,
			}
			//l.Printf("closeSubscriptionParams %v", closeSubscriptionParams)
			closeApplicationSubscriptionParams.SetTimeout(time.Second * time.Duration(c.Timeout))
			closeApplicationSubscriptionParams.SetOrgID(c.OrgID)
			closeApplicationSubscriptionParams.SetEnvID(c.EnvID)
			_, err = c.client_subs.CloseApplicationSubscription(&closeApplicationSubscriptionParams, c.authInfo)
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
	getAPIPlanParams := gravitee_plans.GetAPIPlanParams{
		API:  APIID,
		Plan: PlanID,
	}
	getAPIPlanParams.SetTimeout(time.Second * time.Duration(c.Timeout))
	getAPIPlanParams.SetOrgID(c.OrgID)
	getAPIPlanParams.SetEnvID(c.EnvID)
	plan, err := c.client_plans.GetAPIPlan(&getAPIPlanParams, c.authInfo)
	return plan.Payload, err
}

func (c *APIController) GetPlanByName(APIID string, PlanName string) (*gravitee_models.PlanEntity, error) {
	getAPIPlanParams := gravitee_plans.GetAPIPlansParams{
		API: APIID,
	}
	getAPIPlanParams.SetTimeout(time.Second * time.Duration(c.Timeout))
	getAPIPlanParams.SetOrgID(c.OrgID)
	getAPIPlanParams.SetEnvID(c.EnvID)

	plans, err := c.client_plans.GetAPIPlans(&getAPIPlanParams, c.authInfo)
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

func (c *APIController) GetApplication(AppID string) (*gravitee_models.ApplicationEntity, error) {
	getApplicationParams := gravitee_apps.GetApplicationParams{}
	getApplicationParams.Application = AppID
	getApplicationParams.SetOrgID(c.OrgID)
	getApplicationParams.SetEnvID(c.EnvID)
	getApplicationParams.SetTimeout(time.Second * time.Duration(c.Timeout))
	app, err := c.client_apps.GetApplication(&getApplicationParams, c.authInfo)
	if err != nil {
		l.Printf("unable to GetApplication %s", err)
		return nil, err
	}
	return app.Payload, err
}

func (c *APIController) UpdateApplication(apiClient *platformv1beta1.APIClient) error {
	updateApplicationParams := gravitee_apps.UpdateApplicationParams{}
	updateApplicationParams.Application = apiClient.Status.ID
	updateApplicationParams.Body = &gravitee_models.UpdateApplicationEntity{}
	updateApplicationParams.Body.Name = &apiClient.Spec.Name
	updateApplicationParams.Body.Description = &apiClient.Spec.Description
	updateApplicationParams.Body.Type = apiClient.Spec.Type
	updateApplicationParams.Body.ClientID = apiClient.Spec.ClientID
	updateApplicationParams.SetOrgID(c.OrgID)
	updateApplicationParams.SetEnvID(c.EnvID)
	updateApplicationParams.Body.Settings = &gravitee_models.ApplicationSettings{
		App: &gravitee_models.SimpleApplicationSettings{
			ClientID: apiClient.Spec.ClientID,
			Type:     apiClient.Spec.Type,
		},
	}
	updateApplicationParams.SetTimeout(time.Second * time.Duration(c.Timeout))
	_, err := c.client_apps.UpdateApplication(&updateApplicationParams, c.authInfo)
	if err != nil {
		json_params, _ := json.Marshal(updateApplicationParams)
		l.Printf("updateApplicationParams: %s", json_params)
		l.Printf("unable to DeleteApplication %s", err)
		return err
	}
	return err
}

func (c *APIController) CreateApplication(apiClient *platformv1beta1.APIClient) (*gravitee_models.ApplicationEntity, error) {
	createApplicationParams := gravitee_apps.CreateApplicationParams{}
	createApplicationParams.Application = &gravitee_models.NewApplicationEntity{}
	createApplicationParams.Application.Name = &apiClient.Spec.Name
	createApplicationParams.Application.Description = &apiClient.Spec.Description
	createApplicationParams.Application.Type = apiClient.Spec.Type
	createApplicationParams.Application.ClientID = apiClient.Spec.ClientID
	createApplicationParams.SetOrgID(c.OrgID)
	createApplicationParams.SetEnvID(c.EnvID)
	createApplicationParams.SetTimeout(time.Second * time.Duration(c.Timeout))
	app, err := c.client_apps.CreateApplication(&createApplicationParams, c.authInfo)
	if err != nil {
		json_params, _ := json.Marshal(createApplicationParams)
		l.Printf("createApplicationParams: %s", json_params)
		l.Printf("unable to CreateApplication %s", err)
		return nil, err
	}
	return app.Payload, err
}

func (c *APIController) DeleteApplication(apiClient *platformv1beta1.APIClient) error {
	getApplicationSubscriptionsParams := gravitee_subs.GetApplicationSubscriptionsParams{}
	getApplicationSubscriptionsParams.Application = apiClient.Status.ID
	getApplicationSubscriptionsParams.SetTimeout(time.Second * time.Duration(c.Timeout))
	getApplicationSubscriptionsParams.SetOrgID(c.OrgID)
	getApplicationSubscriptionsParams.SetEnvID(c.EnvID)
	subs, err := c.client_subs.GetApplicationSubscriptions(&getApplicationSubscriptionsParams, c.authInfo)
	if err != nil {
		json_params, _ := json.Marshal(getApplicationSubscriptionsParams)
		l.Printf("getApplicationSubscriptionsParams: %s", json_params)
		l.Panicf("unable to GetApplicationSubscriptions %s", err)
		return err
	}
	for _, sub_ext := range subs.Payload.Data {
		sub_ext_map := sub_ext.(map[string]interface{})
		closeApplicationSubscriptionParams := gravitee_subs.CloseApplicationSubscriptionParams{
			Application:  apiClient.Status.ID,
			Subscription: sub_ext_map["id"].(string),
		}
		closeApplicationSubscriptionParams.SetTimeout(time.Second * time.Duration(c.Timeout))
		closeApplicationSubscriptionParams.SetOrgID(c.OrgID)
		closeApplicationSubscriptionParams.SetEnvID(c.EnvID)
		_, err = c.client_subs.CloseApplicationSubscription(&closeApplicationSubscriptionParams, c.authInfo)
		if err != nil {
			json_params, _ := json.Marshal(closeApplicationSubscriptionParams)
			l.Printf("closeApplicationSubscriptionParams: %s", json_params)
			l.Printf("unable to close Plan %s", err)
			return err
		}
	}
	deleteApplicationParams := gravitee_apps.DeleteApplicationParams{
		Application: apiClient.Status.ID,
	}
	deleteApplicationParams.SetTimeout(time.Second * time.Duration(c.Timeout))
	deleteApplicationParams.SetOrgID(c.OrgID)
	deleteApplicationParams.SetEnvID(c.EnvID)
	_, err = c.client_apps.DeleteApplication(&deleteApplicationParams, c.authInfo)
	if err != nil {
		json_params, _ := json.Marshal(deleteApplicationParams)
		l.Printf("deleteApplicationParams: %s", json_params)
		l.Printf("unable to delete Application %s", err)
		return err
	}
	return nil
}

func (c *APIController) GetAPIAnalytics(apiEndpoint *platformv1beta1.APIEndpoint) (map[string]float64, error) {
	p_type := "stats"
	p_field := "response-time"
	t_interval := time.Second * time.Duration(10)
	t_to := time.Now().UTC()
	t_from := t_to.Add(-t_interval)
	p_interval := t_interval.Milliseconds()
	p_to := t_to.UnixMilli()
	p_from := t_from.UnixMilli()

	getAPIAnalyticsHitsParams := gravitee_analytics.GetAPIAnalyticsHitsParams{}
	getAPIAnalyticsHitsParams.SetOrgID(c.OrgID)
	getAPIAnalyticsHitsParams.SetEnvID(c.EnvID)
	getAPIAnalyticsHitsParams.SetAPI(apiEndpoint.Status.ID)
	getAPIAnalyticsHitsParams.SetType(p_type)
	getAPIAnalyticsHitsParams.SetField(&p_field)
	getAPIAnalyticsHitsParams.SetInterval(&p_interval)
	getAPIAnalyticsHitsParams.SetFrom(&p_from)
	getAPIAnalyticsHitsParams.SetTo(&p_to)
	getAPIAnalyticsHitsParams.SetTimeout(time.Second * time.Duration(c.Timeout))
	results, err := c.client_analytics.GetAPIAnalyticsHits(&getAPIAnalyticsHitsParams, c.authInfo)
	// json_params, _ := json.Marshal(getAPIAnalyticsHitsParams)
	// l.Printf("getAPIAnalyticsHitsParams: %s", json_params)
	if err != nil {
		l.Printf("unable to GetAPIAnalyticsHits %s", err)
		return nil, err
	}
	analytics := make(map[string]float64)
	for k, v := range results.Payload.(map[string]interface{}) {
		analytics[k], _ = v.(json.Number).Float64()
	}
	// json_results, _ := json.Marshal(results.Payload)
	// l.Printf("GetAPIAnalyticsHitsResults: %s", json_results)
	return analytics, nil
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
