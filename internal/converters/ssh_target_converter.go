package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type SshTargetConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
	MachinePolicyMap  map[string]string
	AccountMap        map[string]string
	EnvironmentMap    map[string]string
}

func (c SshTargetConverter) ToHcl() (map[string]string, map[string]string, error) {
	collection := octopus.GeneralCollection[octopus.SshEndpointResource]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, nil, err
	}

	results := map[string]string{}
	resultsMap := map[string]string{}

	for _, target := range collection.Items {
		if target.Endpoint.CommunicationStyle == "Ssh" {
			targetName := "target_" + util.SanitizeName(target.Name)

			terraformResource := terraform.TerraformSshConnectionDeploymentTarget{
				Type:               "octopusdeploy_ssh_connection_deployment_target",
				Name:               targetName,
				AccountId:          c.getAccount(target.Endpoint.AccountId),
				Environments:       c.lookupEnvironments(target.EnvironmentIds),
				Fingerprint:        target.Endpoint.Fingerprint,
				Host:               target.Endpoint.Host,
				ResourceName:       target.Name,
				Roles:              target.Roles,
				DotNetCorePlatform: &target.Endpoint.DotNetCorePlatform,
				MachinePolicyId:    c.getMachinePolicy(target.MachinePolicyId),
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			results["space_population/target_"+targetName+".tf"] = string(file.Bytes())
			resultsMap[target.Id] = "${octopusdeploy_ssh_connection_deployment_target." + targetName + ".id}"
		}
	}

	return results, resultsMap, nil
}

func (c SshTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c SshTargetConverter) lookupEnvironments(envs []string) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = c.EnvironmentMap[v]
	}
	return newEnvs
}

func (c SshTargetConverter) getAccount(account string) string {
	accountLookup, ok := c.AccountMap[account]
	if !ok {
		return ""
	}

	return accountLookup
}

func (c SshTargetConverter) getMachinePolicy(machine string) *string {
	machineLookup, ok := c.MachinePolicyMap[machine]
	if !ok {
		return nil
	}

	return &machineLookup
}
