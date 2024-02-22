package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

// FilterSteps removes any actions tha the terraform provider does not support (yet), and removes any
// empty steps that result from this filtering process.
func FilterSteps(steps []octopus.Step) []octopus.Step {
	return lo.Filter(steps, func(item octopus.Step, index int) bool {

		// Valid actions are those that are not from the new steps framework
		validActions := lo.Filter(item.Actions, func(item octopus.Action, index int) bool {
			if len(item.Inputs) != 0 {
				zap.L().Error("Action " + strutil.EmptyIfNil(item.Name) +
					" has the \"Items\" property, which indicates that it is from the new step framework. " +
					"These steps are not supported and are not exported.")
				return false
			}
			return true
		})

		item.Actions = validActions

		// valid steps have at least one action
		return len(validActions) != 0
	})
}
