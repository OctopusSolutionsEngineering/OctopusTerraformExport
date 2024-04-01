package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

// FilterSteps removes any actions tha the terraform provider does not support (yet), and removes any
// empty steps that result from this filtering process.
func FilterSteps(steps []octopus.Step, IgnoreInvalidExcludeExcept bool, Excluder ExcludeByName, ExcludeAllSteps bool, ExcludeSteps args.StringSliceArgs, ExcludeStepsRegex args.StringSliceArgs, ExcludeStepsExcept args.StringSliceArgs) []octopus.Step {

	// If invalid exceptions are ignored, we need to check every entry in the ExcludeStepsExcept collection
	// to make sure it references a valid step.
	if IgnoreInvalidExcludeExcept {
		// ExcludeStepsExcept will only include items that are valid step names.
		// This may result in an empty ExcludeStepsExcept collection, in which case all steps are included in the export.
		// This is to guard against invalid step names resulting in no steps being exported.
		ExcludeStepsExcept = lo.Filter(ExcludeStepsExcept, func(exclusion string, index int) bool {
			return lo.ContainsBy(steps, func(step octopus.Step) bool {
				return strutil.EmptyIfNil(step.Name) == exclusion
			})
		})
	}

	return lo.Filter(steps, func(item octopus.Step, index int) bool {

		if Excluder.IsResourceExcludedWithRegex(strutil.EmptyIfNil(item.Name), ExcludeAllSteps, ExcludeSteps, ExcludeStepsRegex, ExcludeStepsExcept) {
			return false
		}

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
