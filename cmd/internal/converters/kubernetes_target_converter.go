package converters

import (
	"errors"
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
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
	"golang.org/x/sync/errgroup"
	"strings"
)

const octopusdeployKubernetesClusterDeploymentTargetDataType = "octopusdeploy_deployment_targets"
const octopusdeployKubernetesClusterDeploymentTargetResourceType = "octopusdeploy_kubernetes_cluster_deployment_target"

type KubernetesTargetConverter struct {
	TargetConverter

	MachinePolicyConverter   ConverterWithStatelessById
	AccountConverter         ConverterAndLookupWithStatelessById
	EnvironmentConverter     ConverterAndLookupWithStatelessById
	CertificateConverter     ConverterAndLookupWithStatelessById
	ExcludeAllTargets        bool
	ExcludeTargets           args.StringSliceArgs
	ExcludeTargetsRegex      args.StringSliceArgs
	ExcludeTargetsExcept     args.StringSliceArgs
	ExcludeTenantTags        args.StringSliceArgs
	ExcludeTenantTagSets     args.StringSliceArgs
	TagSetConverter          ConvertToHclByResource[octopus.TagSet]
	ErrGroup                 *errgroup.Group
	LimitResourceCount       int
	IncludeSpaceInPopulation bool
	IncludeIds               bool
}

