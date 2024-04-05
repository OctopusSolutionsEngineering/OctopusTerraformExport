package octopus

type Target struct {
	SpaceId        string
	EnvironmentIds []string
}

type TargetResource interface {
	GetEnvironmentIds() []string
}

func (t Target) GetEnvironmentIds() []string {
	return t.EnvironmentIds
}
