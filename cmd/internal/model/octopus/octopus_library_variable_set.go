package octopus

type LibraryVariableSet struct {
	Id            string
	Name          string
	Description   *string
	VariableSetId string
	ContentType   *string
	Templates     []Template
}
