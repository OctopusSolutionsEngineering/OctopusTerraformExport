package octopus

type ProjectGroup struct {
	Id                string
	SpaceId           string
	Name              string
	Description       *string
	EnvironmentIds    []string
	RetentionPolicyId *string
}
