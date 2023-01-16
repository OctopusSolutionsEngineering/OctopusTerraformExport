package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type KubernetesTargetConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
	MachinePolicyMap  map[string]string
	AccountMap        map[string]string
	EnvironmentMap    map[string]string
}

func (c KubernetesTargetConverter) ToHcl() (map[string]string, map[string]string, error) {
	collection := octopus.GeneralCollection[octopus.KubernetesEndpointResource]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, nil, err
	}

	results := map[string]string{}
	resultsMap := map[string]string{}

	for _, target := range collection.Items {
		if target.Endpoint.CommunicationStyle == "Kubernetes" {
			targetName := "target_" + util.SanitizeName(target.Name)

			terraformResource := terraform.TerraformKubernetesEndpointResource{
				Type:                            "octopusdeploy_kubernetes_cluster_deployment_target",
				Name:                            targetName,
				ClusterUrl:                      util.EmptyIfNil(target.Endpoint.ClusterUrl),
				Environments:                    c.lookupEnvironments(target.EnvironmentIds),
				ResourceName:                    target.Name,
				Roles:                           target.Roles,
				ClusterCertificate:              target.Endpoint.ClusterCertificate,
				DefaultWorkerPoolId:             nil,
				HealthStatus:                    nil,
				Id:                              nil,
				IsDisabled:                      util.NilIfFalse(target.IsDisabled),
				MachinePolicyId:                 c.getMachinePolicy(target.MachinePolicyId),
				Namespace:                       util.NilIfEmptyPointer(target.Endpoint.Namespace),
				OperatingSystem:                 nil,
				ProxyId:                         nil,
				RunningInContainer:              nil,
				ShellName:                       nil,
				ShellVersion:                    nil,
				SkipTlsVerification:             util.ParseBoolPointer(target.Endpoint.SkipTlsVerification),
				SpaceId:                         nil,
				Status:                          nil,
				StatusSummary:                   nil,
				TenantTags:                      target.TenantTags,
				TenantedDeploymentParticipation: target.TenantedDeploymentParticipation,
				Tenants:                         target.TenantIds,
				Thumbprint:                      nil,
				Uri:                             target.Uri,
				Endpoint: terraform.TerraformKubernetesEndpoint{
					CommunicationStyle:  "Kubernetes",
					ClusterCertificate:  target.Endpoint.ClusterCertificate,
					ClusterUrl:          target.Endpoint.ClusterUrl,
					Namespace:           target.Endpoint.Namespace,
					SkipTlsVerification: util.ParseBoolPointer(target.Endpoint.SkipTlsVerification),
					DefaultWorkerPoolId: target.Endpoint.DefaultWorkerPoolId,
				},
				Container: terraform.TerraformKubernetesContainer{
					FeedId: target.Endpoint.Container.FeedId,
					Image:  target.Endpoint.Container.Image,
				},
				Authentication:                      c.getK8sAuth(&target),
				AwsAccountAuthentication:            c.getAwsAuth(&target),
				AzureServicePrincipalAuthentication: c.getAzureAuth(&target),
				CertificateAuthentication:           c.getCertAuth(&target),
				GcpAccountAuthentication:            c.getGoogleAuth(&target),
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			results["space_population/target_"+targetName+".tf"] = string(file.Bytes())
			resultsMap[target.Id] = "${octopusdeploy_kubernetes_cluster_deployment_target." + targetName + ".id}"
		}
	}

	return results, resultsMap, nil
}

func (c KubernetesTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c KubernetesTargetConverter) getAwsAuth(target *octopus.KubernetesEndpointResource) *terraform.TerraformAwsAccountAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesAws" {
		return &terraform.TerraformAwsAccountAuthentication{
			AccountId:                 c.getAccount(target.Endpoint.Authentication.AccountId),
			ClusterName:               util.EmptyIfNil(target.Endpoint.Authentication.ClusterName),
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

func (c KubernetesTargetConverter) getK8sAuth(target *octopus.KubernetesEndpointResource) *terraform.TerraformAccountAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesStandard" {
		return &terraform.TerraformAccountAuthentication{
			AccountId: c.getAccount(target.Endpoint.Authentication.AccountId),
		}
	}

	return nil
}

func (c KubernetesTargetConverter) getGoogleAuth(target *octopus.KubernetesEndpointResource) *terraform.TerraformGcpAccountAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesGoogleCloud" {
		return &terraform.TerraformGcpAccountAuthentication{
			AccountId:                 c.getAccount(target.Endpoint.Authentication.AccountId),
			ClusterName:               util.EmptyIfNil(target.Endpoint.Authentication.ClusterName),
			Project:                   util.EmptyIfNil(target.Endpoint.Authentication.Project),
			ImpersonateServiceAccount: target.Endpoint.Authentication.ImpersonateServiceAccount,
			Region:                    target.Endpoint.Authentication.Region,
			ServiceAccountEmails:      target.Endpoint.Authentication.ServiceAccountEmails,
			Zone:                      target.Endpoint.Authentication.Zone,
			UseVmServiceAccount:       target.Endpoint.Authentication.UseVmServiceAccount,
		}
	}

	return nil
}

func (c KubernetesTargetConverter) getCertAuth(target *octopus.KubernetesEndpointResource) *terraform.TerraformCertificateAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesCertificate" {
		return &terraform.TerraformCertificateAuthentication{
			ClientCertificate: util.EmptyIfNil(target.Endpoint.Authentication.ClientCertificate),
		}
	}

	return nil
}

func (c KubernetesTargetConverter) getAzureAuth(target *octopus.KubernetesEndpointResource) *terraform.TerraformAzureServicePrincipalAuthentication {
	if target.Endpoint.Authentication.AuthenticationType == "KubernetesAzure" {
		return &terraform.TerraformAzureServicePrincipalAuthentication{
			AccountId:            c.getAccount(target.Endpoint.Authentication.AccountId),
			ClusterName:          util.EmptyIfNil(target.Endpoint.Authentication.ClusterName),
			ClusterResourceGroup: util.EmptyIfNil(target.Endpoint.Authentication.ClusterResourceGroup),
		}
	}

	return nil
}

func (c KubernetesTargetConverter) lookupEnvironments(envs []string) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = c.EnvironmentMap[v]
	}
	return newEnvs
}

func (c KubernetesTargetConverter) getAccount(accountPointer *string) string {
	account := util.EmptyIfNil(accountPointer)
	accountLookup, ok := c.AccountMap[account]
	if !ok {
		return ""
	}

	return accountLookup
}

func (c KubernetesTargetConverter) getMachinePolicy(machine string) *string {
	machineLookup, ok := c.MachinePolicyMap[machine]
	if !ok {
		return nil
	}

	return &machineLookup
}
