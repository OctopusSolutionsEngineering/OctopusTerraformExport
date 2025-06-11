package octopus

type RunbookProcess struct {
	Id        string
	RunbookId string
	Steps     []Step
}

func (a *RunbookProcess) GetId() string {
	return a.Id
}

func (a *RunbookProcess) GetParentId() string {
	return a.RunbookId
}

func (a *RunbookProcess) GetSteps() []Step {
	return a.Steps
}
