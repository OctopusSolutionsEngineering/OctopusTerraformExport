package model

type VariableSet struct {
	Id        *string
	Variables []Variable
}

type Variable struct {
	Id          *string
	Name        *string
	Value       *string
	Description *string
	IsEditable  bool
	Type        *string
	IsSensitive bool
	Prompt      Prompt
}

type Prompt struct {
	Label           *string
	Description     *string
	Required        bool
	DisplaySettings map[string]string
}
