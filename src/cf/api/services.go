package api

import (
	"cf"
	"cf/configuration"
	"cf/errors"
	"cf/models"
	"cf/net"
	"fmt"
	"net/url"
	"strings"
)

type ServiceRepository interface {
	PurgeServiceOffering(offering models.ServiceOffering) errors.Error
	FindServiceOfferingByLabelAndProvider(name, provider string) (offering models.ServiceOffering, apiErr errors.Error)
	GetAllServiceOfferings() (offerings models.ServiceOfferings, apiErr errors.Error)
	GetServiceOfferingsForSpace(spaceGuid string) (offerings models.ServiceOfferings, apiErr errors.Error)
	FindInstanceByName(name string) (instance models.ServiceInstance, apiErr errors.Error)
	CreateServiceInstance(name, planGuid string) (identicalAlreadyExists bool, apiErr errors.Error)
	RenameService(instance models.ServiceInstance, newName string) (apiErr errors.Error)
	DeleteService(instance models.ServiceInstance) (apiErr errors.Error)
	FindServicePlanByDescription(planDescription ServicePlanDescription) (planGuid string, apiErr errors.Error)
	GetServiceInstanceCountForServicePlan(v1PlanGuid string) (count int, apiErr errors.Error)
	MigrateServicePlanFromV1ToV2(v1PlanGuid, v2PlanGuid string) (changedCount int, apiErr errors.Error)
}

type CloudControllerServiceRepository struct {
	config  configuration.Reader
	gateway net.Gateway
}

func NewCloudControllerServiceRepository(config configuration.Reader, gateway net.Gateway) (repo CloudControllerServiceRepository) {
	repo.config = config
	repo.gateway = gateway
	return
}

func (repo CloudControllerServiceRepository) GetServiceOfferingsForSpace(spaceGuid string) (offerings models.ServiceOfferings, apiErr errors.Error) {
	return repo.getServiceOfferings(
		fmt.Sprintf("%s/v2/spaces/%s/services?inline-relations-depth=1", repo.config.ApiEndpoint(), spaceGuid),
	)
}

func (repo CloudControllerServiceRepository) GetAllServiceOfferings() (offerings models.ServiceOfferings, apiErr errors.Error) {
	return repo.getServiceOfferings(
		fmt.Sprintf("%s/v2/services?inline-relations-depth=1", repo.config.ApiEndpoint()),
	)
}

func (repo CloudControllerServiceRepository) getServiceOfferings(path string) (offerings models.ServiceOfferings, apiErr errors.Error) {
	resources := new(PaginatedServiceOfferingResources)
	apiErr = repo.gateway.GetResource(path, repo.config.AccessToken(), resources)
	if apiErr != nil {
		return
	}

	for _, r := range resources.Resources {
		offerings = append(offerings, r.ToModel())
	}

	return
}

func (repo CloudControllerServiceRepository) FindInstanceByName(name string) (instance models.ServiceInstance, apiErr errors.Error) {
	path := fmt.Sprintf("%s/v2/spaces/%s/service_instances?return_user_provided_service_instances=true&q=%s&inline-relations-depth=1", repo.config.ApiEndpoint(), repo.config.SpaceFields().Guid, url.QueryEscape("name:"+name))

	resources := new(PaginatedServiceInstanceResources)
	apiErr = repo.gateway.GetResource(path, repo.config.AccessToken(), resources)
	if apiErr != nil {
		return
	}

	if len(resources.Resources) == 0 {
		apiErr = errors.NewModelNotFoundError("Service instance", name)
		return
	}

	instanceResource := resources.Resources[0]
	instance = instanceResource.ToModel()

	if instanceResource.Entity.ServicePlan.Metadata.Guid != "" {
		resource := &ServiceOfferingResource{}
		path = fmt.Sprintf("%s/v2/services/%s", repo.config.ApiEndpoint(), instanceResource.Entity.ServicePlan.Entity.ServiceOfferingGuid)
		apiErr = repo.gateway.GetResource(path, repo.config.AccessToken(), resource)
		instance.ServiceOffering = resource.ToFields()
	}

	return
}

func (repo CloudControllerServiceRepository) CreateServiceInstance(name, planGuid string) (identicalAlreadyExists bool, apiErr errors.Error) {
	path := fmt.Sprintf("%s/v2/service_instances", repo.config.ApiEndpoint())
	data := fmt.Sprintf(
		`{"name":"%s","service_plan_guid":"%s","space_guid":"%s", "async": true}`,
		name, planGuid, repo.config.SpaceFields().Guid,
	)

	apiErr = repo.gateway.CreateResource(path, repo.config.AccessToken(), strings.NewReader(data))

	if apiErr != nil && apiErr.ErrorCode() == cf.SERVICE_INSTANCE_NAME_TAKEN {
		serviceInstance, findInstanceErr := repo.FindInstanceByName(name)

		if findInstanceErr == nil && serviceInstance.ServicePlan.Guid == planGuid {
			apiErr = nil
			identicalAlreadyExists = true
			return
		}
	}
	return
}

