package octopus

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/samber/lo"
	"strings"
)

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

func (v *Variable) GetId() string {
	if v == nil {
		return ""
	}
	return v.Id
}

func (v *Variable) GetName() string {
	if v == nil {
		return ""
	}
	return v.Name
}

// GetVariableSetId returns the ID of the variable set and the variable. This generates a unique value
// because cloning a project results in duplicate variable IDs.
func (v *Variable) GetVariableSetId(variableSet *VariableSet) string {
	return strutil.EmptyIfNil(variableSet.Id) + "-" + v.Id
}

type Scope struct {
	Environment  []string
	Role         []string
	Machine      []string
	Channel      []string
	TenantTag    []string
	Action       []string
	ProcessOwner []string
}

type Prompt struct {
	Label           *string
	Description     *string
	Required        bool
	DisplaySettings map[string]string
}

func (c *Scope) ScopeDescription(prefix string, suffix string, dependencies *data.ResourceDetailsCollection) string {
	if c == nil {
		return ""
	}

	if len(c.Environment) == 0 && len(c.Role) == 0 && len(c.Machine) == 0 && len(c.Channel) == 0 && len(c.TenantTag) == 0 && len(c.Action) == 0 {
		return ""
	}

	description := "Scoped to "
	scopes := []string{}

	if len(c.Environment) != 0 {
		environments := lo.Map(c.Environment, func(item string, index int) string {
			return dependencies.GetResourceName("Environments", item)
		})
		scopes = append(scopes, " Environments "+strings.Join(environments, ","))
	}

	if len(c.Role) != 0 {
		scopes = append(scopes, " Roles "+strings.Join(c.Role, ","))
	}

	if len(c.Machine) != 0 {
		machines := lo.Map(c.Machine, func(item string, index int) string {
			return dependencies.GetResourceName("Machines", item)
		})
		scopes = append(scopes, " Machine "+strings.Join(machines, ","))
	}

	if len(c.Channel) != 0 {
		channels := lo.Map(c.Channel, func(item string, index int) string {
			return dependencies.GetResourceName("Channels", item)
		})
		scopes = append(scopes, " Channel "+strings.Join(channels, ","))
	}

	if len(c.TenantTag) != 0 {
		scopes = append(scopes, " TenantTag "+strings.Join(c.TenantTag, ","))
	}

	if len(c.Action) != 0 {
		scopes = append(scopes, " Action "+strings.Join(c.Action, ","))
	}

	return prefix + description + strings.Join(scopes, "; ") + suffix
}
