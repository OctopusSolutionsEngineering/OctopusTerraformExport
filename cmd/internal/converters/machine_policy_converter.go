package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	octopus2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"time"
)

type MachinePolicyConverter struct {
	Client client.OctopusClient
}

func (c MachinePolicyConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.MachinePolicy]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Machine Policy: " + resource.Id)
		err = c.toHcl(resource, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c MachinePolicyConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus2.MachinePolicy{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	zap.L().Info("Machine Policy: " + resource.Id)
	return c.toHcl(resource, true, dependencies)
}

func (c MachinePolicyConverter) toHcl(machinePolicy octopus2.MachinePolicy, _ bool, dependencies *ResourceDetailsCollection) error {

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
			data := terraform2.TerraformMachinePolicyData{
				Type:        "octopusdeploy_machine_policies",
				Name:        "default_machine_policy",
				Ids:         nil,
				PartialName: &machinePolicy.Name,
				Skip:        0,
				Take:        1,
			}
			file := hclwrite.NewEmptyFile()
			block := gohcl.EncodeAsBlock(data, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a machine policy called \""+data.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.machine_policies) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		} else {

			terraformResource := terraform2.TerraformMachinePolicy{
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
				MachineCleanupPolicy: terraform2.TerraformMachineCleanupPolicy{
					DeleteMachinesBehavior:        &machinePolicy.MachineCleanupPolicy.DeleteMachinesBehavior,
					DeleteMachinesElapsedTimespan: c.convertDurationToNumber(machinePolicy.MachineCleanupPolicy.DeleteMachinesElapsedTimeSpan),
				},
				TerraformMachineConnectivityPolicy: terraform2.TerraformMachineConnectivityPolicy{
					MachineConnectivityBehavior: machinePolicy.MachineConnectivityPolicy.MachineConnectivityBehavior,
				},
				TerraformMachineHealthCheckPolicy: terraform2.TerraformMachineHealthCheckPolicy{
					BashHealthCheckPolicy: terraform2.TerraformBashHealthCheckPolicy{
						RunType:    machinePolicy.MachineHealthCheckPolicy.BashHealthCheckPolicy.RunType,
						ScriptBody: machinePolicy.MachineHealthCheckPolicy.BashHealthCheckPolicy.ScriptBody,
					},
					PowershellHealthCheckPolicy: terraform2.TerraformPowershellHealthCheckPolicy{
						RunType:    machinePolicy.MachineHealthCheckPolicy.PowerShellHealthCheckPolicy.RunType,
						ScriptBody: machinePolicy.MachineHealthCheckPolicy.PowerShellHealthCheckPolicy.ScriptBody,
					},
					HealthCheckCron:         machinePolicy.MachineHealthCheckPolicy.HealthCheckCron,
					HealthCheckCronTimezone: machinePolicy.MachineHealthCheckPolicy.HealthCheckCronTimezone,
					HealthCheckInterval:     c.convertDurationPointerToNumber(machinePolicy.MachineHealthCheckPolicy.HealthCheckInterval),
					HealthCheckType:         machinePolicy.MachineHealthCheckPolicy.HealthCheckType,
				},
				TerraformMachineUpdatePolicy: terraform2.TerraformMachineUpdatePolicy{
					CalamariUpdateBehavior:  machinePolicy.MachineUpdatePolicy.CalamariUpdateBehavior,
					TentacleUpdateAccountId: machinePolicy.MachineUpdatePolicy.TentacleUpdateAccountId,
					TentacleUpdateBehavior:  machinePolicy.MachineUpdatePolicy.TentacleUpdateBehavior,
				},
			}
			file := hclwrite.NewEmptyFile()

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), machinePolicy.Name, "octopusdeploy_machine_policy", policyName))

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
