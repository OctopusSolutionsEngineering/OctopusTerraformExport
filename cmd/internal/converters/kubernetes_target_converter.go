package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
)

type KubernetesTargetConverter struct {
	Client                 client.OctopusClient
	MachinePolicyConverter ConverterById
	AccountConverter       ConverterById
	CertificateConverter   ConverterById
	EnvironmentConverter   ConverterById
	ExcludeAllTargets      bool
	ExcludeTargets         args.ExcludeTargets
	ExcludeTargetsRegex    args.ExcludeTargets
	ExcludeTargetsExcept   args.ExcludeTargets
	ExcludeTenantTags      args.ExcludeTenantTags
	ExcludeTenantTagSets   args.ExcludeTenantTagSets
	Excluder               ExcludeByName
	TagSetConverter        TagSetConverter
}

func (c KubernetesTargetConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.KubernetesEndpointResource]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Kubernetes Target: " + resource.Id)
		err = c.toHcl(resource, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c KubernetesTargetConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.KubernetesEndpointResource{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	zap.L().Info("Kubernetes Target: " + resource.Id)
	return c.toHcl(resource, true, dependencies)
}

func (c KubernetesTargetConverter) ToHclLookupById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.Machine{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsExcept, c.ExcludeTargetsRegex) {
		return nil
	}

	if resource.Endpoint.CommunicationStyle != "Kubernetes" {
		return nil
	}

	thisResource := ResourceDetails{}

	resourceName := "target_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data.octopusdeploy_deployment_targets." + resourceName + ".deployment_targets[0].id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := terraform.TerraformDeploymentTargetsData{
			Type:        "octopusdeploy_deployment_targets",
			Name:        resourceName,
			Ids:         nil,
			PartialName: &resource.Name,
			Skip:        0,
			Take:        1,
		}
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve a deployment target called \""+resource.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.deployment_targets) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c KubernetesTargetConverter) toHcl(target octopus.KubernetesEndpointResource, recursive bool, dependencies *ResourceDetailsCollection) error {
	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(target.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsExcept, c.ExcludeTargetsRegex) {
		return nil
	}

	if target.Endpoint.CommunicationStyle == "Kubernetes" {
		if recursive {
			err := c.exportDependencies(target, dependencies)

			if err != nil {
				return err
			}
		}

		targetName := "target_" + sanitizer.SanitizeName(target.Name)

		thisResource := ResourceDetails{}
		thisResource.FileName = "space_population/" + targetName + ".tf"
		thisResource.Id = target.Id
		thisResource.ResourceType = c.GetResourceType()
		thisResource.Lookup = "${octopusdeploy_kubernetes_cluster_deployment_target." + targetName + ".id}"
		thisResource.ToHcl = func() (string, error) {

			terraformResource := terraform.TerraformKubernetesEndpointResource{
				Type:                            "octopusdeploy_kubernetes_cluster_deployment_target",
				Name:                            targetName,
				ClusterUrl:                      strutil.EmptyIfNil(target.Endpoint.ClusterUrl),
				Environments:                    c.lookupEnvironments(target.EnvironmentIds, dependencies),
				ResourceName:                    target.Name,
				Roles:                           target.Roles,
				ClusterCertificate:              dependencies.GetResourcePointer("Certificates", target.Endpoint.ClusterCertificate),
				ClusterCertificatePath:          target.Endpoint.ClusterCertificatePath,
				DefaultWorkerPoolId:             c.getWorkerPool(target.Endpoint.DefaultWorkerPoolId, dependencies),
				HealthStatus:                    nil,
				Id:                              nil,
				IsDisabled:                      strutil.NilIfFalse(target.IsDisabled),
				MachinePolicyId:                 c.getMachinePolicy(target.MachinePolicyId, dependencies),
				Namespace:                       strutil.NilIfEmptyPointer(target.Endpoint.Namespace),
				OperatingSystem:                 nil,
				ProxyId:                         nil,
				RunningInContainer:              nil,
				ShellName:                       nil,
				ShellVersion:                    nil,
				SkipTlsVerification:             strutil.ParseBoolPointer(target.Endpoint.SkipTlsVerification),
				SpaceId:                         nil,
				Status:                          nil,
				StatusSummary:                   nil,
				TenantTags:                      c.Excluder.FilteredTenantTags(target.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
				TenantedDeploymentParticipation: target.TenantedDeploymentParticipation,
				Tenants:                         dependencies.GetResources("Tenants", target.TenantIds...),
				Thumbprint:                      nil,
				Uri:                             target.Uri,
				Endpoint: terraform.TerraformKubernetesEndpoint{
					CommunicationStyle: "Kubernetes",
				},
				Container: terraform.TerraformKubernetesContainer{
					FeedId: target.Endpoint.Container.FeedId,
					Image:  target.Endpoint.Container.Image,
				},
				Authentication:                      c.getK8sAuth(&target, dependencies),
				AwsAccountAuthentication:            c.getAwsAuth(&target, dependencies),
				AzureServicePrincipalAuthentication: c.getAzureAuth(&target, dependencies),
				CertificateAuthentication:           c.getCertAuth(&target, dependencies),
				GcpAccountAuthentication:            c.getGoogleAuth(&target, dependencies),
				PodAuthentication:                   c.getPodAuth(&target),
			}
			file := hclwrite.NewEmptyFile()

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + target.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_kubernetes_cluster_deployment_target." + targetName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")
			err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, targetBlock, dependencies, recursive)
			if err != nil {
				return "", err
			}
			file.Body().AppendBlock(targetBlock)

			return string(file.Bytes()), nil
		}

		dependencies.AddResource(thisResource)
	}

	return nil
}

