package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"golang.org/x/sync/errgroup"
)

// OctopusWorkerPoolProcessor exposes a bunch of common functions for exporting the processes associated with
// projects and runbooks.
type OctopusWorkerPoolProcessor struct {
	WorkerPoolConverter     ConverterAndLookupById
	LookupDefaultWorkerPool bool
	Client                  client.OctopusClient
	ErrGroup                *errgroup.Group
}

// ResolveWorkerPoolId looks up the default worker pool if the action does not specify a pool. This allows
// the exported project to reference the same worker pool rather than falling back to the default.
func (c OctopusWorkerPoolProcessor) ResolveWorkerPoolId(workerPoolId string) (string, error) {
	if !c.LookupDefaultWorkerPool || workerPoolId != "" {
		return workerPoolId, nil
	}

	batchClient := client.BatchingOctopusApiClient[octopus.MachinePolicy]{
		Client: c.Client,
	}

	done := make(chan struct{})
	defer close(done)

	channel := batchClient.GetAllResourcesBatch(done, "WorkerPools")

	for resourceWrapper := range channel {
		if resourceWrapper.Err != nil {
			return "", resourceWrapper.Err
		}

		resource := resourceWrapper.Res
		if resource.IsDefault {
			return resource.Id, nil
		}
	}

	return "", nil
}
