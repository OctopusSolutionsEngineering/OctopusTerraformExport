package octopus

type RunbookProcess struct {
	Id        string
	ProjectId string
	RunbookId string
	Steps     []Step
}
