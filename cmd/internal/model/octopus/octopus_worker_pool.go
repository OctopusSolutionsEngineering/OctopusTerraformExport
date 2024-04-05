package octopus

type WorkerPool struct {
	Id             string
	SpaceId        string
	Name           string
	Slug           *string
	WorkerPoolType string
	WorkerType     *string
	Description    *string
	IsDefault      bool
	CanAddWorkers  bool
	SortOrder      int
}
