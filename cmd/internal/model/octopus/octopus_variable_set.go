package octopus

import "strings"

type VariableSet struct {
	Id        *string
	OwnerId   *string
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

func (c *Scope) ScopeDescription(prefix string, suffix string) string {
	if c == nil {
		return ""
	}

	if len(c.Environment) == 0 && len(c.Role) == 0 && len(c.Machine) == 0 && len(c.Channel) == 0 && len(c.TenantTag) == 0 && len(c.Action) == 0 {
		return ""
	}

	description := "Scoped to "
	scopes := []string{}

	if len(c.Environment) != 0 {
		scopes = append(scopes, " Environments "+strings.Join(c.Environment, ","))
	}

	if len(c.Role) != 0 {
		scopes = append(scopes, " Roles "+strings.Join(c.Role, ","))
	}

	if len(c.Machine) != 0 {
		scopes = append(scopes, " Machine "+strings.Join(c.Role, ","))
	}

	if len(c.Channel) != 0 {
		scopes = append(scopes, " Channel "+strings.Join(c.Role, ","))
	}

	if len(c.TenantTag) != 0 {
		scopes = append(scopes, " TenantTag "+strings.Join(c.Role, ","))
	}

	if len(c.Action) != 0 {
		scopes = append(scopes, " Action "+strings.Join(c.Role, ","))
	}

	return prefix + description + strings.Join(scopes, "; ") + suffix
}
