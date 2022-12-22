package octopus

type Lifecycle struct {
	Id                      string
	Name                    *string
	Slug                    *string
	Description             *string
	Phases                  []Phase
	ReleaseRetentionPolicy  Policy
	TentacleRetentionPolicy Policy
}

type Phase struct {
	Id                                 string
	Name                               *string
	AutomaticDeploymentTargets         []string
	OptionalDeploymentTargets          []string
	MinimumEnvironmentsBeforePromotion int
	IsOptionalPhase                    bool
	ReleaseRetentionPolicy             Policy
	TentacleRetentionPolicy            Policy
}

type Policy struct {
	Unit              *string
	QuantityToKeep    *int
	ShouldKeepForever *bool
}
