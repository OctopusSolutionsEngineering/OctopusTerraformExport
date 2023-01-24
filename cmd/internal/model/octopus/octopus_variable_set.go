package octopus

type VariableSet struct {
	Id        *string
	Variables []Variable
}

type Variable struct {
	Id          string
	Name        string
	Value       *string
	Description *string
	Scope       Scope
	IsEditable  bool
	Type        string
	IsSensitive bool
	Prompt      Prompt
}

type Scope struct {
	Environment []string
	Role        []string
	Machine     []string
	Channel     []string
	TenantTag   []string
	Action      []string
}

type Prompt struct {
	Label           *string
	Description     *string
	Required        bool
	DisplaySettings map[string]string
}
