package converters

import (
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
	"github.com/samber/lo"
	"go.uber.org/zap"
)

const octopusdeployAzureServiceFabricClusterDeploymentDataType = "octopusdeploy_deployment_targets"
const octopusdeployAzureServiceFabricClusterDeploymentResourceType = "octopusdeploy_azure_service_fabric_cluster_deployment_target"

type AzureServiceFabricTargetConverter struct {
	Client                    client.OctopusClient
	MachinePolicyConverter    ConverterById
	EnvironmentConverter      ConverterById
	ExcludeAllTargets         bool
	ExcludeTargets            args.ExcludeTargets
	ExcludeTargetsRegex       args.ExcludeTargets
	ExcludeTargetsExcept      args.ExcludeTargets
	DummySecretVariableValues bool
	DummySecretGenerator      DummySecretGenerator
	ExcludeTenantTags         args.ExcludeTenantTags
	ExcludeTenantTagSets      args.ExcludeTenantTagSets
	Excluder                  ExcludeByName
	TagSetConverter           ConvertToHclByResource[octopus.TagSet]
}

func (c AzureServiceFabricTargetConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) error {
	return c.allToHcl(false, dependencies)
}

func (c AzureServiceFabricTargetConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) error {
	return c.allToHcl(true, dependencies)
}

func (c AzureServiceFabricTargetConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllTargets {
		return nil
	}

	collection := octopus.GeneralCollection[octopus.AzureServiceFabricResource]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	targets := lo.Filter(collection.Items, func(item octopus.AzureServiceFabricResource, index int) bool {
		return c.isAzureServiceFabricCluster(item)
	})

	for _, resource := range targets {
		zap.L().Info("Azure Service Fabric Target: " + resource.Id)
		err = c.toHcl(resource, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c AzureServiceFabricTargetConverter) isAzureServiceFabricCluster(resource octopus.AzureServiceFabricResource) bool {
	return resource.Endpoint.CommunicationStyle == "AzureServiceFabricCluster"
}

func (c AzureServiceFabricTargetConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c AzureServiceFabricTargetConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c AzureServiceFabricTargetConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.AzureServiceFabricResource{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	if !c.isAzureServiceFabricCluster(resource) {
		return nil
	}

	zap.L().Info("Azure Service Fabric Target: " + resource.Id)
	return c.toHcl(resource, true, stateless, dependencies)
}

func (c AzureServiceFabricTargetConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.AzureServiceFabricResource{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	if !c.isAzureServiceFabricCluster(resource) {
		return nil
	}

	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsRegex, c.ExcludeTargetsExcept) {
		return nil
	}

	if !c.isAzureServiceFabricCluster(resource) {
		return nil
	}

	thisResource := data.ResourceDetails{}

	resourceName := "target_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data." + octopusdeployAzureServiceFabricClusterDeploymentDataType + "." + resourceName + ".deployment_targets[0].id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := c.buildData(resourceName, resource)
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve a deployment target called \""+resource.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.deployment_targets) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c AzureServiceFabricTargetConverter) buildData(resourceName string, resource octopus.AzureServiceFabricResource) terraform.TerraformDeploymentTargetsData {
	return terraform.TerraformDeploymentTargetsData{
		Type:        octopusdeployAzureServiceFabricClusterDeploymentDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: &resource.Name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c AzureServiceFabricTargetConverter) writeData(file *hclwrite.File, resource octopus.AzureServiceFabricResource, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c AzureServiceFabricTargetConverter) toHcl(target octopus.AzureServiceFabricResource, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(target.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsRegex, c.ExcludeTargetsExcept) {
		return nil
	}

	if !c.isAzureServiceFabricCluster(target) {
		return nil
	}

	if recursive {
		err := c.exportDependencies(target, dependencies)

		if err != nil {
			return err
		}
	}

	targetName := "target_" + sanitizer.SanitizeName(target.Name)

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + targetName + ".tf"
	thisResource.Id = target.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${" + octopusdeployAzureServiceFabricClusterDeploymentResourceType + "." + targetName + ".id}"
	thisResource.Parameters = []data.ResourceParameter{
		{
			VariableName:  targetName,
			Label:         "Service Fabric Target " + target.Name + " aad_user_credential_password",
			Description:   "The aad_user_credential_password value associated with the target \"" + target.Name + "\"",
			ResourceName:  sanitizer.SanitizeParameterName(dependencies, target.Name, "AadUserCredentialPassword"),
			ParameterType: "AadUserCredentialPassword",
			Sensitive:     true,
		},
	}

	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployAzureServiceFabricClusterDeploymentDataType + "." + targetName + ".deployment_targets) != 0 " +
			"? data." + octopusdeployAzureServiceFabricClusterDeploymentDataType + "." + targetName + ".deployment_targets[0].id " +
			": " + octopusdeployAzureServiceFabricClusterDeploymentResourceType + "." + targetName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployAzureServiceFabricClusterDeploymentResourceType + "." + targetName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployAzureServiceFabricClusterDeploymentResourceType + "." + targetName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		passwordLookup := "${var." + targetName + "}"

		terraformResource := terraform.TerraformAzureServiceFabricClusterDeploymentTarget{
			Type:                            octopusdeployAzureServiceFabricClusterDeploymentResourceType,
			Name:                            targetName,
			Environments:                    c.lookupEnvironments(target.EnvironmentIds, dependencies),
			ResourceName:                    target.Name,
			Roles:                           target.Roles,
			ConnectionEndpoint:              target.Endpoint.ConnectionEndpoint,
			AadClientCredentialSecret:       &target.Endpoint.AadClientCredentialSecret,
			AadCredentialType:               &target.Endpoint.AadCredentialType,
			AadUserCredentialPassword:       &passwordLookup,
			AadUserCredentialUsername:       &target.Endpoint.AadUserCredentialUsername,
			CertificateStoreLocation:        &target.Endpoint.CertificateStoreLocation,
			CertificateStoreName:            &target.Endpoint.CertificateStoreName,
			ClientCertificateVariable:       &target.Endpoint.ClientCertVariable,
			HealthStatus:                    &target.HealthStatus,
			IsDisabled:                      &target.IsDisabled,
			MachinePolicyId:                 c.getMachinePolicy(target.MachinePolicyId, dependencies),
			OperatingSystem:                 nil,
			SecurityMode:                    nil,
			ServerCertificateThumbprint:     nil,
			ShellName:                       &target.ShellName,
			ShellVersion:                    &target.ShellVersion,
			SpaceId:                         nil,
			Status:                          nil,
			StatusSummary:                   nil,
			TenantTags:                      c.Excluder.FilteredTenantTags(target.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			TenantedDeploymentParticipation: &target.TenantedDeploymentParticipation,
			Tenants:                         dependencies.GetResources("Tenants", target.TenantIds...),
			Thumbprint:                      &target.Thumbprint,
			Uri:                             nil,
			Endpoint:                        nil,
		}
		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, target, targetName)
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployAzureServiceFabricClusterDeploymentDataType + "." + targetName + ".deployment_targets) != 0 ? 0 : 1}")
		}

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), target.Name, octopusdeployAzureServiceFabricClusterDeploymentResourceType, targetName))

		targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, targetBlock, dependencies, recursive)
		if err != nil {
			return "", err
		}

		// When using dummy values, we expect the secrets will be updated later
		if c.DummySecretVariableValues || stateless {

			ignoreAll := terraform.EmptyBlock{}
			lifecycleBlock := gohcl.EncodeAsBlock(ignoreAll, "lifecycle")
			targetBlock.Body().AppendBlock(lifecycleBlock)

			if c.DummySecretVariableValues {
				hcl.WriteUnquotedAttribute(lifecycleBlock, "ignore_changes", "[aad_user_credential_password]")
			}

			if stateless {
				hcl.WriteUnquotedAttribute(lifecycleBlock, "prevent_destroy", "true")
			}
		}

		file.Body().AppendBlock(targetBlock)

		secretVariableResource := terraform.TerraformVariable{
			Name:        targetName,
			Type:        "string",
			Nullable:    true,
			Sensitive:   true,
			Description: "The aad_user_credential_password value associated with the target \"" + target.Name + "\"",
		}

		if c.DummySecretVariableValues {
			secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
		}

		block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
		hcl.WriteUnquotedAttribute(block, "type", "string")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)

	return nil
}

func (c AzureServiceFabricTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c AzureServiceFabricTargetConverter) lookupEnvironments(envs []string, dependencies *data.ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return newEnvs
}

func (c AzureServiceFabricTargetConverter) getMachinePolicy(machine string, dependencies *data.ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("MachinePolicies", machine)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c AzureServiceFabricTargetConverter) getWorkerPool(pool string, dependencies *data.ResourceDetailsCollection) *string {
	if len(pool) == 0 {
		return nil
	}

	machineLookup := dependencies.GetResource("WorkerPools", pool)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c AzureServiceFabricTargetConverter) exportDependencies(target octopus.AzureServiceFabricResource, dependencies *data.ResourceDetailsCollection) error {

	// The machine policies need to be exported
	err := c.MachinePolicyConverter.ToHclById(target.MachinePolicyId, dependencies)

	if err != nil {
		return err
	}

	// Export the environments
	for _, e := range target.EnvironmentIds {
		err = c.EnvironmentConverter.ToHclById(e, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}
