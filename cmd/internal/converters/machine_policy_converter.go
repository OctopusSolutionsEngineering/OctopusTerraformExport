package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"strconv"
	"strings"
	"time"
)

const octopusdeployMachinePoliciesDataType = "octopusdeploy_machine_policies"
const octopusdeployMachinePolicyResourceType = "octopusdeploy_machine_policy"

type MachinePolicyConverter struct {
	Client                       client.OctopusClient
	ErrGroup                     *errgroup.Group
	ExcludeMachinePolicies       args.StringSliceArgs
	ExcludeMachinePoliciesRegex  args.StringSliceArgs
	ExcludeMachinePoliciesExcept args.StringSliceArgs
	ExcludeAllMachinePolicies    bool
	Excluder                     ExcludeByName
	LimitResourceCount           int
	IncludeIds                   bool
	IncludeSpaceInPopulation     bool
}

func (c MachinePolicyConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c MachinePolicyConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c MachinePolicyConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllMachinePolicies {
		return nil
	}

	batchClient := client.BatchingOctopusApiClient[octopus.MachinePolicy]{
		Client: c.Client,
	}

	done := make(chan struct{})
	defer close(done)

	channel := batchClient.GetAllResourcesBatch(done, c.GetResourceType())

	for resourceWrapper := range channel {
		if resourceWrapper.Err != nil {
			return resourceWrapper.Err
		}

		resource := resourceWrapper.Res
		if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllMachinePolicies, c.ExcludeMachinePolicies, c.ExcludeMachinePoliciesRegex, c.ExcludeMachinePoliciesExcept) {
			continue
		}

		zap.L().Info("Machine Policy: " + resource.Id)
		err := c.toHcl(resource, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c MachinePolicyConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c MachinePolicyConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c MachinePolicyConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.MachinePolicy{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllMachinePolicies, c.ExcludeMachinePolicies, c.ExcludeMachinePoliciesRegex, c.ExcludeMachinePoliciesExcept) {
		return nil
	}

	zap.L().Info("Machine Policy: " + resource.Id)
	return c.toHcl(resource, true, stateless, dependencies)
}

func (c MachinePolicyConverter) buildData(resourceName string, resource octopus.MachinePolicy) terraform.TerraformMachinePolicyData {
	return terraform.TerraformMachinePolicyData{
		Type:        octopusdeployMachinePoliciesDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: &resource.Name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c MachinePolicyConverter) writeData(file *hclwrite.File, resource octopus.MachinePolicy, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c MachinePolicyConverter) toHcl(machinePolicy octopus.MachinePolicy, _ bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {

	if c.Excluder.IsResourceExcludedWithRegex(machinePolicy.Name, c.ExcludeAllMachinePolicies, c.ExcludeMachinePolicies, c.ExcludeMachinePoliciesRegex, c.ExcludeMachinePoliciesExcept) {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + machinePolicy.Id)
		return nil
	}

	policyName := "machinepolicy_" + sanitizer.SanitizeName(machinePolicy.Name)

	thisResource := data.ResourceDetails{}
	thisResource.Name = machinePolicy.Name
	thisResource.FileName = "space_population/" + policyName + ".tf"
	thisResource.Id = machinePolicy.Id
	thisResource.ResourceType = c.GetResourceType()

	if machinePolicy.Name == "Default Machine Policy" {
		thisResource.Lookup = "${data." + octopusdeployMachinePoliciesDataType + ".default_machine_policy.machine_policies[0].id}"
	} else {
		if stateless {
			thisResource.Lookup = "${length(data." + octopusdeployMachinePoliciesDataType + "." + policyName + ".machine_policies) != 0 " +
				"? data." + octopusdeployMachinePoliciesDataType + "." + policyName + ".machine_policies[0].id " +
				": " + octopusdeployMachinePolicyResourceType + "." + policyName + "[0].id}"
			thisResource.Dependency = "${" + octopusdeployMachinePolicyResourceType + "." + policyName + "}"
		} else {
			thisResource.Lookup = "${" + octopusdeployMachinePolicyResourceType + "." + policyName + ".id}"
		}
	}

	thisResource.ToHcl = func() (string, error) {
		if machinePolicy.Name == "Default Machine Policy" {
			data := c.buildData("default_machine_policy", machinePolicy)
			file := hclwrite.NewEmptyFile()
			block := gohcl.EncodeAsBlock(data, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a machine policy called \""+data.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.machine_policies) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		} else {

			terraformResource := terraform.TerraformMachinePolicy{
				Type:                         octopusdeployMachinePolicyResourceType,
				Name:                         policyName,
				ResourceName:                 machinePolicy.Name,
				Id:                           strutil.InputPointerIfEnabled(c.IncludeIds, &machinePolicy.Id),
				SpaceId:                      strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", machinePolicy.SpaceId)),
				Description:                  machinePolicy.Description,
				ConnectionConnectTimeout:     c.convertDurationToNumber(machinePolicy.ConnectionConnectTimeout),
				ConnectionRetryCountLimit:    machinePolicy.ConnectionRetryCountLimit,
				ConnectionRetrySleepInterval: c.convertDurationToNumber(machinePolicy.ConnectionRetrySleepInterval),
				ConnectionRetryTimeLimit:     c.convertDurationToNumber(machinePolicy.ConnectionRetryTimeLimit),
				//PollingRequestMaximumMessageProcessingTimeout: c.convertDurationToNumber(machinePolicy.PollingRequestMaximumMessageProcessingTimeout),
				MachineCleanupPolicy: terraform.TerraformMachineCleanupPolicy{
					DeleteMachinesBehavior:        &machinePolicy.MachineCleanupPolicy.DeleteMachinesBehavior,
					DeleteMachinesElapsedTimespan: c.convertDurationToNumber(machinePolicy.MachineCleanupPolicy.DeleteMachinesElapsedTimeSpan),
				},
				TerraformMachineConnectivityPolicy: terraform.TerraformMachineConnectivityPolicy{
					MachineConnectivityBehavior: machinePolicy.MachineConnectivityPolicy.MachineConnectivityBehavior,
				},
				TerraformMachineHealthCheckPolicy: terraform.TerraformMachineHealthCheckPolicy{
					BashHealthCheckPolicy: terraform.TerraformBashHealthCheckPolicy{
						RunType:    machinePolicy.MachineHealthCheckPolicy.BashHealthCheckPolicy.RunType,
						ScriptBody: machinePolicy.MachineHealthCheckPolicy.BashHealthCheckPolicy.ScriptBody,
					},
					PowershellHealthCheckPolicy: terraform.TerraformPowershellHealthCheckPolicy{
						RunType:    machinePolicy.MachineHealthCheckPolicy.PowerShellHealthCheckPolicy.RunType,
						ScriptBody: machinePolicy.MachineHealthCheckPolicy.PowerShellHealthCheckPolicy.ScriptBody,
					},
					HealthCheckCron:         machinePolicy.MachineHealthCheckPolicy.HealthCheckCron,
					HealthCheckCronTimezone: machinePolicy.MachineHealthCheckPolicy.HealthCheckCronTimezone,
					HealthCheckInterval:     c.convertDurationPointerToNumber(machinePolicy.MachineHealthCheckPolicy.HealthCheckInterval),
					HealthCheckType:         machinePolicy.MachineHealthCheckPolicy.HealthCheckType,
				},
				TerraformMachineUpdatePolicy: terraform.TerraformMachineUpdatePolicy{
					CalamariUpdateBehavior:  machinePolicy.MachineUpdatePolicy.CalamariUpdateBehavior,
					TentacleUpdateAccountId: machinePolicy.MachineUpdatePolicy.TentacleUpdateAccountId,
					TentacleUpdateBehavior:  machinePolicy.MachineUpdatePolicy.TentacleUpdateBehavior,
				},
			}
			file := hclwrite.NewEmptyFile()

			if stateless {
				c.writeData(file, machinePolicy, policyName)
				terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployMachinePoliciesDataType + "." + policyName + ".machine_policies) != 0 ? 0 : 1}")
			}

			block := gohcl.EncodeAsBlock(terraformResource, "resource")

			if stateless {
				hcl.WriteLifecyclePreventDestroyAttribute(block)
			}

			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c MachinePolicyConverter) GetResourceType() string {
	return "MachinePolicies"
}

// convertDurationToNumber converts the durations returned by the API (e.g. "00:02:00") into nanoseconds.
func (c MachinePolicyConverter) convertDurationToNumber(duration string) *time.Duration {
	zero := time.Duration(0)

	split := strings.Split(duration, ":")
	hours, hourErr := strconv.Atoi(split[0])
	if hourErr != nil {
		return &zero
	}

	min, minErr := strconv.Atoi(split[1])
	if minErr != nil {
		return &zero
	}

	sec, secErr := strconv.Atoi(split[2])
	if secErr != nil {
		return &zero
	}

	seconds := time.Hour*time.Duration(hours) + time.Minute*time.Duration(min) + time.Second*time.Duration(sec)
	return &seconds
}

// convertDurationPointerToNumber converts the durations returned by the API (e.g. "00:02:00") into nanoseconds.
func (c MachinePolicyConverter) convertDurationPointerToNumber(duration *string) *time.Duration {
	zero := time.Duration(0)

	if duration == nil {
		return &zero
	}

	return c.convertDurationToNumber(*duration)
}
