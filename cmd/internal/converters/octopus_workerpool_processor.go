package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	octopus2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
)

// OctopusWorkerPoolProcessor exposes a bunch of common functions for exporting the processes associated with
// projects and runbooks.
type OctopusWorkerPoolProcessor struct {
	WorkerPoolConverter     ConverterAndLookupById
	LookupDefaultWorkerPool bool
	Client                  client.OctopusClient
}

// ResolveWorkerPoolId looks up the default worker pool if the action does not specify a pool. This allows
// the exported project to reference the same worker pool rather than falling back to the default.
func (c OctopusWorkerPoolProcessor) ResolveWorkerPoolId(workerPoolId string) (string, error) {
	if !c.LookupDefaultWorkerPool || workerPoolId != "" {
		return workerPoolId, nil
	}

	collection := octopus2.GeneralCollection[octopus2.WorkerPool]{}
	err := c.Client.GetAllResources("WorkerPools", &collection, []string{"take", "1000"})

	if err != nil {
		return "", err
	}

	for _, resource := range collection.Items {
		if resource.IsDefault {
			return resource.Id, nil
		}
	}

	return "", nil
}