func (c KubernetesTargetConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c KubernetesTargetConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c KubernetesTargetConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllTargets {
		return nil
	}

	collection := octopus.GeneralCollection[octopus.KubernetesEndpointResource]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	var filterErrors error
	targets := lo.Filter(collection.Items, func(item octopus.KubernetesEndpointResource, index int) bool {

		err, noEnvironments := c.HasNoEnvironments(item)

		if err != nil {
			filterErrors = errors.Join(filterErrors, err)
			return false
		}

		if noEnvironments {
			return false
		}

		return c.isKubernetesTarget(item)
	})

	if filterErrors != nil {
		return filterErrors
	}

	for _, resource := range targets {
		zap.L().Info("Kubernetes Target: " + resource.Id)
		err = c.toHcl(resource, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c KubernetesTargetConverter) isKubernetesTarget(resource octopus.KubernetesEndpointResource) bool {
	return resource.Endpoint.CommunicationStyle == "Kubernetes"
}

func (c KubernetesTargetConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c KubernetesTargetConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c KubernetesTargetConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
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

	if !c.isKubernetesTarget(resource) {
		return nil
	}

	err, noEnvironments := c.HasNoEnvironments(resource)

	if err != nil {
		return err
	}

	if noEnvironments {
		return nil
	}

	zap.L().Info("Kubernetes Target: " + resource.Id)
	return c.toHcl(resource, true, stateless, dependencies)
}

func (c KubernetesTargetConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
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

	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsRegex, c.ExcludeTargetsExcept) {
		return nil
	}

	if !c.isKubernetesTarget(resource) {
		return nil
	}

	err, noEnvironments := c.HasNoEnvironments(resource)

	if err != nil {
		return err
	}

	if noEnvironments {
		return nil
	}

	thisResource := data.ResourceDetails{}

	resourceName := "target_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.Name = resource.Name
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data." + octopusdeployKubernetesClusterDeploymentTargetDataType + "." + resourceName + ".deployment_targets[0].id}"
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

func (c KubernetesTargetConverter) buildData(resourceName string, resource octopus.KubernetesEndpointResource) terraform.TerraformDeploymentTargetsData {
	return terraform.TerraformDeploymentTargetsData{
		Type:        octopusdeployKubernetesClusterDeploymentTargetDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: &resource.Name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c KubernetesTargetConverter) writeData(file *hclwrite.File, resource octopus.KubernetesEndpointResource, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c KubernetesTargetConverter) toHcl(target octopus.KubernetesEndpointResource, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(target.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsRegex, c.ExcludeTargetsExcept) {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + target.Id)
		return nil
	}

	if !c.isKubernetesTarget(target) {
		return nil
	}

	err, noEnvironments := c.HasNoEnvironments(target)

	if err != nil {
		return err
	}

	if noEnvironments {
		return nil
	}

	if recursive {
		if stateless {
			if err := c.exportStatelessDependencies(target, dependencies); err != nil {
				return err
			}
		} else {
			if err := c.exportDependencies(target, dependencies); err != nil {
				return err
			}
		}
	}

	targetName := "target_" + sanitizer.SanitizeName(target.Name)

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + targetName + ".tf"
	thisResource.Id = target.Id
	thisResource.Name = target.Name
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = c.getLookup(stateless, targetName)
	thisResource.Dependency = c.getDependency(stateless, targetName)

	thisResource.ToHcl = func() (string, error) {

		// don't lookup empty certificate values
		var clusterCertificate *string = nil
		if len(strutil.EmptyIfNil(target.Endpoint.ClusterCertificate)) != 0 {
			clusterCertificate = dependencies.GetResourcePointer("Certificates", target.Endpoint.ClusterCertificate)
		}

		terraformResource := terraform.TerraformKubernetesEndpointResource{
			Type:                            octopusdeployKubernetesClusterDeploymentTargetResourceType,
			Name:                            targetName,
			Id:                              strutil.InputPointerIfEnabled(c.IncludeIds, &target.Id),
			SpaceId:                         strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", target.SpaceId)),
			ClusterUrl:                      strutil.EmptyIfNil(target.Endpoint.ClusterUrl),
			Environments:                    c.lookupEnvironments(target.EnvironmentIds, dependencies),
			ResourceName:                    target.Name,
			Roles:                           target.Roles,
			ClusterCertificate:              clusterCertificate,
			ClusterCertificatePath:          target.Endpoint.ClusterCertificatePath,
			DefaultWorkerPoolId:             c.getWorkerPool(target.Endpoint.DefaultWorkerPoolId, dependencies),
			HealthStatus:                    nil,
			IsDisabled:                      strutil.NilIfFalse(target.IsDisabled),
			MachinePolicyId:                 c.getMachinePolicy(target.MachinePolicyId, dependencies),
			Namespace:                       strutil.NilIfEmptyPointer(target.Endpoint.Namespace),
			OperatingSystem:                 nil,
			ProxyId:                         nil,
			RunningInContainer:              nil,
			ShellName:                       nil,
			ShellVersion:                    nil,
			SkipTlsVerification:             strutil.ParseBoolPointer(target.Endpoint.SkipTlsVerification),
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

		if stateless {
			c.writeData(file, target, targetName)
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployKubernetesClusterDeploymentTargetDataType + "." + targetName + ".deployment_targets) != 0 ? 0 : 1}")
		}

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDestroyAttribute(block)
		}

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, block, dependencies, recursive)
		if err != nil {
			return "", err
		}
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)

	return nil
}

func (c KubernetesTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c KubernetesTargetConverter) getAwsAuth(target *octopus.KubernetesEndpointResource, dependencies *data.ResourceDetailsCollection) *terraform.TerraformAwsAccountAuthentication {
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

func (c KubernetesTargetConverter) getK8sAuth(target *octopus.KubernetesEndpointResource, dependencies *data.ResourceDetailsCollection) *terraform.TerraformAccountAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesStandard" {
		return &terraform.TerraformAccountAuthentication{
			AccountId: c.getAccount(target.Endpoint.Authentication.AccountId, dependencies),
		}
	}

	return nil
}

