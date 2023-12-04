package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	octopus2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
)

type WorkerPoolConverter struct {
	Client client.OctopusClient
}

func (c WorkerPoolConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.WorkerPool]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Worker Pool: " + resource.Id)
		err = c.toHcl(resource, false, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c WorkerPoolConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus2.WorkerPool{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	zap.L().Info("Worker Pool: " + resource.Id)
	return c.toHcl(resource, true, false, dependencies)
}

func (c WorkerPoolConverter) ToHclLookupById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	pool := octopus2.WorkerPool{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &pool)

	if err != nil {
		return err
	}

	return c.toHcl(pool, false, true, dependencies)
}

func (c WorkerPoolConverter) toHcl(pool octopus2.WorkerPool, _ bool, lookup bool, dependencies *ResourceDetailsCollection) error {
	resourceName := "workerpool_" + sanitizer.SanitizeNamePointer(&pool.Name)

	thisResource := ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = pool.Id
	thisResource.ResourceType = c.GetResourceType()

	if pool.WorkerPoolType == "DynamicWorkerPool" {
		forceLookup := lookup || pool.Name == "Hosted Windows" || pool.Name == "Hosted Ubuntu"

		if forceLookup {
			c.createDynamicWorkerPoolLookupResource(resourceName, &thisResource, pool)
		} else {
			c.createDynamicWorkerPoolResource(resourceName, &thisResource, pool)
		}
	} else if pool.WorkerPoolType == "StaticWorkerPool" {
		forceLookup := lookup || pool.Name == "Default Worker Pool"

		if forceLookup {
			c.createStaticWorkerPoolLookupResource(resourceName, &thisResource, pool)
		} else {
			c.createStaticWorkerPoolResource(resourceName, &thisResource, pool)
		}
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c WorkerPoolConverter) GetResourceType() string {
	return "WorkerPools"
}

func (c WorkerPoolConverter) createDynamicWorkerPoolLookupResource(resourceName string, thisResource *ResourceDetails, pool octopus2.WorkerPool) {
	thisResource.Lookup = "${data.octopusdeploy_worker_pools." + resourceName + ".worker_pools[0].id}"

	thisResource.ToHcl = func() (string, error) {
		data := terraform2.TerraformWorkerPoolData{
			Type:         "octopusdeploy_worker_pools",
			Name:         resourceName,
			ResourceName: &pool.Name,
			Ids:          nil,
			PartialName:  nil,
			Skip:         0,
			Take:         1,
		}
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(data, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve a worker pool called \""+pool.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.worker_pools) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}
}

func (c WorkerPoolConverter) createDynamicWorkerPoolResource(resourceName string, thisResource *ResourceDetails, pool octopus2.WorkerPool) {
	thisResource.Lookup = "${octopusdeploy_dynamic_worker_pool." + resourceName + ".id}"

	thisResource.ToHcl = func() (string, error) {
		terraformResource := terraform2.TerraformWorkerPool{
			Type:         "octopusdeploy_dynamic_worker_pool",
			Name:         resourceName,
			ResourceName: pool.Name,
			Description:  pool.Description,
			IsDefault:    pool.IsDefault,
			SortOrder:    pool.SortOrder,
			WorkerType:   pool.WorkerType,
		}
		file := hclwrite.NewEmptyFile()

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), pool.Name, "octopusdeploy_dynamic_worker_pool", resourceName))

		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		return string(file.Bytes()), nil
	}
}

func (c WorkerPoolConverter) createStaticWorkerPoolResource(resourceName string, thisResource *ResourceDetails, pool octopus2.WorkerPool) {
	thisResource.Lookup = "${octopusdeploy_static_worker_pool." + resourceName + ".id}"

	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform2.TerraformWorkerPool{
			Type:         "octopusdeploy_static_worker_pool",
			Name:         resourceName,
			ResourceName: pool.Name,
			Description:  pool.Description,
			IsDefault:    pool.IsDefault,
			SortOrder:    pool.SortOrder,
			WorkerType:   pool.WorkerType,
		}
		file := hclwrite.NewEmptyFile()

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), pool.Name, "octopusdeploy_static_worker_pool", resourceName))

		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		return string(file.Bytes()), nil
	}
}

func (c WorkerPoolConverter) createStaticWorkerPoolLookupResource(resourceName string, thisResource *ResourceDetails, pool octopus2.WorkerPool) {
	thisResource.Lookup = "${data.octopusdeploy_worker_pools." + resourceName + ".worker_pools[0].id}"

	thisResource.ToHcl = func() (string, error) {
		data := terraform2.TerraformWorkerPoolData{
			Type:         "octopusdeploy_worker_pools",
			Name:         resourceName,
			ResourceName: &pool.Name,
			Ids:          nil,
			PartialName:  nil,
			Skip:         0,
			Take:         1,
		}
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(data, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve a worker pool called \""+pool.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.worker_pools) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil

	}
}