func (c KubernetesTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c KubernetesTargetConverter) getAwsAuth(target *octopus.KubernetesEndpointResource, dependencies *ResourceDetailsCollection) *terraform.TerraformAwsAccountAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesAws" {
		return &terraform.TerraformAwsAccountAuthentication{
			AccountId:                 c.getAccount(target.Endpoint.Authentication.AccountId, dependencies),
			ClusterName:               strutil.EmptyIfNil(target.Endpoint.Authentication.ClusterName),
			AssumeRole:                target.Endpoint.Authentication.AssumeRole,
			AssumeRoleExternalId:      target.Endpoint.Authentication.AssumeRoleExternalId,
			AssumeRoleSessionDuration: target.Endpoint.Authentication.AssumeRoleSessionDurationSeconds,
			AssumedRoleArn:            target.Endpoint.Authentication.AssumedRoleArn,
			AssumedRoleSession:        target.Endpoint.Authentication.AssumedRoleSession,
			UseInstanceRole:           target.Endpoint.Authentication.UseInstanceRole,
		}
	}

	return nil
}

func (c KubernetesTargetConverter) getK8sAuth(target *octopus.KubernetesEndpointResource, dependencies *ResourceDetailsCollection) *terraform.TerraformAccountAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesStandard" {
		return &terraform.TerraformAccountAuthentication{
			AccountId: c.getAccount(target.Endpoint.Authentication.AccountId, dependencies),
		}
	}

	return nil
}

func (c KubernetesTargetConverter) getGoogleAuth(target *octopus.KubernetesEndpointResource, dependencies *ResourceDetailsCollection) *terraform.TerraformGcpAccountAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesGoogleCloud" {
		return &terraform.TerraformGcpAccountAuthentication{
			AccountId:                 c.getAccount(target.Endpoint.Authentication.AccountId, dependencies),
			ClusterName:               strutil.EmptyIfNil(target.Endpoint.Authentication.ClusterName),
			Project:                   strutil.EmptyIfNil(target.Endpoint.Authentication.Project),
			ImpersonateServiceAccount: target.Endpoint.Authentication.ImpersonateServiceAccount,
			Region:                    target.Endpoint.Authentication.Region,
			ServiceAccountEmails:      target.Endpoint.Authentication.ServiceAccountEmails,
			Zone:                      target.Endpoint.Authentication.Zone,
			UseVmServiceAccount:       target.Endpoint.Authentication.UseVmServiceAccount,
		}
	}

	return nil
}

func (c KubernetesTargetConverter) getCertAuth(target *octopus.KubernetesEndpointResource, dependencies *ResourceDetailsCollection) *terraform.TerraformCertificateAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesCertificate" {
		return &terraform.TerraformCertificateAuthentication{
			ClientCertificate: dependencies.GetResourcePointer("Certificates", target.Endpoint.Authentication.ClientCertificate),
		}
	}

	return nil
}

func (c KubernetesTargetConverter) getAzureAuth(target *octopus.KubernetesEndpointResource, dependencies *ResourceDetailsCollection) *terraform.TerraformAzureServicePrincipalAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesAzure" {
		return &terraform.TerraformAzureServicePrincipalAuthentication{
			AccountId:            c.getAccount(target.Endpoint.Authentication.AccountId, dependencies),
			ClusterName:          strutil.EmptyIfNil(target.Endpoint.Authentication.ClusterName),
			ClusterResourceGroup: strutil.EmptyIfNil(target.Endpoint.Authentication.ClusterResourceGroup),
		}
	}

	return nil
}

func (c KubernetesTargetConverter) getPodAuth(target *octopus.KubernetesEndpointResource) *terraform.TerraformPodAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesPodService" {
		return &terraform.TerraformPodAuthentication{
			TokenPath: strutil.EmptyIfNil(target.Endpoint.Authentication.TokenPath),
		}
	}

	return nil
}

func (c KubernetesTargetConverter) lookupEnvironments(envs []string, dependencies *ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return newEnvs
}

func (c KubernetesTargetConverter) getMachinePolicy(machine string, dependencies *ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("MachinePolicies", machine)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c KubernetesTargetConverter) getAccount(account *string, dependencies *ResourceDetailsCollection) string {
	if account == nil {
		return ""
	}

	accountLookup := dependencies.GetResource("Accounts", *account)
	if accountLookup == "" {
		return ""
	}

	return accountLookup
}

func (c KubernetesTargetConverter) getWorkerPool(pool *string, dependencies *ResourceDetailsCollection) *string {
	if pool == nil {
		return nil
	}

	workerPoolLookup := dependencies.GetResource("WorkerPools", *pool)
	if workerPoolLookup == "" {
		return nil
	}

	return &workerPoolLookup
}

func (c KubernetesTargetConverter) exportDependencies(target octopus.KubernetesEndpointResource, dependencies *ResourceDetailsCollection) error {
	// The machine policies need to be exported
	err := c.MachinePolicyConverter.ToHclById(target.MachinePolicyId, dependencies)

	if err != nil {
		return err
	}

	// Export the accounts
	if target.Endpoint.Authentication.AccountId != nil {
		err = c.AccountConverter.ToHclById(*target.Endpoint.Authentication.AccountId, dependencies)

		if err != nil {
			return err
		}
	}

	// Export the certificate
	if target.Endpoint.Authentication.ClientCertificate != nil {
		err = c.CertificateConverter.ToHclById(*target.Endpoint.Authentication.ClientCertificate, dependencies)

		if err != nil {
			return err
		}
	}

	if target.Endpoint.ClusterCertificate != nil {
		err = c.CertificateConverter.ToHclById(*target.Endpoint.ClusterCertificate, dependencies)

		if err != nil {
			return err
		}
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
