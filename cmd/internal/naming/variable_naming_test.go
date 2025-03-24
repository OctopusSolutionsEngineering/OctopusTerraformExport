package naming

import (
	"testing"

	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
)

func TestVariableSecretName(t *testing.T) {
	variable := octopus.Variable{Id: "test-id"}
	expected := "variable_6cc41d5ec590ab78cccecf81ef167d418c309a4598e8e45fef78039f7d9aa9fe_sensitive_value" // Replace with the actual expected hash
	result := VariableSecretName(variable)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestVariableValueName(t *testing.T) {
	variable := octopus.Variable{Id: "test-id"}
	expected := "variable_6cc41d5ec590ab78cccecf81ef167d418c309a4598e8e45fef78039f7d9aa9fe_value" // Replace with the actual expected hash
	result := VariableValueName(variable)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestTenantVariableValueName(t *testing.T) {
	tenantVariable := octopus.TenantVariable{Id: "test-id"}
	expected := "tenantvariable_6cc41d5ec590ab78cccecf81ef167d418c309a4598e8e45fef78039f7d9aa9fe_value"
	result := TenantVariableValueName(tenantVariable)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestTenantVariableSecretName(t *testing.T) {
	tenantVariable := octopus.TenantVariable{Id: "test-id"}
	expected := "tenantvariable_6cc41d5ec590ab78cccecf81ef167d418c309a4598e8e45fef78039f7d9aa9fe_sensitive_value"
	result := TenantVariableSecretName(tenantVariable)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestDeploymentProcessPropertySecretName(t *testing.T) {
	named := octopus.NameId{Id: "named-id"}
	action := octopus.Action{Id: "action-id"}
	property := "property"
	expected := "action_4120b0e641f4cf206433a3a2a69e1dbc960f1e424bac355a1346ba3382b61ca1_sensitive_value"
	result := DeploymentProcessPropertySecretName(named, action, property)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestGitCredentialSecretName(t *testing.T) {
	gitCredentials := octopus.GitCredentials{Name: "test-name"}
	expected := "gitcredential_test_name_sensitive_value"
	result := GitCredentialSecretName(gitCredentials)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestCertificateDataName(t *testing.T) {
	certificate := octopus.Certificate{Name: "test-cert"}
	expected := "certificate_test_cert_data"
	result := CertificateDataName(certificate)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestCertificatePasswordName(t *testing.T) {
	certificate := octopus.Certificate{Name: "test-cert"}
	expected := "certificate_test_cert_password"
	result := CertificatePasswordName(certificate)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestFeedSecretName(t *testing.T) {
	resource := octopus.Feed{Name: "test-feed"}
	expected := "feed_test_feed_password"
	result := FeedSecretName(resource)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestFeedSecretKeyName(t *testing.T) {
	resource := octopus.Feed{Name: "test-feed"}
	expected := "feed_test_feed_secretkey"
	result := FeedSecretKeyName(resource)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestStepTemplateParameterSecretName(t *testing.T) {
	template := octopus.StepTemplate{Id: "template-id"}
	parameter := octopus.StepTemplateParameters{Id: "parameter-id"}
	expected := "steptemplate_3835f375eb06473148f5fe0f15db6fca934fcd35ff7a87d6f113b6a4ae4e3764_sensitive_value"
	result := StepTemplateParameterSecretName(template, parameter)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestMachineProxyPassword(t *testing.T) {
	machine := octopus.MachineProxy{Name: "test-machine"}
	expected := "machine_proxy_test_machine_password"
	result := MachineProxyPassword(machine)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}
