package client

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
)

type GenericOctopusApiClient[T any] struct {
	Client OctopusClient
}

type ResultError[T any] struct {
	Res T
	Err error
}

// GetAllResourcesBatch retrieves all the resources of a given type but in small batches.
// This allows the resources to be exported in smaller chunks, which is useful for large spaces.
func (c *GenericOctopusApiClient[T]) GetAllResourcesBatch(done <-chan struct{}, resourceType string) <-chan ResultError[T] {

	pageSize := 30
	chnl := make(chan ResultError[T])

	go func() {
		skip := 0
		items := 0

		defer func() {
			close(chnl)
		}()

		for ok := true; ok; ok = (items != 0) {
			collection := new(octopus.GeneralCollection[T])
			err := c.Client.GetAllResources(resourceType, collection, []string{"take", fmt.Sprint(pageSize)}, []string{"skip", fmt.Sprint(skip)})

			if err != nil {
				chnl <- ResultError[T]{Res: *new(T), Err: err}
				break
			}

			for _, item := range collection.Items {
				// https://go.dev/blog/pipelines#explicit-cancellation
				select {
				case <-done:
					// Any signal on the done channel means we should stop processing
					return
				default:
					// No messages on the done channel means we send the next item
					chnl <- ResultError[T]{Res: item, Err: nil}
				}
			}

			items = len(collection.Items)
			skip += pageSize
		}
	}()

	return chnl
}
