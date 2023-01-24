package octopus

type ProjectGroup struct {
	Id                string
	Name              string
	Description       *string
	EnvironmentIds    []string
	RetentionPolicyId *string
}
