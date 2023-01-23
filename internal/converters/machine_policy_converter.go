package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/sanitizer"
	"strconv"
	"strings"
	"time"
)

type MachinePolicyConverter struct {
	Client client.OctopusClient
}

func (c MachinePolicyConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.MachinePolicy]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, machinePolicy := range collection.Items {
		err = c.toHcl(machinePolicy, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c MachinePolicyConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	machinePolicy := octopus.MachinePolicy{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &machinePolicy)

	if err != nil {
		return err
	}

	return c.toHcl(machinePolicy, true, dependencies)
}

func (c MachinePolicyConverter) toHcl(machinePolicy octopus.MachinePolicy, recursive bool, dependencies *ResourceDetailsCollection) error {

	policyName := "machinepolicy_" + sanitizer.SanitizeName(machinePolicy.Name)

	thisResource := ResourceDetails{}
	thisResource.FileName = "space_population/" + policyName + ".tf"
	thisResource.Id = machinePolicy.Id
	thisResource.ResourceType = c.GetResourceType()

	if machinePolicy.Name == "Default Machine Policy" {
		thisResource.Lookup = "${data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id}"
	} else {
		thisResource.Lookup = "${octopusdeploy_machine_policy." + policyName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {
		if machinePolicy.Name == "Default Machine Policy" {
			data := terraform.TerraformMachinePolicyData{
				Type:        "octopusdeploy_machine_policies",
				Name:        "default_machine_policy",
				Ids:         nil,
				PartialName: &machinePolicy.Name,
				Skip:        0,
				Take:        1,
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(data, "data"))

			return string(file.Bytes()), nil
		} else {

			terraformResource := terraform.TerraformMachinePolicy{
				Type:                         "octopusdeploy_machine_policy",
				Name:                         policyName,
				ResourceName:                 machinePolicy.Name,
				Id:                           nil,
				Description:                  machinePolicy.Description,
				ConnectionConnectTimeout:     c.convertDurationToNumber(machinePolicy.ConnectionConnectTimeout),
				ConnectionRetryCountLimit:    machinePolicy.ConnectionRetryCountLimit,
				ConnectionRetrySleepInterval: c.convertDurationToNumber(machinePolicy.ConnectionRetrySleepInterval),
				ConnectionRetryTimeLimit:     c.convertDurationToNumber(machinePolicy.ConnectionRetryTimeLimit),
				PollingRequestMaximumMessageProcessingTimeout: c.convertDurationToNumber(machinePolicy.PollingRequestMaximumMessageProcessingTimeout),
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
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

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
