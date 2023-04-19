package octopus

type RunbookProcess struct {
	Id        string
	RunbookId string
	Steps     []Step
}
