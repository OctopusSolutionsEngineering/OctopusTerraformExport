package client

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
)

// BatchingOctopusApiClient is a wrapper over the regular client that exposes the ability
// to return a group of resources while making smaller batched requests to the Octopus API.
// This has the benefit of allowing large collections to be processed in a lazy fashion.
type BatchingOctopusApiClient[T any] struct {
	Client OctopusClient
}

// ResultError captures either a successful result or an error.
type ResultError[T any] struct {
	Res T
	Err error
}

// GetAllResourcesBatch retrieves all the resources of a given type but in small batches.
// This allows the resources to be exported in smaller chunks, which is useful for large spaces.
func (c *BatchingOctopusApiClient[T]) GetAllResourcesBatch(done <-chan struct{}, resourceType string) <-chan ResultError[T] {

	pageSize := 30
	chnl := make(chan ResultError[T])

	go func() {
		skip := 0
		items := 0

		defer func() {
			close(chnl)
		}()

		for ok := true; ok; ok = items == pageSize {
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
				case chnl <- ResultError[T]{Res: item, Err: nil}: // Send the item on the channel
				}
			}

			items = len(collection.Items)
			skip += pageSize
		}
	}()

	return chnl
}

// GetAllResourcesArrayBatch retrieves all the resources of a given type as a plain array but in small batches.
// This allows the resources to be exported in smaller chunks, which is useful for large spaces.
func (c *BatchingOctopusApiClient[T]) GetAllResourcesArrayBatch(done <-chan struct{}, resourceType string) <-chan ResultError[T] {

	pageSize := 30
	chnl := make(chan ResultError[T])

	go func() {
		skip := 0
		items := 0

		defer func() {
			close(chnl)
		}()

		for ok := true; ok; ok = items == pageSize {
			collection := []T{}
			err := c.Client.GetAllResources(resourceType, collection, []string{"take", fmt.Sprint(pageSize)}, []string{"skip", fmt.Sprint(skip)})

			if err != nil {
				chnl <- ResultError[T]{Res: *new(T), Err: err}
				break
			}

			for _, item := range collection {
				// https://go.dev/blog/pipelines#explicit-cancellation
				select {
				case <-done:
					// Any signal on the done channel means we should stop processing
					return
				case chnl <- ResultError[T]{Res: item, Err: nil}: // Send the item on the channel
				}
			}

			items = len(collection)
			skip += pageSize
		}
	}()

	return chnl
}

// GetAllGlobalResourcesBatch retrieves all the lobal (i.e. not space specific) resources of a given type but in small batches.
// This allows the resources to be exported in smaller chunks, which is useful for large spaces.
func (c *BatchingOctopusApiClient[T]) GetAllGlobalResourcesBatch(done <-chan struct{}, resourceType string) <-chan ResultError[T] {

	pageSize := 30
	chnl := make(chan ResultError[T])

	go func() {
		skip := 0
		items := 0

		defer func() {
			close(chnl)
		}()

		for ok := true; ok; ok = items == pageSize {
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
				case chnl <- ResultError[T]{Res: item, Err: nil}: // Send the item on the channel
				}
			}

			items = len(collection.Items)
			skip += pageSize
		}
	}()

	return chnl
}
