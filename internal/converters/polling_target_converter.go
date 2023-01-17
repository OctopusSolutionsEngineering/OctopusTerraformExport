package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type PollingTargetConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
	MachinePolicyMap  map[string]string
	AccountMap        map[string]string
	EnvironmentMap    map[string]string
}

func (c PollingTargetConverter) ToHcl() (map[string]string, map[string]string, error) {
	collection := octopus.GeneralCollection[octopus.PollingEndpointResource]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, nil, err
	}

	results := map[string]string{}
	resultsMap := map[string]string{}

	for _, target := range collection.Items {
		if target.Endpoint.CommunicationStyle == "TentacleActive" {
			targetName := "target_" + util.SanitizeName(target.Name)

			terraformResource := terraform.TerraformPollingTentacleDeploymentTarget{
				Type:                            "octopusdeploy_polling_tentacle_deployment_target",
				Name:                            targetName,
				Environments:                    c.lookupEnvironments(target.EnvironmentIds),
				ResourceName:                    target.Name,
				Roles:                           target.Roles,
				TentacleUrl:                     target.Endpoint.Uri,
				CertificateSignatureAlgorithm:   nil,
				HealthStatus:                    nil,
				IsDisabled:                      &target.IsDisabled,
				MachinePolicyId:                 c.getMachinePolicy(target.MachinePolicyId),
				OperatingSystem:                 nil,
				ShellName:                       &target.ShellName,
				ShellVersion:                    &target.ShellVersion,
				SpaceId:                         nil,
				Status:                          nil,
				StatusSummary:                   nil,
				TenantTags:                      target.TenantTags,
				TenantedDeploymentParticipation: &target.TenantedDeploymentParticipation,
				Tenants:                         target.TenantIds,
				TentacleVersionDetails:          terraform.TerraformTentacleVersionDetails{},
				Uri:                             nil,
				Thumbprint:                      target.Thumbprint,
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			results["space_population/target_"+targetName+".tf"] = string(file.Bytes())
			resultsMap[target.Id] = "${octopusdeploy_polling_tentacle_deployment_target." + targetName + ".id}"
		}
	}

	return results, resultsMap, nil
}

func (c PollingTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c PollingTargetConverter) lookupEnvironments(envs []string) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = c.EnvironmentMap[v]
	}
	return newEnvs
}

func (c PollingTargetConverter) getMachinePolicy(machine string) *string {
	machineLookup, ok := c.MachinePolicyMap[machine]
	if !ok {
		return nil
	}

	return &machineLookup
}
