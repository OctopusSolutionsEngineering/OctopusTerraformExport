package octopus

type LibraryVariableSet struct {
	NameId
	Description   *string
	VariableSetId string
	ContentType   *string
	Templates     []Template
}
