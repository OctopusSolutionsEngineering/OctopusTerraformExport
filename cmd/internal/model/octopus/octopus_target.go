package octopus

type Target struct {
	EnvironmentIds []string
}

type TargetResource interface {
	GetEnvironmentIds() []string
}

func (t Target) GetEnvironmentIds() []string {
	return t.EnvironmentIds
}
