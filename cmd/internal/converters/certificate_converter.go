package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	octopus2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
)

type CertificateConverter struct {
	Client                    client.OctopusClient
	DummySecretVariableValues bool
	DummySecretGenerator      DummySecretGenerator
	ExcludeTenantTags         args.ExcludeTenantTags
	ExcludeTenantTagSets      args.ExcludeTenantTagSets
	Excluder                  ExcludeByName
	TagSetConverter           TagSetConverter
}

func (c CertificateConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.Certificate]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Certificate: " + resource.Id)
		err = c.toHcl(resource, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c CertificateConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus2.Certificate{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	zap.L().Info("Certificate: " + resource.Id)
	return c.toHcl(resource, true, dependencies)
}

func (c CertificateConverter) ToHclLookupById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	certificate := octopus2.Certificate{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &certificate)

	if err != nil {
		return err
	}

	thisResource := ResourceDetails{}

	certificateName := "certificate_" + sanitizer.SanitizeName(certificate.Name)

	thisResource.FileName = "space_population/" + certificateName + ".tf"
	thisResource.Id = certificate.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data.octopusdeploy_certificates." + certificateName + ".certificates[0].id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := terraform2.TerraformCertificateData{
			Type:        "octopusdeploy_certificates",
			Name:        certificateName,
			Ids:         nil,
			PartialName: &certificate.Name,
			Skip:        0,
			Take:        1,
		}
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve a certificate called \""+certificate.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.certificates) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c CertificateConverter) toHcl(certificate octopus2.Certificate, recursive bool, dependencies *ResourceDetailsCollection) error {
	/*
		Note we don't export the tenants or environments that this certificate might be exposed to.
		It is assumed the exported project links up all required environments, and the certificate
		will link itself to any available environments or tenants.
	*/

	certificateName := "certificate_" + sanitizer.SanitizeName(certificate.Name)

	thisResource := ResourceDetails{}
	thisResource.FileName = "space_population/" + certificateName + ".tf"
	thisResource.Id = certificate.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_certificate." + certificateName + ".id}"
	thisResource.ToHcl = func() (string, error) {

		file := hclwrite.NewEmptyFile()

		err := c.writeMainResource(file, certificateName, certificate, recursive, dependencies)

		if err != nil {
			return "", err
		}

		err = c.writeVariables(file, certificateName, certificate)

		if err != nil {
			return "", err
		}

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c CertificateConverter) writeVariables(file *hclwrite.File, certificateName string, certificate octopus2.Certificate) error {

	defaultPassword := ""
	certificatePassword := terraform2.TerraformVariable{
		Name:        certificateName + "_password",
		Type:        "string",
		Nullable:    true,
		Sensitive:   true,
		Description: "The password used by the certificate " + certificate.Name,
		Default:     &defaultPassword,
	}

	if c.DummySecretVariableValues {
		certificatePassword.Default = c.DummySecretGenerator.GetDummyCertificatePassword()
	}

	block := gohcl.EncodeAsBlock(certificatePassword, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)

	certificateData := terraform2.TerraformVariable{
		Name:        certificateName + "_data",
		Type:        "string",
		Nullable:    false,
		Sensitive:   true,
		Description: "The certificate data used by the certificate " + certificate.Name,
	}

	if c.DummySecretVariableValues {
		certificateData.Default = c.DummySecretGenerator.GetDummyCertificate()
	}

	block = gohcl.EncodeAsBlock(certificateData, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)

	return nil
}

func (c CertificateConverter) writeMainResource(file *hclwrite.File, certificateName string, certificate octopus2.Certificate, recursive bool, dependencies *ResourceDetailsCollection) error {
	terraformResource := terraform2.TerraformCertificate{
		Type:            "octopusdeploy_certificate",
		Name:            certificateName,
		SpaceId:         nil,
		ResourceName:    certificate.Name,
		Password:        "${var." + certificateName + "_password}",
		CertificateData: "${var." + certificateName + "_data}",
		Archived:        &certificate.Archived,
		//CertificateDataFormat:           certificate.CertificateDataFormat,
		Environments: c.lookupEnvironments(certificate.EnvironmentIds, dependencies),
		//HasPrivateKey:                   certificate.HasPrivateKey,
		//IsExpired:                       certificate.IsExpired,
		//IssuerCommonName:                certificate.IssuerCommonName,
		//IssuerDistinguishedName:         certificate.IssuerDistinguishedName,
		//IssuerOrganization:              certificate.IssuerOrganization,
		//NotAfter:                        certificate.NotAfter,
		//NotBefore:                       certificate.NotBefore,
		Notes: &certificate.Notes,
		//ReplacedBy:                      nil, // ReplacedBy does not seem to be used
		//SelfSigned:                      certificate.SelfSigned,
		//SerialNumber:                    certificate.SerialNumber,
		//SignatureAlgorithmName:          certificate.SignatureAlgorithmName,
		//SubjectAlternativeNames:         certificate.SubjectAlternativeNames,
		//SubjectCommonName:               certificate.SubjectCommonName,
		//SubjectDistinguishedName:        certificate.SubjectDistinguishedName,
		//SubjectOrganization:             certificate.SubjectOrganization,
		TenantTags:                      c.Excluder.FilteredTenantTags(certificate.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
		TenantedDeploymentParticipation: &certificate.TenantedDeploymentParticipation,
		Tenants:                         c.lookupTenants(certificate.TenantIds, dependencies),
		//Thumbprint:                      certificate.Thumbprint,
		//Version:                         certificate.Version,
	}

	// Add a comment with the import command
	baseUrl, _ := c.Client.GetSpaceBaseUrl()
	file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), certificate.Name, "octopusdeploy_certificate", certificateName))

	targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")
	err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, targetBlock, dependencies, recursive)

	// When using dummy values, we expect the secrets will be updated later
	if c.DummySecretVariableValues {
		hcl.WriteLifecycleAttribute(targetBlock, "[password, certificate_data]")
	}

	if err != nil {
		return err
	}

	file.Body().AppendBlock(targetBlock)

	return nil
}

func (c CertificateConverter) GetResourceType() string {
	return "Certificates"
}

func (c CertificateConverter) lookupEnvironments(envs []string, dependencies *ResourceDetailsCollection) []string {
	newEnvs := make([]string, 0)
	for _, v := range envs {
		environment := dependencies.GetResource("Environments", v)
		if environment != "" {
			newEnvs = append(newEnvs, environment)
		}
	}
	return newEnvs
}

func (c CertificateConverter) lookupTenants(tenants []string, dependencies *ResourceDetailsCollection) []string {
	newTenants := make([]string, 0)
	for _, v := range tenants {
		tenant := dependencies.GetResource("Tenants", v)
		if tenant != "" {
			newTenants = append(newTenants, tenant)
		}
	}
	return newTenants
}
