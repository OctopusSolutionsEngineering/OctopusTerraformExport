package enhancedconverter

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type CertificateConverter struct {
	Client client.OctopusClient
}

func (c CertificateConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.Certificate]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		err = c.toHcl(resource, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c CertificateConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	certificate := octopus.Certificate{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &certificate)

	if err != nil {
		return err
	}

	return c.toHcl(certificate, dependencies)
}

func (c CertificateConverter) toHcl(certificate octopus.Certificate, dependencies *ResourceDetailsCollection) error {
	/*
		Note we don't export the tenants or environments that this certificate might be exposed to.
		It is assumed the exported project links up all required environments, and the certificate
		will link itself to any available environments or tenants.
	*/

	certificateName := "certificate_" + util.SanitizeName(certificate.Name)

	thisResource := ResourceDetails{}
	thisResource.FileName = "space_population/" + certificateName + ".tf"
	thisResource.Id = certificate.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_certificate." + certificateName + ".id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := terraform.TerraformCertificate{
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
			TenantTags:                      &certificate.TenantTags,
			TenantedDeploymentParticipation: &certificate.TenantedDeploymentParticipation,
			Tenants:                         c.lookupTenants(certificate.TenantIds, dependencies),
			//Thumbprint:                      certificate.Thumbprint,
			//Version:                         certificate.Version,
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		defaultPassword := ""
		certificatePassword := terraform.TerraformVariable{
			Name:        certificateName + "_password",
			Type:        "string",
			Nullable:    true,
			Sensitive:   true,
			Description: "The password used by the certificate " + certificate.Name,
			Default:     &defaultPassword,
		}

		block := gohcl.EncodeAsBlock(certificatePassword, "variable")
		util.WriteUnquotedAttribute(block, "type", "string")
		file.Body().AppendBlock(block)

		certificateData := terraform.TerraformVariable{
			Name:        certificateName + "_data",
			Type:        "string",
			Nullable:    false,
			Sensitive:   true,
			Description: "The certificate data used by the certificate " + certificate.Name,
		}

		block = gohcl.EncodeAsBlock(certificateData, "variable")
		util.WriteUnquotedAttribute(block, "type", "string")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
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

func (c CertificateConverter) lookupTenants(envs []string, dependencies *ResourceDetailsCollection) []string {
	newTenants := make([]string, 0)
	for _, v := range envs {
		tenant := dependencies.GetResource("Tenants", v)
		if tenant != "" {
			newTenants = append(newTenants, tenant)
		}
	}
	return newTenants
}
