package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const octopusdeployWorkerPoolsDataType = "octopusdeploy_worker_pools"
const octopusdeployStaticWorkerPoolResourcePool = "octopusdeploy_static_worker_pool"
const octopusdeployDynamicWorkerPoolResourceType = "octopusdeploy_dynamic_worker_pool"

type WorkerPoolConverter struct {
	Client                   client.OctopusClient
	ErrGroup                 *errgroup.Group
	ExcludeWorkerpools       args.StringSliceArgs
	ExcludeWorkerpoolsRegex  args.StringSliceArgs
	ExcludeWorkerpoolsExcept args.StringSliceArgs
	ExcludeAllWorkerpools    bool
	Excluder                 ExcludeByName
	LimitResourceCount       int
	IncludeSpaceInPopulation bool
	IncludeIds               bool
}

func (c WorkerPoolConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c WorkerPoolConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c WorkerPoolConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllWorkerpools {
		return nil
	}

	batchClient := client.BatchingOctopusApiClient[octopus.WorkerPool]{
		Client: c.Client,
	}

	done := make(chan struct{})
	defer close(done)

	channel := batchClient.GetAllResourcesBatch(done, c.GetResourceType())

	for resourceWrapper := range channel {
		if resourceWrapper.Err != nil {
			return resourceWrapper.Err
		}

		resource := resourceWrapper.Res
		if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllWorkerpools, c.ExcludeWorkerpools, c.ExcludeWorkerpoolsRegex, c.ExcludeWorkerpoolsExcept) {
			continue
		}

		zap.L().Info("Worker Pool: " + resource.Id)
		err := c.toHcl(resource, false, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c WorkerPoolConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c WorkerPoolConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c WorkerPoolConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.WorkerPool{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllWorkerpools, c.ExcludeWorkerpools, c.ExcludeWorkerpoolsRegex, c.ExcludeWorkerpoolsExcept) {
		return nil
	}

	zap.L().Info("Worker Pool: " + resource.Id)
	return c.toHcl(resource, true, false, stateless, dependencies)
}

func (c WorkerPoolConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	pool := octopus.WorkerPool{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &pool)

	if err != nil {
		return err
	}

	if c.Excluder.IsResourceExcludedWithRegex(pool.Name, c.ExcludeAllWorkerpools, c.ExcludeWorkerpools, c.ExcludeWorkerpoolsRegex, c.ExcludeWorkerpoolsExcept) {
		return nil
	}

	return c.toHcl(pool, false, true, false, dependencies)
}

func (c WorkerPoolConverter) buildData(resourceName string, resource octopus.WorkerPool) terraform.TerraformWorkerPoolData {
	return terraform.TerraformWorkerPoolData{
		Type:         octopusdeployWorkerPoolsDataType,
		Name:         resourceName,
		ResourceName: nil,
		Ids:          nil,
		PartialName:  &resource.Name,
		Skip:         0,
		Take:         1,
	}
}

// writeData appends the data block for stateless modules
func (c WorkerPoolConverter) writeData(file *hclwrite.File, resource octopus.WorkerPool, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c WorkerPoolConverter) toHcl(pool octopus.WorkerPool, _ bool, lookup bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.Excluder.IsResourceExcludedWithRegex(pool.Name, c.ExcludeAllWorkerpools, c.ExcludeWorkerpools, c.ExcludeWorkerpoolsRegex, c.ExcludeWorkerpoolsExcept) {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + pool.Id)
		return nil
	}

	resourceName := "workerpool_" + sanitizer.SanitizeNamePointer(&pool.Name)

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = pool.Id
	thisResource.Name = pool.Name
	thisResource.ResourceType = c.GetResourceType()

	if pool.WorkerPoolType == "DynamicWorkerPool" {
		forceLookup := lookup || pool.Name == "Hosted Windows" || pool.Name == "Hosted Ubuntu"

		if forceLookup {
			c.createDynamicWorkerPoolLookupResource(resourceName, &thisResource, pool)
		} else {
			c.createDynamicWorkerPoolResource(resourceName, &thisResource, dependencies, pool, stateless)
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

func (c WorkerPoolConverter) createDynamicWorkerPoolLookupResource(resourceName string, thisResource *data.ResourceDetails, pool octopus.WorkerPool) {
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

func (c WorkerPoolConverter) createDynamicWorkerPoolResource(resourceName string, thisResource *data.ResourceDetails, dependencies *data.ResourceDetailsCollection, pool octopus.WorkerPool, stateless bool) {
	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools) != 0 " +
			"? data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools[0].id " +
			": " + octopusdeployDynamicWorkerPoolResourceType + "." + resourceName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployDynamicWorkerPoolResourceType + "." + resourceName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployDynamicWorkerPoolResourceType + "." + resourceName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {
		terraformResource := terraform.TerraformWorkerPool{
			Type:         octopusdeployDynamicWorkerPoolResourceType,
			Name:         resourceName,
			Id:           strutil.InputPointerIfEnabled(c.IncludeIds, &pool.Id),
			SpaceId:      strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", pool.SpaceId)),
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

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDestroyAttribute(block)
		}

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}
}

func (c WorkerPoolConverter) createStaticWorkerPoolResource(resourceName string, thisResource *data.ResourceDetails, pool octopus.WorkerPool, stateless bool) {
	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools) != 0 " +
			"? data." + octopusdeployWorkerPoolsDataType + "." + resourceName + ".worker_pools[0].id " +
			": " + octopusdeployStaticWorkerPoolResourcePool + "." + resourceName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployStaticWorkerPoolResourcePool + "." + resourceName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployStaticWorkerPoolResourcePool + "." + resourceName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformWorkerPool{
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

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDestroyAttribute(block)
		}

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}
}

func (c WorkerPoolConverter) createStaticWorkerPoolLookupResource(resourceName string, thisResource *data.ResourceDetails, pool octopus.WorkerPool) {
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
