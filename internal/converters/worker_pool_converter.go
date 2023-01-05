package converters

import (
	"fmt"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type WorkerPoolConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
}

func (c WorkerPoolConverter) ToHcl() (map[string]string, map[string]string, error) {
	collection := octopus.GeneralCollection[octopus.WorkerPool]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, nil, err
	}

	results := map[string]string{}
	workerPoolMap := map[string]string{}

	for _, pool := range collection.Items {
		resourceName := "workerpool_" + util.SanitizeName(&pool.Name)

		if pool.WorkerPoolType == "DynamicWorkerPool" {
			/*
				These default pools are expected to be created in a new space, so
				we use a data lookup to reference them rather than create them.
			*/
			if pool.Name == "Hosted Windows" || pool.Name == "Hosted Ubuntu" {
				data := terraform.TerraformWorkerPoolData{
					Type:         "octopusdeploy_worker_pools",
					Name:         resourceName,
					ResourceName: &pool.Name,
					Ids:          nil,
					PartialName:  nil,
					Skip:         0,
					Take:         1,
				}
				file := hclwrite.NewEmptyFile()
				file.Body().AppendBlock(gohcl.EncodeAsBlock(data, "data"))

				results["space_population/"+resourceName+".tf"] = string(file.Bytes())
				workerPoolMap[pool.Id] = "${data.octopusdeploy_worker_pools." + resourceName + ".worker_pools[0].id}"
			} else {
				terraformResource := terraform.TerraformWorkerPool{
					Type:         "octopusdeploy_dynamic_worker_pool",
					Name:         resourceName,
					ResourceName: pool.Name,
					Description:  pool.Description,
					IsDefault:    pool.IsDefault,
					SortOrder:    pool.SortOrder,
					WorkerType:   pool.WorkerType,
				}
				file := hclwrite.NewEmptyFile()
				file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

				results["space_population/"+resourceName+".tf"] = string(file.Bytes())
				workerPoolMap[pool.Id] = "${octopusdeploy_dynamic_worker_pool." + resourceName + ".id}"
			}
		}

		if pool.WorkerPoolType == "StaticWorkerPool" {
			/*
				This is the default pool available in every space. Use a data lookup for this pool.
			*/
			if pool.Name == "Default Worker Pool" {
				data := terraform.TerraformWorkerPoolData{
					Type:         "octopusdeploy_worker_pools",
					Name:         resourceName,
					ResourceName: &pool.Name,
					Ids:          nil,
					PartialName:  nil,
					Skip:         0,
					Take:         1,
				}
				file := hclwrite.NewEmptyFile()
				file.Body().AppendBlock(gohcl.EncodeAsBlock(data, "data"))

				results["space_population/"+resourceName+".tf"] = string(file.Bytes())
				workerPoolMap[pool.Id] = "${data.octopusdeploy_worker_pools." + resourceName + ".worker_pools[0].id}"
			} else {
				terraformResource := terraform.TerraformWorkerPool{
					Type:         "octopusdeploy_static_worker_pool",
					Name:         resourceName,
					ResourceName: pool.Name,
					Description:  pool.Description,
					IsDefault:    pool.IsDefault,
					SortOrder:    pool.SortOrder,
					WorkerType:   pool.WorkerType,
				}
				file := hclwrite.NewEmptyFile()
				file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

				results["space_population/"+resourceName+".tf"] = string(file.Bytes())
				workerPoolMap[pool.Id] = "${octopusdeploy_static_worker_pool." + resourceName + ".id}"
			}
		}
	}

	return results, workerPoolMap, nil
}

func (c WorkerPoolConverter) ToHclById(id string) (map[string]string, error) {
	pool := octopus.WorkerPool{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &pool)

	if err != nil {
		return nil, err
	}

	resourceName := "workerpool_" + util.SanitizeName(&pool.Name)

	if pool.WorkerPoolType == "DynamicWorkerPool" {
		terraformResource := terraform.TerraformWorkerPool{
			Type:         "octopusdeploy_dynamic_worker_pool",
			Name:         resourceName,
			ResourceName: pool.Name,
			Description:  pool.Description,
			IsDefault:    pool.IsDefault,
			SortOrder:    pool.SortOrder,
			WorkerType:   pool.WorkerType,
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		return map[string]string{
			resourceName + ".tf": string(file.Bytes()),
		}, nil
	}

	if pool.WorkerPoolType == "StaticWorkerPool" {
		terraformResource := terraform.TerraformWorkerPool{
			Type:         "octopusdeploy_static_worker_pool",
			Name:         resourceName,
			ResourceName: pool.Name,
			Description:  pool.Description,
			IsDefault:    pool.IsDefault,
			SortOrder:    pool.SortOrder,
			WorkerType:   pool.WorkerType,
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		return map[string]string{
			resourceName + ".tf": string(file.Bytes()),
		}, nil
	}

	fmt.Println("Worker pool type of " + pool.WorkerPoolType + " was unexpected.")

	return map[string]string{}, nil
}

func (c WorkerPoolConverter) ToHclByName(name string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (c WorkerPoolConverter) GetResourceType() string {
	return "WorkerPools"
}