func (c KubernetesTargetConverter) getGoogleAuth(target *octopus.KubernetesEndpointResource, dependencies *data.ResourceDetailsCollection) *terraform.TerraformGcpAccountAuthentication {
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

func (c KubernetesTargetConverter) getCertAuth(target *octopus.KubernetesEndpointResource, dependencies *data.ResourceDetailsCollection) *terraform.TerraformCertificateAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesCertificate" {
		if len(strutil.EmptyIfNil(target.Endpoint.Authentication.ClientCertificate)) == 0 {
			return nil
		}

		return &terraform.TerraformCertificateAuthentication{
			ClientCertificate: dependencies.GetResourcePointer("Certificates", target.Endpoint.Authentication.ClientCertificate),
		}
	}

	return nil
}

func (c KubernetesTargetConverter) getAzureAuth(target *octopus.KubernetesEndpointResource, dependencies *data.ResourceDetailsCollection) *terraform.TerraformAzureServicePrincipalAuthentication {
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

func (c KubernetesTargetConverter) lookupEnvironments(envs []string, dependencies *data.ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return lo.Filter(newEnvs, func(item string, index int) bool {
		return strings.TrimSpace(item) != ""
	})
}

func (c KubernetesTargetConverter) getMachinePolicy(machine string, dependencies *data.ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("MachinePolicies", machine)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c KubernetesTargetConverter) getAccount(account *string, dependencies *data.ResourceDetailsCollection) string {
	if account == nil {
		return ""
	}

	accountLookup := dependencies.GetResource("Accounts", *account)
	if accountLookup == "" {
		return ""
	}

	return accountLookup
}

func (c KubernetesTargetConverter) getWorkerPool(pool *string, dependencies *data.ResourceDetailsCollection) *string {
	if len(strutil.EmptyIfNil(pool)) == 0 {
		return nil
	}

	workerPoolLookup := dependencies.GetResource("WorkerPools", *pool)
	if workerPoolLookup == "" {
		return nil
	}

	return &workerPoolLookup
}

func (c KubernetesTargetConverter) exportDependencies(target octopus.KubernetesEndpointResource, dependencies *data.ResourceDetailsCollection) error {
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

func (c KubernetesTargetConverter) exportStatelessDependencies(target octopus.KubernetesEndpointResource, dependencies *data.ResourceDetailsCollection) error {
	// The machine policies need to be exported
	err := c.MachinePolicyConverter.ToHclStatelessById(target.MachinePolicyId, dependencies)

	if err != nil {
		return err
	}

	// Export the accounts
	if target.Endpoint.Authentication.AccountId != nil {
		err = c.AccountConverter.ToHclStatelessById(*target.Endpoint.Authentication.AccountId, dependencies)

		if err != nil {
			return err
		}
	}

	// Export the certificate
	if target.Endpoint.Authentication.ClientCertificate != nil {
		err = c.CertificateConverter.ToHclStatelessById(*target.Endpoint.Authentication.ClientCertificate, dependencies)

		if err != nil {
			return err
		}
	}

	if target.Endpoint.ClusterCertificate != nil {
		err = c.CertificateConverter.ToHclStatelessById(*target.Endpoint.ClusterCertificate, dependencies)

		if err != nil {
			return err
		}
	}

	// Export the environments
	for _, e := range target.EnvironmentIds {
		err = c.EnvironmentConverter.ToHclStatelessById(e, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c *KubernetesTargetConverter) getLookup(stateless bool, targetName string) string {
	if stateless {
		return "${length(data." + octopusdeployKubernetesClusterDeploymentTargetDataType + "." + targetName + ".deployment_targets) != 0 " +
			"? data." + octopusdeployKubernetesClusterDeploymentTargetDataType + "." + targetName + ".deployment_targets[0].id " +
			": " + octopusdeployKubernetesClusterDeploymentTargetResourceType + "." + targetName + "[0].id}"
	}
	return "${" + octopusdeployKubernetesClusterDeploymentTargetResourceType + "." + targetName + ".id}"
}

func (c *KubernetesTargetConverter) getDependency(stateless bool, targetName string) string {
	if stateless {
		return "${" + octopusdeployKubernetesClusterDeploymentTargetResourceType + "." + targetName + "}"
	}
	return ""
}
