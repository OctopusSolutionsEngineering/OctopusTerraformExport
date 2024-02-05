package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	octopus2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
)

const octopusdeployWorkerPoolsDataType = "octopusdeploy_worker_pools"
const octopusdeployStaticWorkerPoolResourcePool = "octopusdeploy_static_worker_pool"
const octopusdeployDynamicWorkerPoolResourceType = "octopusdeploy_dynamic_worker_pool"

type WorkerPoolConverter struct {
	Client client.OctopusClient
}

func (c WorkerPoolConverter) AllToHcl(dependencies *ResourceDetailsCollection) error {
	return c.allToHcl(false, dependencies)
}

func (c WorkerPoolConverter) AllToStatelessHcl(dependencies *ResourceDetailsCollection) error {
	return c.allToHcl(true, dependencies)
}

func (c WorkerPoolConverter) allToHcl(stateless bool, dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.WorkerPool]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Worker Pool: " + resource.Id)
		err = c.toHcl(resource, false, false, stateless, dependencies)

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
	return c.toHcl(resource, true, false, false, dependencies)
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

	return c.toHcl(pool, false, true, false, dependencies)
}

func (c WorkerPoolConverter) buildData(resourceName string, resource octopus2.WorkerPool) terraform2.TerraformWorkerPoolData {
	return terraform2.TerraformWorkerPoolData{
		Type:         octopusdeployWorkerPoolsDataType,
		Name:         resourceName,
		ResourceName: &resource.Name,
		Ids:          nil,
		PartialName:  nil,
		Skip:         0,
		Take:         1,
	}
}

// writeData appends the data block for stateless modules
func (c WorkerPoolConverter) writeData(file *hclwrite.File, resource octopus2.WorkerPool, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c WorkerPoolConverter) toHcl(pool octopus2.WorkerPool, _ bool, lookup bool, stateless bool, dependencies *ResourceDetailsCollection) error {
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
			c.createDynamicWorkerPoolResource(resourceName, &thisResource, pool, stateless)
		}
	} else if pool.WorkerPoolType == "StaticWorkerPool" {
		forceLookup := lookup || pool.Name == "Default Worker Pool"

		if forceLookup {
			c.createStaticWorkerPoolLookupResource(resourceName, &thisResource, pool)
		} else {
			c.createStaticWorkerPoolResource(resourceName, &thisResource, pool, stateless)
		}
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c WorkerPoolConverter) GetResourceType() string {
	return "WorkerPools"
}

func (c WorkerPoolConverter) createDynamicWorkerPoolLookupResource(resourceName string, thisResource *ResourceDetails, pool octopus2.WorkerPool) {
	thisResource.Lookup = "${data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools[0].id}"

	thisResource.ToHcl = func() (string, error) {
		data := c.buildData(resourceName, pool)
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(data, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve a worker pool called \""+pool.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.worker_pools) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}
}

func (c WorkerPoolConverter) createDynamicWorkerPoolResource(resourceName string, thisResource *ResourceDetails, pool octopus2.WorkerPool, stateless bool) {
	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools) != 0 " +
			"? data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools[0].id " +
			": " + octopusdeployDynamicWorkerPoolResourceType + "." + resourceName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployDynamicWorkerPoolResourceType + "." + resourceName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployDynamicWorkerPoolResourceType + "." + resourceName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {
		terraformResource := terraform2.TerraformWorkerPool{
			Type:         octopusdeployDynamicWorkerPoolResourceType,
			Name:         resourceName,
			ResourceName: pool.Name,
			Description:  pool.Description,
			IsDefault:    pool.IsDefault,
			SortOrder:    pool.SortOrder,
			WorkerType:   pool.WorkerType,
		}
		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, pool, resourceName)
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools) != 0 ? 0 : 1}")
		}

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), pool.Name, octopusdeployDynamicWorkerPoolResourceType, resourceName))

		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		return string(file.Bytes()), nil
	}
}

func (c WorkerPoolConverter) createStaticWorkerPoolResource(resourceName string, thisResource *ResourceDetails, pool octopus2.WorkerPool, stateless bool) {
	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools) != 0 " +
			"? data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools[0].id " +
			": " + octopusdeployStaticWorkerPoolResourcePool + "." + resourceName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployStaticWorkerPoolResourcePool + "." + resourceName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployStaticWorkerPoolResourcePool + "." + resourceName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform2.TerraformWorkerPool{
			Type:         octopusdeployStaticWorkerPoolResourcePool,
			Name:         resourceName,
			ResourceName: pool.Name,
			Description:  pool.Description,
			IsDefault:    pool.IsDefault,
			SortOrder:    pool.SortOrder,
			WorkerType:   pool.WorkerType,
		}
		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, pool, resourceName)
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools) != 0 ? 0 : 1}")
		}

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), pool.Name, octopusdeployStaticWorkerPoolResourcePool, resourceName))

		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		return string(file.Bytes()), nil
	}
}

func (c WorkerPoolConverter) createStaticWorkerPoolLookupResource(resourceName string, thisResource *ResourceDetails, pool octopus2.WorkerPool) {
	thisResource.Lookup = "${data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools[0].id}"

	thisResource.ToHcl = func() (string, error) {
		data := c.buildData(resourceName, pool)
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(data, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve a worker pool called \""+pool.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.worker_pools) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}
}