func (repo CloudControllerServiceRepository) RenameService(instance models.ServiceInstance, newName string) (apiErr errors.Error) {
	body := fmt.Sprintf(`{"name":"%s"}`, newName)
	path := fmt.Sprintf("%s/v2/service_instances/%s", repo.config.ApiEndpoint(), instance.Guid)

	if instance.IsUserProvided() {
		path = fmt.Sprintf("%s/v2/user_provided_service_instances/%s", repo.config.ApiEndpoint(), instance.Guid)
	}
	return repo.gateway.UpdateResource(path, repo.config.AccessToken(), strings.NewReader(body))
}

func (repo CloudControllerServiceRepository) DeleteService(instance models.ServiceInstance) (apiErr errors.Error) {
	if len(instance.ServiceBindings) > 0 {
		return errors.NewErrorWithMessage("Cannot delete service instance, apps are still bound to it")
	}
	path := fmt.Sprintf("%s/v2/service_instances/%s", repo.config.ApiEndpoint(), instance.Guid)
	return repo.gateway.DeleteResource(path, repo.config.AccessToken())
}

func (repo CloudControllerServiceRepository) PurgeServiceOffering(offering models.ServiceOffering) errors.Error {
	url := fmt.Sprintf("%s/v2/services/%s?purge=true", repo.config.ApiEndpoint(), offering.Guid)
	return repo.gateway.DeleteResource(url, repo.config.AccessToken())
}

func (repo CloudControllerServiceRepository) FindServiceOfferingByLabelAndProvider(label, provider string) (offering models.ServiceOffering, apiErr errors.Error) {
	path := fmt.Sprintf("%s/v2/services?q=%s", repo.config.ApiEndpoint(), url.QueryEscape("label:"+label+";provider:"+provider))

	resources := new(PaginatedServiceOfferingResources)
	apiErr = repo.gateway.GetResource(path, repo.config.AccessToken(), resources)

	if apiErr != nil {
		return
	} else if len(resources.Resources) == 0 {
		apiErr = errors.NewModelNotFoundError("Service offering", label+" "+provider)
	} else {
		offering = resources.Resources[0].ToModel()
	}
	return
}

func (repo CloudControllerServiceRepository) FindServicePlanByDescription(planDescription ServicePlanDescription) (planGuid string, apiErr errors.Error) {
	path := fmt.Sprintf("%s/v2/services?inline-relations-depth=1&q=%s",
		repo.config.ApiEndpoint(),
		url.QueryEscape("label:"+planDescription.ServiceLabel+";provider:"+planDescription.ServiceProvider))

	response := new(PaginatedServiceOfferingResources)
	apiErr = repo.gateway.GetResource(path, repo.config.AccessToken(), response)
	if apiErr != nil {
		return
	}

	for _, serviceOfferingResource := range response.Resources {
		for _, servicePlanResource := range serviceOfferingResource.Entity.ServicePlans {
			if servicePlanResource.Entity.Name == planDescription.ServicePlanName {
				planGuid = servicePlanResource.Metadata.Guid
				return
			}
		}
	}

	apiErr = errors.NewModelNotFoundError("Plan", planDescription.String())

	return
}

func (repo CloudControllerServiceRepository) MigrateServicePlanFromV1ToV2(v1PlanGuid, v2PlanGuid string) (changedCount int, apiErr errors.Error) {
	path := fmt.Sprintf("%s/v2/service_plans/%s/service_instances", repo.config.ApiEndpoint(), v1PlanGuid)
	body := strings.NewReader(fmt.Sprintf(`{"service_plan_guid":"%s"}`, v2PlanGuid))
	response := new(ServiceMigrateV1ToV2Response)

	apiErr = repo.gateway.UpdateResourceForResponse(path, repo.config.AccessToken(), body, response)
	if apiErr != nil {
		return
	}

	changedCount = response.ChangedCount
	return
}

func (repo CloudControllerServiceRepository) GetServiceInstanceCountForServicePlan(v1PlanGuid string) (count int, apiErr errors.Error) {
	path := fmt.Sprintf("%s/v2/service_plans/%s/service_instances?results-per-page=1", repo.config.ApiEndpoint(), v1PlanGuid)
	response := new(PaginatedServiceInstanceResources)
	apiErr = repo.gateway.GetResource(path, repo.config.AccessToken(), response)
	count = response.TotalResults
	return
}
