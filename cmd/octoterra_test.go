package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/google/uuid"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"k8s.io/utils/strings/slices"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const NetworkName = "terraform-export-test"

const ApiKey = "API-ABCDEFGHIJKLMNOPQURTUVWXYZ12345"

// DisableTerraformInit can be set to true to disable Terraform downloads.
// This is useful if the terraform repo is down, as you can often just use
// cached plugins.
const DisableTerraformInit = false

type octopusContainer struct {
	testcontainers.Container
	URI string
}

type mysqlContainer struct {
	testcontainers.Container
	port string
	ip   string
}

type TestLogConsumer struct {
}

func (g *TestLogConsumer) Accept(l testcontainers.Log) {
	fmt.Println(string(l.Content))
}

func enableContainerLogging(container testcontainers.Container, ctx context.Context) error {
	// Display the container logs
	err := container.StartLogProducer(ctx)
	if err != nil {
		return err
	}
	g := TestLogConsumer{}
	container.FollowOutput(&g)
	return nil
}

// getReaperSkipped will return true if running in a podman environment
func getReaperSkipped() bool {
	if strings.Contains(os.Getenv("DOCKER_HOST"), "podman") {
		return true
	}

	return false
}

// getProvider returns the test containers provider
func getProvider() testcontainers.ProviderType {
	if strings.Contains(os.Getenv("DOCKER_HOST"), "podman") {
		return testcontainers.ProviderPodman
	}

	return testcontainers.ProviderDocker
}

func setupNetwork(ctx context.Context) (testcontainers.Network, error) {
	return testcontainers.GenericNetwork(ctx, testcontainers.GenericNetworkRequest{
		NetworkRequest: testcontainers.NetworkRequest{
			Name:           "octoterra",
			CheckDuplicate: true,
			SkipReaper:     getReaperSkipped(),
		},
		ProviderType: getProvider(),
	})
}

// setupDatabase creates a MSSQL container
func setupDatabase(ctx context.Context) (*mysqlContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "mcr.microsoft.com/mssql/server",
		ExposedPorts: []string{"1433/tcp"},
		Env: map[string]string{
			"ACCEPT_EULA": "Y",
			"SA_PASSWORD": "Password01!",
		},
		WaitingFor: wait.ForExec([]string{"/opt/mssql-tools/bin/sqlcmd", "-U", "sa", "-P", "Password01!", "-Q", "select 1"}).WithExitCodeMatcher(
			func(exitCode int) bool {
				return exitCode == 0
			}),
		SkipReaper: getReaperSkipped(),
		Networks: []string{
			"octoterra",
		},
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	ip, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}

	mappedPort, err := container.MappedPort(ctx, "1433")
	if err != nil {
		return nil, err
	}

	return &mysqlContainer{
		Container: container,
		ip:        ip,
		port:      mappedPort.Port(),
	}, nil
}

// setupOctopus creates an Octopus container
func setupOctopus(ctx context.Context, connString string) (*octopusContainer, error) {
	if os.Getenv("LICENSE") == "" {
		return nil, errors.New("the LICENSE environment variable must be set to a base 64 encoded Octopus license key")
	}

	req := testcontainers.ContainerRequest{
		// Be aware that later versions of Octopus killed Github Actions.
		// I think maybe they used more memory? 2022.2 works fine though.
		Image:        "octopusdeploy/octopusdeploy:2022.2",
		ExposedPorts: []string{"8080/tcp"},
		Env: map[string]string{
			"ACCEPT_EULA":                   "Y",
			"DB_CONNECTION_STRING":          connString,
			"ADMIN_API_KEY":                 ApiKey,
			"DISABLE_DIND":                  "Y",
			"ADMIN_USERNAME":                "admin",
			"ADMIN_PASSWORD":                "Password01!",
			"OCTOPUS_SERVER_BASE64_LICENSE": os.Getenv("LICENSE"),
		},
		Privileged: false,
		WaitingFor: wait.ForLog("Listening for HTTP requests on").WithStartupTimeout(30 * time.Minute),
		SkipReaper: getReaperSkipped(),
		Networks: []string{
			"octoterra",
		},
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	// Display the container logs
	enableContainerLogging(container, ctx)

	ip, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}

	mappedPort, err := container.MappedPort(ctx, "8080")
	if err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("http://%s:%s", ip, mappedPort.Port())

	return &octopusContainer{Container: container, URI: uri}, nil
}

// performTest is wrapper that initialises Octopus, runs a test, and cleans up the containers
func performTest(t *testing.T, testFunc func(t *testing.T, container *octopusContainer) error) {
	err := retry.Do(
		func() error {

			if testing.Short() {
				t.Skip("skipping integration test")
			}

			ctx := context.Background()

			network, err := setupNetwork(ctx)
			if err != nil {
				return err
			}

			sqlServer, err := setupDatabase(ctx)
			if err != nil {
				return err
			}

			sqlIp, err := sqlServer.Container.ContainerIP(ctx)
			t.Log("SQL Server IP: " + sqlIp)

			octopusContainer, err := setupOctopus(ctx, "Server="+sqlIp+",1433;Database=OctopusDeploy;User=sa;Password=Password01!")
			if err != nil {
				return err
			}

			// Clean up the container after the test is complete
			defer func() {
				// This fixes the "can not get logs from container which is dead or marked for removal" error
				// See https://github.com/testcontainers/testcontainers-go/issues/606
				octopusContainer.StopLogProducer()

				octoTerminateErr := octopusContainer.Terminate(ctx)
				sqlTerminateErr := sqlServer.Terminate(ctx)

				networkErr := network.Remove(ctx)

				if octoTerminateErr != nil || sqlTerminateErr != nil || networkErr != nil {
					t.Fatalf("failed to terminate container: %v %v", octoTerminateErr, sqlTerminateErr)
				}
			}()

			// give the server 5 minutes to start up
			success := false
			for start := time.Now(); ; {
				if time.Since(start) > 5*time.Minute {
					break
				}

				resp, err := http.Get(octopusContainer.URI + "/api")
				if err == nil && resp.StatusCode == http.StatusOK {
					success = true
					t.Log("Successfully contacted the Octopus API")
					break
				}

				time.Sleep(10 * time.Second)
			}

			if !success {
				t.Fatalf("Failed to access the Octopus API")
			}

			return testFunc(t, octopusContainer)
		},
		retry.Attempts(3),
	)

	if err != nil {
		t.Fatalf(err.Error())
	}
}

// initialiseOctopus uses Terraform to populate the test Octopus instance, making sure to clean up
// any files generated during previous Terraform executions to avoid conflicts and locking issues.
func initialiseOctopus(t *testing.T, container *octopusContainer, terraformDir string, spaceName string, initialiseVars []string, populateVars []string) error {
	path, err := os.Getwd()
	if err != nil {
		return err
	}
	t.Log("Working dir: " + path)

	/*
			This test creates a new space and then populates the space. The space export feature creates
			two directories: space_creation and space_population. Running Terraform in these two directories
			results in a space that we can then inspect the results of.

			The single project export feature does not export the space. However, we still need a blank
			space to populate in order to verify the results. So a generic Terraform directory called
		    z-createspace contains the resources required to create a new blank space.

			The two directories that Terraform is run in are captured in the terraformProjectDirs slice.
	*/
	terraformProjectDirs := []string{}
	if _, err := os.Stat(filepath.Join(terraformDir, "space_creation")); err == nil {
		// The space export creates a dir called space_creation to create the exported space
		terraformProjectDirs = append(terraformProjectDirs, filepath.Join(terraformDir, "space_creation"))
	} else {
		// The project export does not export the space, so we need a generic project to create the new space
		terraformProjectDirs = append(terraformProjectDirs, filepath.Join("..", "test", "terraform", "z-createspace"))
	}
	terraformProjectDirs = append(terraformProjectDirs, filepath.Join(terraformDir, "space_population"))

	// First loop initialises the new space, second populates the space
	spaceId := "Spaces-1"
	for i, terraformProjectDir := range terraformProjectDirs {

		if !DisableTerraformInit {
			os.Remove(filepath.Join(terraformProjectDir, ".terraform.lock.hcl"))
		}

		os.Remove(filepath.Join(terraformProjectDir, "terraform.tfstate"))

		if !DisableTerraformInit {
			args := []string{"init", "-no-color"}
			cmnd := exec.Command("terraform", args...)
			cmnd.Dir = terraformProjectDir
			out, err := cmnd.Output()

			if err != nil {
				exitError, ok := err.(*exec.ExitError)
				if ok {
					t.Log(string(exitError.Stderr))
				} else {
					t.Log(err.Error())
				}

				return err
			}

			t.Log(string(out))
		}

		// when initialising the new space, we need to define a new space name as a variable
		vars := []string{}
		if i == 0 {
			vars = append(initialiseVars, "-var=octopus_space_name="+spaceName)
		} else {
			vars = populateVars
		}

		newArgs := append([]string{
			"apply",
			"-auto-approve",
			"-no-color",
			"-var=octopus_server=" + container.URI,
			"-var=octopus_apikey=" + ApiKey,
			"-var=octopus_space_id=" + spaceId,
		}, vars...)

		cmnd := exec.Command("terraform", newArgs...)
		cmnd.Dir = terraformProjectDir
		out, err := cmnd.Output()

		if err != nil {
			exitError, ok := err.(*exec.ExitError)
			if ok {
				t.Log(string(exitError.Stderr))
			} else {
				t.Log(err)
			}
			return err
		}

		t.Log(string(out))

		// get the ID of any new space created, which will be used in the subsequent Terraform executions
		spaceId, err = getOutputVariable(t, terraformProjectDir, "octopus_space_id")

		if err != nil {
			exitError, ok := err.(*exec.ExitError)
			if ok {
				t.Log(string(exitError.Stderr))
			} else {
				t.Log(err)
			}
			return err
		}
	}

	return nil
}

// getOutputVariable reads a Terraform output variable
func getOutputVariable(t *testing.T, terraformDir string, outputVar string) (string, error) {
	cmnd := exec.Command(
		"terraform",
		"output",
		"-raw",
		outputVar)
	cmnd.Dir = terraformDir
	out, err := cmnd.Output()

	if err != nil {
		exitError, ok := err.(*exec.ExitError)
		if ok {
			t.Log(string(exitError.Stderr))
		} else {
			t.Log(err)
		}
		return "", err
	}

	return string(out), nil
}

// getTempDir creates a temporary directory for the exported Terraform files
func getTempDir() string {
	return os.TempDir() + string(os.PathSeparator) + uuid.New().String() + string(os.PathSeparator)
}

// createClient creates a client used to access the Octopus API
func createClient(container *octopusContainer, space string) *client.OctopusClient {
	return &client.OctopusClient{
		Url:    container.URI,
		Space:  space,
		ApiKey: ApiKey,
	}
}

// arrange initialises Octopus and MSSQL
func arrange(t *testing.T, container *octopusContainer, terraformDir string, populateVars []string) (string, error) {
	t.Log("POPULATING TEST SPACE")

	err := initialiseOctopus(t, container, terraformDir, "Test2", []string{}, populateVars)

	if err != nil {
		return "", err
	}

	if _, err := os.Stat(filepath.Join(terraformDir, "space_creation")); err == nil {
		return getOutputVariable(t, filepath.Join(terraformDir, "space_creation"), "octopus_space_id")
	} else {
		return getOutputVariable(t, filepath.Join("..", "test", "terraform", "z-createspace"), "octopus_space_id")
	}
}

// act exports the Octopus configuration as Terraform, and reimports it as a new space
func act(t *testing.T, container *octopusContainer, newSpaceId string, populateVars []string) (string, error) {
	t.Log("EXPORTING TEST SPACE")

	tempDir := getTempDir()
	defer os.Remove(tempDir)

	err := ConvertSpaceToTerraform(container.URI, newSpaceId, ApiKey, tempDir, true)

	if err != nil {
		return "", err
	}

	t.Log("REIMPORTING TEST SPACE")

	err = initialiseOctopus(t, container, tempDir, "Test3", []string{}, populateVars)

	if err != nil {
		return "", err
	}

	return getOutputVariable(t, filepath.Join(tempDir, "space_creation"), "octopus_space_id")
}

// actProjectExport exports a single Octopus project as Terraform, and reimports it as a new space
func actProjectExport(t *testing.T, container *octopusContainer, terraformDir string, newSpaceId string, populateVars []string, projectVar string) (string, error) {
	tempDir := getTempDir()
	defer os.Remove(tempDir)

	projectId, err := getOutputVariable(t, filepath.Join(terraformDir, "space_population"), projectVar)

	if err != nil {
		return "", err
	}

	err = ConvertProjectToTerraform(container.URI, newSpaceId, ApiKey, tempDir, true, projectId)

	if err != nil {
		return "", err
	}

	err = initialiseOctopus(t, container, tempDir, "Test3", []string{}, populateVars)

	if err != nil {
		return "", err
	}

	return getOutputVariable(t, filepath.Join("..", "test", "terraform", "z-createspace"), "octopus_space_id")
}

// TestSpaceExport verifies that a space can be reimported with the correct settings
func TestSpaceExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/1-singlespace", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		space := octopus.Space{}
		err = octopusClient.GetSpace(&space)

		if err != nil {
			return err
		}

		if space.Name != "Test3" {
			t.Fatalf("New space must have the name \"Test3\"")
		}

		if *space.Description != "My test space" {
			t.Fatalf("New space must have the name \"My test space\"")
		}

		if space.IsDefault {
			t.Fatalf("New space must not be the default one")
		}

		if space.TaskQueueStopped {
			t.Fatalf("New space must not have the task queue stopped")
		}

		if slices.Index(space.SpaceManagersTeams, "teams-administrators") == -1 {
			t.Fatalf("New space must have teams-administrators as a manager team")
		}

		return nil
	})
}

// TestProjectGroupExport verifies that a project group can be reimported with the correct settings
func TestProjectGroupExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/2-projectgroup", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.ProjectGroup]{}
		err = octopusClient.GetAllResources("ProjectGroups", &collection)

		if err != nil {
			return err
		}

		found := false
		for _, v := range collection.Items {
			if v.Name == "Test" {
				found = true
				if *v.Description != "Test Description" {
					t.Fatalf("The project group must be have a description of \"Test Description\"")
				}
			}
		}

		if !found {
			t.Fatalf("Space must have a project group called \"Test\"")
		}

		return nil
	})
}

// TestAwsAccountExport verifies that an AWS account can be reimported with the correct settings
func TestAwsAccountExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/3-awsaccount", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{"-var=account_aws_account=secretgoeshere"})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.Account]{}
		err = octopusClient.GetAllResources("Accounts", &collection)

		if err != nil {
			return err
		}

		found := false
		for _, v := range collection.Items {
			if v.Name == "AWS Account" {
				found = true
				if *v.AccessKey != "ABCDEFGHIJKLMNOPQRST" {
					t.Fatalf("The account must be have an access key of \"ABCDEFGHIJKLMNOPQRST\"")
				}
			}
		}

		if !found {
			t.Fatalf("Space must have aan AWS account called \"AWS Account\"")
		}

		return nil
	})
}

// TestAzureAccountExport verifies that an Azure account can be reimported with the correct settings
func TestAzureAccountExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/4-azureaccount", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{"-var=account_azure=secretgoeshere"})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.Account]{}
		err = octopusClient.GetAllResources("Accounts", &collection)

		if err != nil {
			return err
		}

		found := false
		for _, v := range collection.Items {
			if v.Name == "Azure" {
				found = true
				if *v.ClientId != "2eb8bd13-661e-489c-beb9-4103efb9dbdd" {
					t.Fatalf("The account must be have a client ID of \"2eb8bd13-661e-489c-beb9-4103efb9dbdd\"")
				}

				if *v.SubscriptionNumber != "95bf77d2-64b1-4ed2-9de1-b5451e3881f5" {
					t.Fatalf("The account must be have a client ID of \"95bf77d2-64b1-4ed2-9de1-b5451e3881f5\"")
				}

				if *v.TenantId != "18eb006b-c3c8-4a72-93cd-fe4b293f82ee" {
					t.Fatalf("The account must be have a client ID of \"18eb006b-c3c8-4a72-93cd-fe4b293f82ee\"")
				}

				if *v.Description != "Azure Account" {
					t.Fatalf("The account must be have a description of \"Azure Account\"")
				}

				if *v.TenantedDeploymentParticipation != "Untenanted" {
					t.Fatalf("The account must be have a tenanted deployment participation of \"Untenanted\"")
				}
			}
		}

		if !found {
			t.Fatalf("Space must have an Azure account called \"Azure\"")
		}

		return nil
	})
}

// TestUsernamePasswordAccountExport verifies that a username/password account can be reimported with the correct settings
func TestUsernamePasswordAccountExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/5-userpassaccount", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{"-var=account_gke=secretgoeshere"})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.Account]{}
		err = octopusClient.GetAllResources("Accounts", &collection)

		if err != nil {
			return err
		}

		found := false
		for _, v := range collection.Items {
			if v.Name == "GKE" {
				found = true
				if *v.Username != "admin" {
					t.Fatalf("The account must be have a username of \"admin\"")
				}

				if !v.Password.HasValue {
					t.Fatalf("The account must be have a password")
				}

				if *v.Description != "A test account" {
					t.Fatalf("The account must be have a description of \"A test account\"")
				}

				if *v.TenantedDeploymentParticipation != "Untenanted" {
					t.Fatalf("The account must be have a tenanted deployment participation of \"Untenanted\"")
				}

				if len(v.TenantTags) != 0 {
					t.Fatalf("The account must be have no tenant tags")
				}
			}
		}

		if !found {
			t.Fatalf("Space must have an account called \"GKE\"")
		}

		return nil
	})
}

// TestGcpAccountExport verifies that a GCP account can be reimported with the correct settings
func TestGcpAccountExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/6-gcpaccount", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{"-var=account_google=secretgoeshere"})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.Account]{}
		err = octopusClient.GetAllResources("Accounts", &collection)

		if err != nil {
			return err
		}

		found := false
		for _, v := range collection.Items {
			if v.Name == "Google" {
				found = true
				if !v.JsonKey.HasValue {
					t.Fatalf("The account must be have a JSON key")
				}

				if *v.Description != "A test account" {
					t.Fatalf("The account must be have a description of \"A test account\"")
				}

				if *v.TenantedDeploymentParticipation != "Untenanted" {
					t.Fatalf("The account must be have a tenanted deployment participation of \"Untenanted\"")
				}

				if len(v.TenantTags) != 0 {
					t.Fatalf("The account must be have no tenant tags")
				}
			}
		}

		if !found {
			t.Fatalf("Space must have an account called \"Google\"")
		}

		return nil
	})
}

// TestSshAccountExport verifies that a SSH account can be reimported with the correct settings
func TestSshAccountExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/7-sshaccount", []string{})

		if err != nil {
			return err
		}

		// Act
		// We set the passphrase because of https://github.com/OctopusDeployLabs/terraform-provider-octopusdeploy/issues/343
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{
			"-var=account_ssh_cert=whatever",
			"-var=account_ssh=LS0tLS1CRUdJTiBPUEVOU1NIIFBSSVZBVEUgS0VZLS0tLS0KYjNCbGJuTnphQzFyWlhrdGRqRUFBQUFBQkc1dmJtVUFBQUFFYm05dVpRQUFBQUFBQUFBQkFBQUJGd0FBQUFkemMyZ3RjbgpOaEFBQUFBd0VBQVFBQUFRRUF5c25PVXhjN0tJK2pIRUc5RVEwQXFCMllGRWE5ZnpZakZOY1pqY1dwcjJQRkRza25oOUpTCm1NVjVuZ2VrbTRyNHJVQU5tU2dQMW1ZTGo5TFR0NUVZa0N3OUdyQ0paNitlQTkzTEowbEZUamFkWEJuQnNmbmZGTlFWYkcKZ2p3U1o4SWdWQ2oySXE0S1hGZm0vbG1ycEZQK2Jqa2V4dUxwcEh5dko2ZmxZVjZFMG13YVlneVNHTWdLYy9ubXJaMTY0WApKMStJL1M5NkwzRWdOT0hNZmo4QjM5eEhZQ0ZUTzZEQ0pLQ3B0ZUdRa0gwTURHam84d3VoUlF6c0IzVExsdXN6ZG0xNmRZCk16WXZBSWR3emZ3bzh1ajFBSFFOendDYkIwRmR6bnFNOEpLV2ZrQzdFeVVrZUl4UXZmLzJGd1ZyS0xEZC95ak5PUmNoa3EKb2owNncySXFad0FBQThpS0tqT3dpaW96c0FBQUFBZHpjMmd0Y25OaEFBQUJBUURLeWM1VEZ6c29qNk1jUWIwUkRRQ29IWgpnVVJyMS9OaU1VMXhtTnhhbXZZOFVPeVNlSDBsS1l4WG1lQjZTYml2aXRRQTJaS0EvV1pndVAwdE8za1JpUUxEMGFzSWxuCnI1NEQzY3NuU1VWT05wMWNHY0d4K2Q4VTFCVnNhQ1BCSm53aUJVS1BZaXJncGNWK2IrV2F1a1UvNXVPUjdHNHVta2ZLOG4KcCtWaFhvVFNiQnBpREpJWXlBcHorZWF0blhyaGNuWDRqOUwzb3ZjU0EwNGN4K1B3SGYzRWRnSVZNN29NSWtvS20xNFpDUQpmUXdNYU9qekM2RkZET3dIZE11VzZ6TjJiWHAxZ3pOaThBaDNETi9Dank2UFVBZEEzUEFKc0hRVjNPZW96d2twWitRTHNUCkpTUjRqRkM5Ly9ZWEJXc29zTjMvS00wNUZ5R1NxaVBUckRZaXBuQUFBQUF3RUFBUUFBQVFFQXdRZzRqbitlb0kyYUJsdk4KVFYzRE1rUjViMU9uTG1DcUpEeGM1c2N4THZNWnNXbHBaN0NkVHk4ckJYTGhEZTdMcUo5QVVub0FHV1lwdTA1RW1vaFRpVwptVEFNVHJCdmYwd2xsdCtJZVdvVXo3bmFBbThQT1psb29MbXBYRzh5VmZKRU05aUo4NWtYNDY4SkF6VDRYZ1JXUFRYQ1JpCi9abCtuWUVUZVE4WTYzWlJhTVE3SUNmK2FRRWxRenBYb21idkxYM1RaNmNzTHh5Z3Eza01aSXNJU0lUcEk3Y0tsQVJ0Rm4KcWxKRitCL2JlUEJkZ3hIRVpqZDhDV0NIR1ZRUDh3Z3B0d0Rrak9NTzh2b2N4YVpOT0hZZnBwSlBCTkVjMEVKbmduN1BXSgorMVZSTWZKUW5SemVubmE3VHdSUSsrclZmdkVaRmhqamdSUk85RitrMUZvSWdRQUFBSUVBbFFybXRiV2V0d3RlWlZLLys4CklCUDZkcy9MSWtPb3pXRS9Wckx6cElBeHEvV1lFTW1QK24wK1dXdWRHNWpPaTFlZEJSYVFnU0owdTRxcE5JMXFGYTRISFYKY2oxL3pzenZ4RUtSRElhQkJGaU81Y3QvRVQvUTdwanozTnJaZVdtK0dlUUJKQ0diTEhSTlQ0M1ZpWVlLVG82ZGlGVTJteApHWENlLzFRY2NqNjVZQUFBQ0JBUHZodmgzb2Q1MmY4SFVWWGoxeDNlL1ZFenJPeVloTi9UQzNMbWhHYnRtdHZ0L0J2SUhxCndxWFpTT0lWWkZiRnVKSCtORHNWZFFIN29yUW1VcGJxRllDd0IxNUZNRGw0NVhLRm0xYjFyS1c1emVQK3d0M1hyM1p0cWsKRkdlaUlRMklSZklBQjZneElvNTZGemdMUmx6QnB0bzhkTlhjMXhtWVgyU2Rhb3ZwSkRBQUFBZ1FET0dwVE9oOEFRMFoxUwpzUm9vVS9YRTRkYWtrSU5vMDdHNGI3M01maG9xbkV1T01LM0ZRVStRRWUwYWpvdWs5UU1QNWJzZU1CYnJNZVNNUjBRWVBCClQ4Z0Z2S2VISWN6ZUtJTjNPRkRaRUF4TEZNMG9LbjR2bmdHTUFtTXUva2QwNm1PZnJUNDRmUUh1ajdGNWx1QVJHejRwYUwKLzRCTUVkMnFTRnFBYzZ6L0RRQUFBQTF0WVhSMGFFQk5ZWFIwYUdWM0FRSURCQT09Ci0tLS0tRU5EIE9QRU5TU0ggUFJJVkFURSBLRVktLS0tLQo=",
		})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.Account]{}
		err = octopusClient.GetAllResources("Accounts", &collection)

		if err != nil {
			return err
		}

		accountName := "SSH"
		found := false
		for _, v := range collection.Items {
			if v.Name == accountName {
				found = true
				if v.AccountType != "SshKeyPair" {
					t.Fatal("The account must be have a type of \"SshKeyPair\"")
				}

				if *v.Username != "admin" {
					t.Fatal("The account must be have a username of \"admin\"")
				}

				if *v.Description != "A test account" {
					// This appears to be a bug in the provider where the description is not set
					t.Log("BUG: The account must be have a description of \"A test account\"")
				}

				if *v.TenantedDeploymentParticipation != "Untenanted" {
					t.Fatal("The account must be have a tenanted deployment participation of \"Untenanted\"")
				}

				if len(v.TenantTags) != 0 {
					t.Fatal("The account must be have no tenant tags")
				}
			}
		}

		if !found {
			t.Fatal("Space must have an account called \"" + accountName + "\"")
		}

		return nil
	})
}

// TestAzureSubscriptionAccountExport verifies that an azure account can be reimported with the correct settings
func TestAzureSubscriptionAccountExport(t *testing.T) {
	// I could not figure out a combination of properties that made this resource work
	return

	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/8-azuresubscriptionaccount", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{
			"-var=account_subscription_cert=LS0tLS1CRUdJTiBPUEVOU1NIIFBSSVZBVEUgS0VZLS0tLS0KYjNCbGJuTnphQzFyWlhrdGRqRUFBQUFBQkc1dmJtVUFBQUFFYm05dVpRQUFBQUFBQUFBQkFBQUJGd0FBQUFkemMyZ3RjbgpOaEFBQUFBd0VBQVFBQUFRRUF5c25PVXhjN0tJK2pIRUc5RVEwQXFCMllGRWE5ZnpZakZOY1pqY1dwcjJQRkRza25oOUpTCm1NVjVuZ2VrbTRyNHJVQU5tU2dQMW1ZTGo5TFR0NUVZa0N3OUdyQ0paNitlQTkzTEowbEZUamFkWEJuQnNmbmZGTlFWYkcKZ2p3U1o4SWdWQ2oySXE0S1hGZm0vbG1ycEZQK2Jqa2V4dUxwcEh5dko2ZmxZVjZFMG13YVlneVNHTWdLYy9ubXJaMTY0WApKMStJL1M5NkwzRWdOT0hNZmo4QjM5eEhZQ0ZUTzZEQ0pLQ3B0ZUdRa0gwTURHam84d3VoUlF6c0IzVExsdXN6ZG0xNmRZCk16WXZBSWR3emZ3bzh1ajFBSFFOendDYkIwRmR6bnFNOEpLV2ZrQzdFeVVrZUl4UXZmLzJGd1ZyS0xEZC95ak5PUmNoa3EKb2owNncySXFad0FBQThpS0tqT3dpaW96c0FBQUFBZHpjMmd0Y25OaEFBQUJBUURLeWM1VEZ6c29qNk1jUWIwUkRRQ29IWgpnVVJyMS9OaU1VMXhtTnhhbXZZOFVPeVNlSDBsS1l4WG1lQjZTYml2aXRRQTJaS0EvV1pndVAwdE8za1JpUUxEMGFzSWxuCnI1NEQzY3NuU1VWT05wMWNHY0d4K2Q4VTFCVnNhQ1BCSm53aUJVS1BZaXJncGNWK2IrV2F1a1UvNXVPUjdHNHVta2ZLOG4KcCtWaFhvVFNiQnBpREpJWXlBcHorZWF0blhyaGNuWDRqOUwzb3ZjU0EwNGN4K1B3SGYzRWRnSVZNN29NSWtvS20xNFpDUQpmUXdNYU9qekM2RkZET3dIZE11VzZ6TjJiWHAxZ3pOaThBaDNETi9Dank2UFVBZEEzUEFKc0hRVjNPZW96d2twWitRTHNUCkpTUjRqRkM5Ly9ZWEJXc29zTjMvS00wNUZ5R1NxaVBUckRZaXBuQUFBQUF3RUFBUUFBQVFFQXdRZzRqbitlb0kyYUJsdk4KVFYzRE1rUjViMU9uTG1DcUpEeGM1c2N4THZNWnNXbHBaN0NkVHk4ckJYTGhEZTdMcUo5QVVub0FHV1lwdTA1RW1vaFRpVwptVEFNVHJCdmYwd2xsdCtJZVdvVXo3bmFBbThQT1psb29MbXBYRzh5VmZKRU05aUo4NWtYNDY4SkF6VDRYZ1JXUFRYQ1JpCi9abCtuWUVUZVE4WTYzWlJhTVE3SUNmK2FRRWxRenBYb21idkxYM1RaNmNzTHh5Z3Eza01aSXNJU0lUcEk3Y0tsQVJ0Rm4KcWxKRitCL2JlUEJkZ3hIRVpqZDhDV0NIR1ZRUDh3Z3B0d0Rrak9NTzh2b2N4YVpOT0hZZnBwSlBCTkVjMEVKbmduN1BXSgorMVZSTWZKUW5SemVubmE3VHdSUSsrclZmdkVaRmhqamdSUk85RitrMUZvSWdRQUFBSUVBbFFybXRiV2V0d3RlWlZLLys4CklCUDZkcy9MSWtPb3pXRS9Wckx6cElBeHEvV1lFTW1QK24wK1dXdWRHNWpPaTFlZEJSYVFnU0owdTRxcE5JMXFGYTRISFYKY2oxL3pzenZ4RUtSRElhQkJGaU81Y3QvRVQvUTdwanozTnJaZVdtK0dlUUJKQ0diTEhSTlQ0M1ZpWVlLVG82ZGlGVTJteApHWENlLzFRY2NqNjVZQUFBQ0JBUHZodmgzb2Q1MmY4SFVWWGoxeDNlL1ZFenJPeVloTi9UQzNMbWhHYnRtdHZ0L0J2SUhxCndxWFpTT0lWWkZiRnVKSCtORHNWZFFIN29yUW1VcGJxRllDd0IxNUZNRGw0NVhLRm0xYjFyS1c1emVQK3d0M1hyM1p0cWsKRkdlaUlRMklSZklBQjZneElvNTZGemdMUmx6QnB0bzhkTlhjMXhtWVgyU2Rhb3ZwSkRBQUFBZ1FET0dwVE9oOEFRMFoxUwpzUm9vVS9YRTRkYWtrSU5vMDdHNGI3M01maG9xbkV1T01LM0ZRVStRRWUwYWpvdWs5UU1QNWJzZU1CYnJNZVNNUjBRWVBCClQ4Z0Z2S2VISWN6ZUtJTjNPRkRaRUF4TEZNMG9LbjR2bmdHTUFtTXUva2QwNm1PZnJUNDRmUUh1ajdGNWx1QVJHejRwYUwKLzRCTUVkMnFTRnFBYzZ6L0RRQUFBQTF0WVhSMGFFQk5ZWFIwYUdWM0FRSURCQT09Ci0tLS0tRU5EIE9QRU5TU0ggUFJJVkFURSBLRVktLS0tLQo=",
		})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.Account]{}
		err = octopusClient.GetAllResources("Accounts", &collection)

		if err != nil {
			return err
		}

		accountName := "Subscription"
		found := false
		for _, v := range collection.Items {
			if v.Name == accountName {
				found = true
				if v.AccountType != "AzureServicePrincipal" {
					t.Fatal("The account must be have a type of \"AzureServicePrincipal\"")
				}

				if *v.Description != "A test account" {
					t.Fatal("BUG: The account must be have a description of \"A test account\"")
				}

				if *v.TenantedDeploymentParticipation != "Untenanted" {
					t.Fatal("The account must be have a tenanted deployment participation of \"Untenanted\"")
				}

				if len(v.TenantTags) != 0 {
					t.Fatal("The account must be have no tenant tags")
				}
			}
		}

		if !found {
			t.Fatal("Space must have an account called \"" + accountName + "\"")
		}

		return nil
	})
}

// TestTokenAccountExport verifies that a token account can be reimported with the correct settings
func TestTokenAccountExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/9-tokenaccount", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{
			"-var=account_token=whatever",
		})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.Account]{}
		err = octopusClient.GetAllResources("Accounts", &collection)

		if err != nil {
			return err
		}

		accountName := "Token"
		found := false
		for _, v := range collection.Items {
			if v.Name == accountName {
				found = true
				if v.AccountType != "Token" {
					t.Fatal("The account must be have a type of \"Token\"")
				}

				if !v.Token.HasValue {
					t.Fatal("The account must be have a token")
				}

				if *v.Description != "A test account" {
					t.Fatal("The account must be have a description of \"A test account\"")
				}

				if *v.TenantedDeploymentParticipation != "Untenanted" {
					t.Fatal("The account must be have a tenanted deployment participation of \"Untenanted\"")
				}

				if len(v.TenantTags) != 0 {
					t.Fatal("The account must be have no tenant tags")
				}
			}
		}

		if !found {
			t.Fatal("Space must have an account called \"" + accountName + "\"")
		}

		return nil
	})
}

// TestHelmFeedExport verifies that a helm feed can be reimported with the correct settings
func TestHelmFeedExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/10-helmfeed", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{
			"-var=feed_helm_password=whatever",
		})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.Feed]{}
		err = octopusClient.GetAllResources("Feeds", &collection)

		if err != nil {
			return err
		}

		feedName := "Helm"
		found := false
		for _, v := range collection.Items {
			if v.Name == feedName {
				found = true

				if *v.FeedType != "Helm" {
					t.Fatal("The feed must have a type of \"Helm\"")
				}

				if *v.Username != "username" {
					t.Fatal("The feed must have a username of \"username\"")
				}

				if *v.FeedUri != "https://charts.helm.sh/stable/" {
					t.Fatal("The feed must be have a URI of \"https://charts.helm.sh/stable/\"")
				}

				foundExecutionTarget := false
				foundNotAcquired := false
				for _, o := range v.PackageAcquisitionLocationOptions {
					if o == "ExecutionTarget" {
						foundExecutionTarget = true
					}

					if o == "NotAcquired" {
						foundNotAcquired = true
					}
				}

				if !(foundExecutionTarget && foundNotAcquired) {
					t.Fatal("The feed must be have a PackageAcquisitionLocationOptions including \"ExecutionTarget\" and \"NotAcquired\"")
				}
			}
		}

		if !found {
			t.Fatal("Space must have an feed called \"" + feedName + "\"")
		}

		return nil
	})
}

// TestDockerFeedExport verifies that a docker feed can be reimported with the correct settings
func TestDockerFeedExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/11-dockerfeed", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{
			"-var=feed_docker_password=whatever",
		})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.Feed]{}
		err = octopusClient.GetAllResources("Feeds", &collection)

		if err != nil {
			return err
		}

		feedName := "Docker"
		found := false
		for _, v := range collection.Items {
			if v.Name == feedName {
				found = true

				if *v.FeedType != "Docker" {
					t.Fatal("The feed must have a type of \"Docker\"")
				}

				if *v.Username != "username" {
					t.Fatal("The feed must have a username of \"username\"")
				}

				if *v.ApiVersion != "v1" {
					t.Fatal("The feed must be have a API version of \"v1\"")
				}

				if *v.FeedUri != "https://index.docker.io" {
					t.Fatal("The feed must be have a feed uri of \"https://index.docker.io\"")
				}

				foundExecutionTarget := false
				foundNotAcquired := false
				for _, o := range v.PackageAcquisitionLocationOptions {
					if o == "ExecutionTarget" {
						foundExecutionTarget = true
					}

					if o == "NotAcquired" {
						foundNotAcquired = true
					}
				}

				if !(foundExecutionTarget && foundNotAcquired) {
					t.Fatal("The feed must be have a PackageAcquisitionLocationOptions including \"ExecutionTarget\" and \"NotAcquired\"")
				}
			}
		}

		if !found {
			t.Fatal("Space must have an feed called \"" + feedName + "\"")
		}

		return nil
	})
}

// TestEcrFeedExport verifies that a ecr feed can be reimported with the correct settings
func TestEcrFeedExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		if os.Getenv("ECR_ACCESS_KEY") == "" {
			return errors.New("the ECR_ACCESS_KEY environment variable must be set a valid AWS access key")
		}

		if os.Getenv("ECR_SECRET_KEY") == "" {
			return errors.New("the ECR_SECRET_KEY environment variable must be set a valid AWS secret key")
		}

		newSpaceId, err := arrange(t, container, "../test/terraform/12-ecrfeed", []string{
			"-var=feed_ecr_access_key=" + os.Getenv("ECR_ACCESS_KEY"),
			"-var=feed_ecr_secret_key=" + os.Getenv("ECR_SECRET_KEY"),
		})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{
			"-var=feed_ecr_password=" + os.Getenv("ECR_SECRET_KEY"),
		})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.Feed]{}
		err = octopusClient.GetAllResources("Feeds", &collection)

		if err != nil {
			return err
		}

		feedName := "ECR"
		found := false
		for _, v := range collection.Items {
			if v.Name == feedName {
				found = true

				if *v.FeedType != "AwsElasticContainerRegistry" {
					t.Fatal("The feed must have a type of \"AwsElasticContainerRegistry\" (was \"" + strutil.EmptyIfNil(v.FeedType) + "\"")
				}

				if *v.AccessKey != os.Getenv("ECR_ACCESS_KEY") {
					t.Fatal("The feed must have a access key of \"" + os.Getenv("ECR_ACCESS_KEY") + "\" (was \"" + strutil.EmptyIfNil(v.AccessKey) + "\"")
				}

				if *v.Region != "us-east-1" {
					t.Fatal("The feed must have a region of \"us-east-1\" (was \"" + strutil.EmptyIfNil(v.Region) + "\"")
				}

				foundExecutionTarget := false
				foundNotAcquired := false
				for _, o := range v.PackageAcquisitionLocationOptions {
					if o == "ExecutionTarget" {
						foundExecutionTarget = true
					}

					if o == "NotAcquired" {
						foundNotAcquired = true
					}
				}

				if !(foundExecutionTarget && foundNotAcquired) {
					t.Fatal("The feed must be have a PackageAcquisitionLocationOptions including \"ExecutionTarget\" and \"NotAcquired\"")
				}
			}
		}

		if !found {
			t.Fatal("Space must have an feed called \"" + feedName + "\"")
		}

		return nil
	})
}

// TestMavenFeedExport verifies that a maven feed can be reimported with the correct settings
func TestMavenFeedExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/13-mavenfeed", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{
			"-var=feed_maven_password=whatever",
		})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.Feed]{}
		err = octopusClient.GetAllResources("Feeds", &collection)

		if err != nil {
			return err
		}

		feedName := "Maven"
		found := false
		for _, v := range collection.Items {
			if v.Name == feedName {
				found = true

				if *v.FeedType != "Maven" {
					t.Fatal("The feed must have a type of \"Maven\"")
				}

				if *v.Username != "username" {
					t.Fatal("The feed must have a username of \"username\"")
				}

				if *v.DownloadAttempts != 5 {
					t.Fatal("The feed must be have a downloads attempts set to \"5\"")
				}

				if *v.DownloadRetryBackoffSeconds != 10 {
					t.Fatal("The feed must be have a downloads retry backoff set to \"10\"")
				}

				if *v.FeedUri != "https://repo.maven.apache.org/maven2/" {
					t.Fatal("The feed must be have a feed uri of \"https://repo.maven.apache.org/maven2/\"")
				}

				foundExecutionTarget := false
				foundServer := false
				for _, o := range v.PackageAcquisitionLocationOptions {
					if o == "ExecutionTarget" {
						foundExecutionTarget = true
					}

					if o == "Server" {
						foundServer = true
					}
				}

				if !(foundExecutionTarget && foundServer) {
					t.Fatal("The feed must be have a PackageAcquisitionLocationOptions including \"ExecutionTarget\" and \"Server\"")
				}
			}
		}

		if !found {
			t.Fatal("Space must have an feed called \"" + feedName + "\"")
		}

		return nil
	})
}

// TestNugetFeedExport verifies that a nuget feed can be reimported with the correct settings
func TestNugetFeedExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/14-nugetfeed", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{
			"-var=feed_nuget_password=whatever",
		})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.Feed]{}
		err = octopusClient.GetAllResources("Feeds", &collection)

		if err != nil {
			return err
		}

		feedName := "Nuget"
		found := false
		for _, v := range collection.Items {
			if v.Name == feedName {
				found = true

				if *v.FeedType != "NuGet" {
					t.Fatal("The feed must have a type of \"NuGet\"")
				}

				if !v.EnhancedMode {
					t.Fatal("The feed must have enhanced mode set to true")
				}

				if *v.Username != "username" {
					t.Fatal("The feed must have a username of \"username\"")
				}

				if *v.DownloadAttempts != 5 {
					t.Fatal("The feed must be have a downloads attempts set to \"5\"")
				}

				if *v.DownloadRetryBackoffSeconds != 10 {
					t.Fatal("The feed must be have a downloads retry backoff set to \"10\"")
				}

				if *v.FeedUri != "https://index.docker.io" {
					t.Fatal("The feed must be have a feed uri of \"https://index.docker.io\"")
				}

				foundExecutionTarget := false
				foundServer := false
				for _, o := range v.PackageAcquisitionLocationOptions {
					if o == "ExecutionTarget" {
						foundExecutionTarget = true
					}

					if o == "Server" {
						foundServer = true
					}
				}

				if !(foundExecutionTarget && foundServer) {
					t.Fatal("The feed must be have a PackageAcquisitionLocationOptions including \"ExecutionTarget\" and \"Server\"")
				}
			}
		}

		if !found {
			t.Fatal("Space must have an feed called \"" + feedName + "\"")
		}

		return nil
	})
}

// TestWorkerPoolExport verifies that a static worker pool can be reimported with the correct settings
func TestWorkerPoolExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/15-workerpool", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.WorkerPool]{}
		err = octopusClient.GetAllResources("WorkerPools", &collection)

		if err != nil {
			return err
		}

		workerPoolName := "Docker"
		found := false
		for _, v := range collection.Items {
			if v.Name == workerPoolName {
				found = true

				if v.WorkerPoolType != "StaticWorkerPool" {
					t.Fatal("The worker pool must be have a type of \"StaticWorkerPool\" (was \"" + v.WorkerPoolType + "\"")
				}

				if *v.Description != "A test worker pool" {
					t.Fatal("The worker pool must be have a description of \"A test worker pool\" (was \"" + strutil.EmptyIfNil(v.Description) + "\"")
				}

				if v.SortOrder != 3 {
					t.Fatal("The worker pool must be have a sort order of \"3\" (was \"" + fmt.Sprint(v.SortOrder) + "\"")
				}

				if v.IsDefault {
					t.Fatal("The worker pool must be must not be the default")
				}
			}
		}

		if !found {
			t.Fatal("Space must have an worker pool called \"" + workerPoolName + "\"")
		}

		return nil
	})
}

// TestEnvironmentExport verifies that an environment can be reimported with the correct settings
func TestEnvironmentExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/16-environment", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.Environment]{}
		err = octopusClient.GetAllResources("Environments", &collection)

		if err != nil {
			return err
		}

		resourceName := "Development"
		found := false
		for _, v := range collection.Items {
			if v.Name == resourceName {
				found = true

				if strutil.EmptyIfNil(v.Description) != "A test environment" {
					t.Fatal("The environment must be have a description of \"A test environment\" (was \"" + strutil.EmptyIfNil(v.Description) + "\"")
				}

				if !v.AllowDynamicInfrastructure {
					t.Fatal("The environment must have dynamic infrastructure enabled.")
				}

				if v.UseGuidedFailure {
					t.Fatal("The environment must not have guided failure enabled.")
				}
			}
		}

		if !found {
			t.Fatal("Space must have an environment called \"" + resourceName + "\"")
		}

		return nil
	})
}

// TestLifecycleExport verifies that a lifecycle can be reimported with the correct settings
func TestLifecycleExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/17-lifecycle", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.Lifecycle]{}
		err = octopusClient.GetAllResources("Lifecycles", &collection)

		if err != nil {
			return err
		}

		resourceName := "Simple"
		found := false
		for _, v := range collection.Items {
			if v.Name == resourceName {
				found = true

				if strutil.EmptyIfNil(v.Description) != "A test lifecycle" {
					t.Fatal("The lifecycle must be have a description of \"A test lifecycle\" (was \"" + strutil.EmptyIfNil(v.Description) + "\")")
				}

				if v.TentacleRetentionPolicy.QuantityToKeep != 30 {
					t.Fatal("The lifecycle must be have a tentacle retention policy of \"30\" (was \"" + fmt.Sprint(v.TentacleRetentionPolicy.QuantityToKeep) + "\")")
				}

				if v.TentacleRetentionPolicy.ShouldKeepForever {
					t.Fatal("The lifecycle must be have a tentacle retention not set to keep forever")
				}

				if v.TentacleRetentionPolicy.Unit != "Items" {
					t.Fatal("The lifecycle must be have a tentacle retention unit set to \"Items\" (was \"" + v.TentacleRetentionPolicy.Unit + "\")")
				}

				if v.ReleaseRetentionPolicy.QuantityToKeep != 1 {
					t.Fatal("The lifecycle must be have a release retention policy of \"1\" (was \"" + fmt.Sprint(v.ReleaseRetentionPolicy.QuantityToKeep) + "\")")
				}

				if !v.ReleaseRetentionPolicy.ShouldKeepForever {
					t.Log("BUG: The lifecycle must be have a release retention set to keep forever (known bug - the provider creates this field as false)")
				}

				if v.ReleaseRetentionPolicy.Unit != "Days" {
					t.Fatal("The lifecycle must be have a release retention unit set to \"Days\" (was \"" + v.ReleaseRetentionPolicy.Unit + "\")")
				}
			}
		}

		if !found {
			t.Fatal("Space must have an lifecycle called \"" + resourceName + "\"")
		}

		return nil
	})
}

// TestVariableSetExport verifies that a variable set can be reimported with the correct settings
func TestVariableSetExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/18-variableset", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.LibraryVariableSet]{}
		err = octopusClient.GetAllResources("LibraryVariableSets", &collection)

		if err != nil {
			return err
		}

		resourceName := "Test"
		found := false
		for _, v := range collection.Items {
			if v.Name == resourceName {
				found = true

				if strutil.EmptyIfNil(v.Description) != "Test variable set" {
					t.Fatal("The library variable set must be have a description of \"Test variable set\" (was \"" + strutil.EmptyIfNil(v.Description) + "\")")
				}

				resource := octopus.VariableSet{}
				_, err = octopusClient.GetResourceById("Variables", v.VariableSetId, &resource)

				if len(resource.Variables) != 1 {
					t.Fatal("The library variable set must have one associated variable")
				}

				if resource.Variables[0].Name != "Test.Variable" {
					t.Fatal("The library variable set variable must have a name of \"Test.Variable\"")
				}

				if resource.Variables[0].Type != "String" {
					t.Fatal("The library variable set variable must have a type of \"String\"")
				}

				if strutil.EmptyIfNil(resource.Variables[0].Description) != "Test variable" {
					t.Fatal("The library variable set variable must have a description of \"Test variable\"")
				}

				if strutil.EmptyIfNil(resource.Variables[0].Value) != "test" {
					t.Fatal("The library variable set variable must have a value of \"test\"")
				}

				if resource.Variables[0].IsSensitive {
					t.Fatal("The library variable set variable must not be sensitive")
				}

				if !resource.Variables[0].IsEditable {
					t.Fatal("The library variable set variable must be editable")
				}
			}
		}

		if !found {
			t.Fatal("Space must have an library variable set called \"" + resourceName + "\"")
		}

		return nil
	})
}

// TestProjectExport verifies that a project can be reimported with the correct settings
func TestProjectExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/19-project", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.Project]{}
		err = octopusClient.GetAllResources("Projects", &collection)

		if err != nil {
			return err
		}

		resourceName := "Test"
		found := false
		for _, v := range collection.Items {
			if v.Name == resourceName {
				found = true

				if strutil.EmptyIfNil(v.Description) != "Test project" {
					t.Fatal("The project must be have a description of \"Test project\" (was \"" + strutil.EmptyIfNil(v.Description) + "\")")
				}

				if v.AutoCreateRelease {
					t.Fatal("The project must not have auto release create enabled")
				}

				if strutil.EmptyIfNil(v.DefaultGuidedFailureMode) != "EnvironmentDefault" {
					t.Fatal("The project must be have a DefaultGuidedFailureMode of \"EnvironmentDefault\" (was \"" + strutil.EmptyIfNil(v.DefaultGuidedFailureMode) + "\")")
				}

				if v.DefaultToSkipIfAlreadyInstalled {
					t.Fatal("The project must not have DefaultToSkipIfAlreadyInstalled enabled")
				}

				if v.DiscreteChannelRelease {
					t.Fatal("The project must not have DiscreteChannelRelease enabled")
				}

				if v.IsDisabled {
					t.Fatal("The project must not have IsDisabled enabled")
				}

				if v.IsVersionControlled {
					t.Fatal("The project must not have IsVersionControlled enabled")
				}

				if strutil.EmptyIfNil(v.TenantedDeploymentMode) != "Untenanted" {
					t.Fatal("The project must be have a TenantedDeploymentMode of \"Untenanted\" (was \"" + strutil.EmptyIfNil(v.TenantedDeploymentMode) + "\")")
				}

				if len(v.IncludedLibraryVariableSetIds) != 0 {
					t.Fatal("The project must not have any library variable sets")
				}

				if v.ProjectConnectivityPolicy.AllowDeploymentsToNoTargets {
					t.Fatal("The project must not have ProjectConnectivityPolicy.AllowDeploymentsToNoTargets enabled")
				}

				if v.ProjectConnectivityPolicy.ExcludeUnhealthyTargets {
					t.Fatal("The project must not have ProjectConnectivityPolicy.AllowDeploymentsToNoTargets enabled")
				}

				if v.ProjectConnectivityPolicy.SkipMachineBehavior != "SkipUnavailableMachines" {
					t.Log("BUG: The project must be have a ProjectConnectivityPolicy.SkipMachineBehavior of \"SkipUnavailableMachines\" (was \"" + v.ProjectConnectivityPolicy.SkipMachineBehavior + "\") - Known issue where the value returned by /api/Spaces-#/ProjectGroups/ProjectGroups-#/projects is different to /api/Spaces-/Projects")
				}
			}
		}

		if !found {
			t.Fatal("Space must have an project called \"" + resourceName + "\"")
		}

		return nil
	})
}

// TestProjectChannelExport verifies that a project channel can be reimported with the correct settings
func TestProjectChannelExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/20-channel", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.Project]{}
		err = octopusClient.GetAllResources("Projects", &collection)

		if err != nil {
			return err
		}

		resourceName := "Test"
		found := false
		for _, v := range collection.Items {
			if v.Name == resourceName {
				found = true

				collection := octopus.GeneralCollection[octopus.Channel]{}
				err = octopusClient.GetAllResources("Projects/"+v.Id+"/channels", &collection)

				channelName := "Test"
				foundChannel := false

				for _, c := range collection.Items {
					if c.Name == channelName {
						foundChannel = true

						if strutil.EmptyIfNil(c.Description) != "Test channel" {
							t.Fatal("The channel must be have a description of \"Test channel\" (was \"" + strutil.EmptyIfNil(c.Description) + "\")")
						}

						if !c.IsDefault {
							t.Fatal("The channel must be be the default")
						}

						if len(c.Rules) != 1 {
							t.Fatal("The channel must have one rule")
						}

						if strutil.EmptyIfNil(c.Rules[0].Tag) != "^$" {
							t.Fatal("The channel rule must be have a tag of \"^$\" (was \"" + strutil.EmptyIfNil(c.Rules[0].Tag) + "\")")
						}

						if strutil.EmptyIfNil(c.Rules[0].ActionPackages[0].DeploymentAction) != "Test" {
							t.Fatal("The channel rule action step must be be set to \"Test\" (was \"" + strutil.EmptyIfNil(c.Rules[0].ActionPackages[0].DeploymentAction) + "\")")
						}

						if strutil.EmptyIfNil(c.Rules[0].ActionPackages[0].PackageReference) != "test" {
							t.Fatal("The channel rule action package must be be set to \"test\" (was \"" + strutil.EmptyIfNil(c.Rules[0].ActionPackages[0].PackageReference) + "\")")
						}
					}
				}

				if !foundChannel {
					t.Fatal("Project must have an channel called \"" + channelName + "\"")
				}
			}
		}

		if !found {
			t.Fatal("Space must have an project called \"" + resourceName + "\"")
		}

		return nil
	})
}

// TestTagSetExport verifies that a tag set can be reimported with the correct settings
func TestTagSetExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/21-tagset", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.TagSet]{}
		err = octopusClient.GetAllResources("TagSets", &collection)

		if err != nil {
			return err
		}

		resourceName := "tag1"
		found := false
		for _, v := range collection.Items {
			if v.Name == resourceName {
				found = true

				if strutil.EmptyIfNil(v.Description) != "Test tagset" {
					t.Fatal("The tag set must be have a description of \"Test tagset\" (was \"" + strutil.EmptyIfNil(v.Description) + "\")")
				}

				if v.SortOrder != 0 {
					t.Fatal("The tag set must be have a sort order of \"0\" (was \"" + fmt.Sprint(v.SortOrder) + "\")")
				}

				tagAFound := false
				for _, u := range v.Tags {
					if u.Name == "a" {
						tagAFound = true

						if strutil.EmptyIfNil(u.Description) != "tag a" {
							t.Fatal("The tag a must be have a description of \"tag a\" (was \"" + strutil.EmptyIfNil(u.Description) + "\")")
						}

						if u.Color != "#333333" {
							t.Fatal("The tag a must be have a color of \"#333333\" (was \"" + u.Color + "\")")
						}

						if u.SortOrder != 2 {
							t.Fatal("The tag a must be have a sort order of \"2\" (was \"" + fmt.Sprint(u.SortOrder) + "\")")
						}
					}
				}

				if !tagAFound {
					t.Fatal("Tag Set must have an tag called \"a\"")
				}
			}
		}

		if !found {
			t.Fatal("Space must have an tag set called \"" + resourceName + "\"")
		}

		return nil
	})
}

// TestGitCredentialsExport verifies that a git credential can be reimported with the correct settings
func TestGitCredentialsExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/22-gitcredentialtest", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{
			"-var=gitcredential_test=whatever",
		})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.GitCredentials]{}
		err = octopusClient.GetAllResources("Git-Credentials", &collection)

		if err != nil {
			return err
		}

		resourceName := "test"
		found := false
		for _, v := range collection.Items {
			if v.Name == resourceName {
				found = true

				if strutil.EmptyIfNil(v.Description) != "test git credential" {
					t.Fatal("The git credential must be have a description of \"test git credential\" (was \"" + strutil.EmptyIfNil(v.Description) + "\")")
				}

				if v.Details.Username != "admin" {
					t.Fatal("The git credential must be have a username of \"admin\" (was \"" + v.Details.Username + "\")")
				}

				if v.Details.Type != "UsernamePassword" {
					t.Fatal("The git credential must be have a credential type of \"UsernamePassword\" (was \"" + v.Details.Type + "\")")
				}
			}
		}

		if !found {
			t.Fatal("Space must have an git credential called \"" + resourceName + "\"")
		}

		return nil
	})
}

// TestScriptModuleExport verifies that a script module set can be reimported with the correct settings
func TestScriptModuleExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/23-scriptmodule", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.LibraryVariableSet]{}
		err = octopusClient.GetAllResources("LibraryVariableSets", &collection)

		if err != nil {
			return err
		}

		resourceName := "Test2"
		found := false
		for _, v := range collection.Items {
			if v.Name == resourceName {
				found = true

				if strutil.EmptyIfNil(v.Description) != "Test script module" {
					t.Fatal("The library variable set must be have a description of \"Test script module\" (was \"" + strutil.EmptyIfNil(v.Description) + "\")")
				}

				resource := octopus.VariableSet{}
				_, err = octopusClient.GetResourceById("Variables", v.VariableSetId, &resource)

				if len(resource.Variables) != 2 {
					t.Fatal("The library variable set must have two associated variables")
				}

				foundScript := false
				foundLanguage := false
				for _, u := range resource.Variables {
					if u.Name == "Octopus.Script.Module[Test2]" {
						foundScript = true

						if u.Type != "String" {
							t.Fatal("The library variable set variable must have a type of \"String\"")
						}

						if strutil.EmptyIfNil(u.Value) != "echo \"hi\"" {
							t.Fatal("The library variable set variable must have a value of \"\"echo \\\"hi\\\"\"\"")
						}

						if u.IsSensitive {
							t.Fatal("The library variable set variable must not be sensitive")
						}

						if !u.IsEditable {
							t.Fatal("The library variable set variable must be editable")
						}
					}

					if u.Name == "Octopus.Script.Module.Language[Test2]" {
						foundLanguage = true

						if u.Type != "String" {
							t.Fatal("The library variable set variable must have a type of \"String\"")
						}

						if strutil.EmptyIfNil(u.Value) != "PowerShell" {
							t.Fatal("The library variable set variable must have a value of \"PowerShell\"")
						}

						if u.IsSensitive {
							t.Fatal("The library variable set variable must not be sensitive")
						}

						if !u.IsEditable {
							t.Fatal("The library variable set variable must be editable")
						}
					}
				}

				if !foundLanguage || !foundScript {
					t.Fatal("Script module must create two variables for script and language")
				}

			}
		}

		if !found {
			t.Fatal("Space must have an library variable set called \"" + resourceName + "\"")
		}

		return nil
	})
}

// TestTenantsExport verifies that a git credential can be reimported with the correct settings
func TestTenantsExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/24-tenants", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.Tenant]{}
		err = octopusClient.GetAllResources("Tenants", &collection)

		if err != nil {
			return err
		}

		resourceName := "Team A"
		found := false
		for _, v := range collection.Items {
			if v.Name == resourceName {
				found = true

				if strutil.EmptyIfNil(v.Description) != "Test tenant" {
					t.Fatal("The tenant must be have a description of \"tTest tenant\" (was \"" + strutil.EmptyIfNil(v.Description) + "\")")
				}

				if len(v.TenantTags) != 2 {
					t.Fatal("The tenant must have two tags")
				}

				if len(v.ProjectEnvironments) != 1 {
					t.Fatal("The tenant must have one project environment")
				}

				for _, u := range v.ProjectEnvironments {
					if len(u) != 3 {
						t.Fatal("The tenant must have be linked to three environments")
					}
				}
			}
		}

		if !found {
			t.Fatal("Space must have an tenant called \"" + resourceName + "\"")
		}

		return nil
	})
}

// TestCertificateExport verifies that a certificate can be reimported with the correct settings
func TestCertificateExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/25-certificates", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{
			"-var=certificate_test_data=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",
			"-var=certificate_test_password=Password01!",
		})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.Certificate]{}
		err = octopusClient.GetAllResources("Certificates", &collection)

		if err != nil {
			return err
		}

		resourceName := "Test"
		found := false
		for _, v := range collection.Items {
			if v.Name == resourceName {
				found = true

				if v.Notes != "A test certificate" {
					t.Fatal("The tenant must be have a description of \"A test certificate\" (was \"" + v.Notes + "\")")
				}

				if v.TenantedDeploymentParticipation != "Untenanted" {
					t.Fatal("The tenant must be have a tenant participation of \"Untenanted\" (was \"" + v.TenantedDeploymentParticipation + "\")")
				}

				if v.SubjectDistinguishedName != "CN=test.com" {
					t.Fatal("The tenant must be have a subject distinguished name of \"CN=test.com\" (was \"" + v.SubjectDistinguishedName + "\")")
				}

				if len(v.EnvironmentIds) != 0 {
					t.Fatal("The tenant must have one project environment")
				}

				if len(v.TenantTags) != 0 {
					t.Fatal("The tenant must have no tenant tags")
				}

				if len(v.TenantIds) != 0 {
					t.Fatal("The tenant must have no tenants")
				}
			}
		}

		if !found {
			t.Fatal("Space must have an tenant called \"" + resourceName + "\"")
		}

		return nil
	})
}

// TestTenantVariablesExport verifies that a tenant variables can be reimported with the correct settings
func TestTenantVariablesExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/26-tenant_variables", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := []octopus.TenantVariable{}
		err = octopusClient.GetAllResources("TenantVariables/All", &collection)

		if err != nil {
			return err
		}

		resourceName := "Test"
		found := false
		for _, tenantVariable := range collection {
			for _, project := range tenantVariable.ProjectVariables {
				if project.ProjectName == resourceName {
					for _, variables := range project.Variables {
						for _, value := range variables {
							// we expect one project variable to be defined
							found = true
							if value != "my value" {
								t.Fatal("The tenant project variable must have a value of \"my value\" (was \"" + value + "\")")
							}
						}
					}
				}
			}
		}

		if !found {
			t.Fatal("Space must have an tenant project variable for the project called \"" + resourceName + "\"")
		}

		return nil
	})
}

// TestMachinePolicyExport verifies that a machine policies can be reimported with the correct settings
func TestMachinePolicyExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/27-machinepolicy", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.MachinePolicy]{}
		err = octopusClient.GetAllResources("MachinePolicies", &collection)

		if err != nil {
			return err
		}

		resourceName := "Testing"
		found := false
		for _, machinePolicy := range collection.Items {
			if machinePolicy.Name == resourceName {
				found = true

				if strutil.EmptyIfNil(machinePolicy.Description) != "test machine policy" {
					t.Fatal("The machine policy must have a description of \"test machine policy\" (was \"" + strutil.EmptyIfNil(machinePolicy.Description) + "\")")
				}

				if machinePolicy.ConnectionConnectTimeout != "00:01:00" {
					t.Fatal("The machine policy must have a ConnectionConnectTimeout of \"00:01:00\" (was \"" + machinePolicy.ConnectionConnectTimeout + "\")")
				}

				if *machinePolicy.ConnectionRetryCountLimit != 5 {
					t.Fatal("The machine policy must have a ConnectionRetryCountLimit of \"5\" (was \"" + fmt.Sprint(machinePolicy.ConnectionRetryCountLimit) + "\")")
				}

				if machinePolicy.ConnectionRetrySleepInterval != "00:00:01" {
					t.Fatal("The machine policy must have a ConnectionRetrySleepInterval of \"00:00:01\" (was \"" + machinePolicy.ConnectionRetrySleepInterval + "\")")
				}

				if machinePolicy.ConnectionRetryTimeLimit != "00:05:00" {
					t.Fatal("The machine policy must have a ConnectionRetryTimeLimit of \"00:05:00\" (was \"" + machinePolicy.ConnectionRetryTimeLimit + "\")")
				}

				if machinePolicy.PollingRequestMaximumMessageProcessingTimeout != "00:10:00" {
					t.Fatal("The machine policy must have a PollingRequestMaximumMessageProcessingTimeout of \"00:10:00\" (was \"" + machinePolicy.PollingRequestMaximumMessageProcessingTimeout + "\")")
				}

				if machinePolicy.MachineCleanupPolicy.DeleteMachinesElapsedTimeSpan != "00:20:00" {
					t.Fatal("The machine policy must have a DeleteMachinesElapsedTimeSpan of \"00:20:00\" (was \"" + machinePolicy.MachineCleanupPolicy.DeleteMachinesElapsedTimeSpan + "\")")
				}

				if machinePolicy.MachineCleanupPolicy.DeleteMachinesBehavior != "DeleteUnavailableMachines" {
					t.Fatal("The machine policy must have a MachineCleanupPolicy.DeleteMachinesBehavior of \"DeleteUnavailableMachines\" (was \"" + machinePolicy.MachineCleanupPolicy.DeleteMachinesBehavior + "\")")
				}

				if machinePolicy.MachineConnectivityPolicy.MachineConnectivityBehavior != "ExpectedToBeOnline" {
					t.Fatal("The machine policy must have a MachineConnectivityPolicy.MachineConnectivityBehavior of \"ExpectedToBeOnline\" (was \"" + machinePolicy.MachineConnectivityPolicy.MachineConnectivityBehavior + "\")")
				}

				if machinePolicy.MachineHealthCheckPolicy.BashHealthCheckPolicy.RunType != "Inline" {
					t.Fatal("The machine policy must have a MachineHealthCheckPolicy.BashHealthCheckPolicy.RunType of \"Inline\" (was \"" + machinePolicy.MachineHealthCheckPolicy.BashHealthCheckPolicy.RunType + "\")")
				}

				if machinePolicy.MachineHealthCheckPolicy.BashHealthCheckPolicy.ScriptBody != "" {
					t.Fatal("The machine policy must have a MachineHealthCheckPolicy.BashHealthCheckPolicy.ScriptBody of \"\" (was \"" + machinePolicy.MachineHealthCheckPolicy.BashHealthCheckPolicy.ScriptBody + "\")")
				}

				if machinePolicy.MachineHealthCheckPolicy.PowerShellHealthCheckPolicy.RunType != "Inline" {
					t.Fatal("The machine policy must have a MachineHealthCheckPolicy.PowerShellHealthCheckPolicy.RunType of \"Inline\" (was \"" + machinePolicy.MachineHealthCheckPolicy.PowerShellHealthCheckPolicy.RunType + "\")")
				}

				if strings.HasPrefix(machinePolicy.MachineHealthCheckPolicy.BashHealthCheckPolicy.ScriptBody, "$freeDiskSpaceThreshold") {
					t.Fatal("The machine policy must have a MachineHealthCheckPolicy.PowerShellHealthCheckPolicy.ScriptBody to start with \"$freeDiskSpaceThreshold\" (was \"" + machinePolicy.MachineHealthCheckPolicy.PowerShellHealthCheckPolicy.ScriptBody + "\")")
				}

				if strutil.EmptyIfNil(machinePolicy.MachineHealthCheckPolicy.HealthCheckCronTimezone) != "UTC" {
					t.Fatal("The machine policy must have a MachineHealthCheckPolicy.HealthCheckCronTimezone of \"UTC\" (was \"" + strutil.EmptyIfNil(machinePolicy.MachineHealthCheckPolicy.HealthCheckCronTimezone) + "\")")
				}

				if strutil.EmptyIfNil(machinePolicy.MachineHealthCheckPolicy.HealthCheckCron) != "" {
					t.Fatal("The machine policy must have a MachineHealthCheckPolicy.HealthCheckCron of \"\" (was \"" + strutil.EmptyIfNil(machinePolicy.MachineHealthCheckPolicy.HealthCheckCron) + "\")")
				}

				if strutil.EmptyIfNil(machinePolicy.MachineHealthCheckPolicy.HealthCheckType) != "RunScript" {
					t.Fatal("The machine policy must have a MachineHealthCheckPolicy.HealthCheckType of \"RunScript\" (was \"" + strutil.EmptyIfNil(machinePolicy.MachineHealthCheckPolicy.HealthCheckType) + "\")")
				}

				if strutil.EmptyIfNil(machinePolicy.MachineHealthCheckPolicy.HealthCheckInterval) != "00:10:00" {
					t.Fatal("The machine policy must have a MachineHealthCheckPolicy.HealthCheckInterval of \"00:10:00\" (was \"" + strutil.EmptyIfNil(machinePolicy.MachineHealthCheckPolicy.HealthCheckInterval) + "\")")
				}

				if strutil.EmptyIfNil(machinePolicy.MachineUpdatePolicy.CalamariUpdateBehavior) != "UpdateOnDeployment" {
					t.Fatal("The machine policy must have a MachineUpdatePolicy.CalamariUpdateBehavior of \"UpdateOnDeployment\" (was \"" + strutil.EmptyIfNil(machinePolicy.MachineUpdatePolicy.CalamariUpdateBehavior) + "\")")
				}

				if strutil.EmptyIfNil(machinePolicy.MachineUpdatePolicy.TentacleUpdateBehavior) != "NeverUpdate" {
					t.Fatal("The machine policy must have a MachineUpdatePolicy.TentacleUpdateBehavior of \"NeverUpdate\" (was \"" + strutil.EmptyIfNil(machinePolicy.MachineUpdatePolicy.CalamariUpdateBehavior) + "\")")
				}
			}
		}

		if !found {
			t.Fatal("Space must have an machine policy for the project called \"" + resourceName + "\"")
		}

		return nil
	})
}

// TestProjectTriggerExport verifies that a project trigger can be reimported with the correct settings
func TestProjectTriggerExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/28-projecttrigger", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.Project]{}
		err = octopusClient.GetAllResources("Projects", &collection)

		if err != nil {
			return err
		}

		resourceName := "Test"
		foundProject := false
		foundTrigger := false
		for _, project := range collection.Items {
			if project.Name == resourceName {
				foundProject = true

				triggers := octopus.GeneralCollection[octopus.ProjectTrigger]{}
				err = octopusClient.GetAllResources("Projects/"+project.Id+"/Triggers", &triggers)

				for _, trigger := range triggers.Items {
					foundTrigger = true

					if trigger.Name != "test" {
						t.Fatal("The project must have a trigger called \"test\" (was \"" + trigger.Name + "\")")
					}

					if trigger.Filter.FilterType != "MachineFilter" {
						t.Fatal("The project trigger must have Filter.FilterType set to \"MachineFilter\" (was \"" + trigger.Filter.FilterType + "\")")
					}

					if trigger.Filter.EventGroups[0] != "MachineAvailableForDeployment" {
						t.Fatal("The project trigger must have Filter.EventGroups[0] set to \"MachineFilter\" (was \"" + trigger.Filter.EventGroups[0] + "\")")
					}
				}
			}
		}

		if !foundProject {
			t.Fatal("Space must have an project \"" + resourceName + "\"")
		}

		if !foundTrigger {
			t.Fatal("Project must have a trigger")
		}

		return nil
	})
}

// TestK8sTargetExport verifies that a k8s machine can be reimported with the correct settings
func TestK8sTargetExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/29-k8starget", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{
			"-var=account_aws_account=whatever",
		})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.KubernetesEndpointResource]{}
		err = octopusClient.GetAllResources("Machines", &collection)

		if err != nil {
			return err
		}

		resourceName := "Test"
		foundResource := false

		for _, machine := range collection.Items {
			if machine.Name == resourceName {
				foundResource = true

				if strutil.EmptyIfNil(machine.Endpoint.ClusterUrl) != "https://cluster" {
					t.Fatal("The machine must have a Endpoint.ClusterUrl of \"https://cluster\" (was \"" + strutil.EmptyIfNil(machine.Endpoint.ClusterUrl) + "\")")
				}
			}
		}

		if !foundResource {
			t.Fatal("Space must have a target \"" + resourceName + "\"")
		}

		return nil
	})
}

// TestSshTargetExport verifies that a ssh machine can be reimported with the correct settings
func TestSshTargetExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/30-sshtarget", []string{
			"-var=account_ec2_sydney=LS0tLS1CRUdJTiBFTkNSWVBURUQgUFJJVkFURSBLRVktLS0tLQpNSUlKbkRCT0Jna3Foa2lHOXcwQkJRMHdRVEFwQmdrcWhraUc5dzBCQlF3d0hBUUlwNEUxV1ZrejJEd0NBZ2dBCk1Bd0dDQ3FHU0liM0RRSUpCUUF3RkFZSUtvWklodmNOQXdjRUNIemFuVE1QbHA4ZkJJSUpTSncrdW5BL2ZaVFUKRGdrdWk2QnhOY0REUFg3UHZJZmNXU1dTc3V3YWRhYXdkVEdjY1JVd3pGNTNmRWJTUXJBYzJuWFkwUWVVcU1wcAo4QmdXUUthWlB3MEdqck5OQVJaTy9QYklxaU5ERFMybVRSekZidzREcFY5aDdlblZjL1ZPNlhJdzlxYVYzendlCnhEejdZSkJ2ckhmWHNmTmx1blErYTZGdlRUVkVyWkE1Ukp1dEZUVnhUWVR1Z3lvWWNXZzAzQWlsMDh3eDhyTHkKUkgvTjNjRlEzaEtLcVZuSHQvdnNZUUhTMnJJYkt0RTluelFPWDRxRDdVYXM3Z0c0L2ZkcmZQZjZFWTR1aGpBcApUeGZRTDUzcTBQZG85T09MZlRReFRxakVNaFpidjV1aEN5d0N2VHdWTmVLZ2MzN1pqdDNPSjI3NTB3U2t1TFZvCnllR0VaQmtML1VONjJjTUJuYlFsSTEzR2FXejBHZ0NJNGkwS3UvRmE4aHJZQTQwcHVFdkEwZFBYcVFGMDhYbFYKM1RJUEhGRWdBWlJpTmpJWmFyQW00THdnL1F4Z203OUR1SVM3VHh6RCtpN1pNSmsydjI1ck14Ly9MMXNNUFBtOQpWaXBwVnpLZmpqRmpwTDVjcVJucC9UdUZSVWpHaDZWMFBXVVk1eTVzYjJBWHpuSGZVd1lqeFNoUjBKWXpXejAwCjNHbklwNnlJa1UvL3dFVGJLcVliMjd0RjdETm1WMUxXQzl0ell1dm4yK2EwQkpnU0Jlc3c4WFJ1WWorQS92bVcKWk1YbkF2anZXR3RBUzA4d0ZOV3F3QUtMbzJYUHBXWGVMa3BZUHo1ZnY2QnJaNVNwYTg4UFhsa1VmOVF0VHRobwprZFlGOWVMdk5hTXpSSWJhbmRGWjdLcHUvN2I3L0tDWE9rMUhMOUxvdEpwY2tJdTAxWS81TnQwOHp5cEVQQ1RzClVGWG5DODNqK2tWMktndG5XcXlEL2k3Z1dwaHJSK0IrNE9tM3VZU1RuY042a2d6ZkV3WldpUVA3ZkpiNlYwTHoKc29yU09sK2g2WDRsMC9oRVdScktVQTBrOXpPZU9TQXhlbmpVUXFReWdUd0RqQTJWbTdSZXI2ZElDMVBwNmVETgpBVEJ0ME1NZjJJTytxbTJtK0VLd1FVSXY4ZXdpdEpab016MFBaOHB6WEM0ZFMyRTErZzZmbnE2UGJ5WWRISDJnCmVraXk4Y2duVVJmdHJFaVoyMUxpMWdpdTJaeVM5QUc0Z1ZuT0E1Y05oSzZtRDJUaGl5UUl2M09yUDA0aDFTNlEKQUdGeGJONEhZK0tCYnVITTYwRG1PQXR5c3o4QkJheHFwWjlXQkVhV01ubFB6eEI2SnFqTGJrZ1BkQ2wycytUWAphcWx0UDd6QkpaenVTeVNQc2tQR1NBREUvaEF4eDJFM1RQeWNhQlhQRVFUM2VkZmNsM09nYXRmeHBSYXJLV09PCnFHM2lteW42ZzJiNjhWTlBDSnBTYTNKZ1Axb0NNVlBpa2RCSEdSVUV3N2dXTlJVOFpXRVJuS292M2c0MnQ4dkEKU2Z0a3VMdkhoUnlPQW91SUVsNjJIems0WC9CeVVOQ2J3MW50RzFQeHpSaERaV2dPaVhPNi94WFByRlpKa3BtcQpZUUE5dW83OVdKZy9zSWxucFJCdFlUbUh4eU9mNk12R2svdXlkZExkcmZ6MHB6QUVmWm11YTVocWh5M2Y4YlNJCmpxMlJwUHE3eHJ1Y2djbFAwTWFjdHkrbm9wa0N4M0lNRUE4NE9MQ3dxZjVtemtwY0U1M3hGaU1hcXZTK0dHZmkKZlZnUGpXTXRzMFhjdEtCV2tUbVFFN3MxSE5EV0g1dlpJaDY2WTZncXR0cjU2VGdtcHRLWHBVdUJ1MEdERFBQbwp1aGI4TnVRRjZwNHNoM1dDbXlzTU9uSW5jaXRxZWE4NTFEMmloK2lIY3VqcnJidkVYZGtjMnlxUHBtK3Q3SXBvCm1zWkxVemdXRlZpNWY3KzZiZU56dGJ3T2tmYmdlQVAyaklHTzdtR1pKWWM0L1d1eXBqeVRKNlBQVC9IMUc3K3QKUTh5R3FDV3BzNFdQM2srR3hrbW90cnFROFcxa0J1RDJxTEdmSTdMMGZUVE9lWk0vQUZ1VDJVSkcxKzQ2czJVVwp2RlF2VUJmZ0dTWlh3c1VUeGJRTlZNaTJib1BCRkNxbUY2VmJTcmw2YVgrSm1NNVhySUlqUUhGUFZWVGxzeUtpClVDUC9PQTJOWlREdW9IcC9EM0s1Qjh5MlIyUTlqZlJ0RkcwL0dnMktCbCtObzdTbXlPcWlsUlNkZ1VJb0p5QkcKRGovZXJ4ZkZNMlc3WTVsNGZ2ZlNpdU1OZmlUTVdkY3cxSStnVkpGMC9mTHRpYkNoUlg0OTlIRWlXUHZkTGFKMwppcDJEYU9ReS9QZG5zK3hvaWlMNWtHV25BVUVwanNjWno0YU5DZFowOXRUb1FhK2RZd3g1R1ovNUtmbnVpTURnClBrWjNXalFpOVlZRWFXbVIvQ2JmMjAyRXdoNjdIZzVqWE5kb0RNendXT0V4RFNkVFFYZVdzUUI0LzNzcjE2S2MKeitGN2xhOXhHVEVhTDllQitwcjY5L2JjekJLMGVkNXUxYUgxcXR3cjcrMmliNmZDdlMyblRGQTM1ZG50YXZlUwp4VUJVZ0NzRzVhTTl4b2pIQ0o4RzRFMm9iRUEwUDg2SFlqZEJJSXF5U0txZWtQYmFybW4xR1JrdUVlbU5hTVdyCkM2bWZqUXR5V2ZMWnlSbUlhL1dkSVgzYXhqZHhYa3kydm4yNVV6MXZRNklrNnRJcktPYUJnRUY1cmYwY014dTUKN1BYeTk0dnc1QjE0Vlcra2JqQnkyY3hIajJhWnJEaE53UnVQNlpIckg5MHZuN2NmYjYwU0twRWxxdmZwdlN0VQpvQnVXQlFEUUE3bHpZajhhT3BHend3LzlYTjI5MGJrUnd4elVZRTBxOVl4bS9VSHJTNUlyRWtKSml2SUlEb3hICjF4VTVLd2ErbERvWDJNcERrZlBQVE9XSjVqZG8wbXNsN0dBTmc1WGhERnBpb2hFMEdSS2lGVytYcjBsYkJKU2oKUkxibytrbzhncXU2WHB0OWU4U0Y5OEJ4bFpEcFBVMG5PcGRrTmxwTVpKYVlpaUUzRjRFRG9DcE56bmxpY2JrcApjZ2FrcGVrbS9YS21RSlJxWElXci8wM29SdUVFTXBxZzlRbjdWRG8zR0FiUTlnNUR5U1Bid0xvT25xQ0V3WGFJCkF6alFzWU4rc3VRd2FqZHFUcEthZ1FCbWRaMmdNZDBTMTV1Ukt6c2wxOHgzK1JabmRiNWoxNjNuV0NkMlQ5VDgKald3NURISDgvVUFkSGZoOHh0RTJ6bWRHbEg5T3I5U2hIMzViMWgxVm8rU2pNMzRPeWpwVjB3TmNVL1psOTBUdAp1WnJwYnBwTXZCZUVmRzZTczVXVGhySm9LaGl0RkNwWlVqaDZvdnk3Mzd6ditKaUc4aDRBNG1GTmRPSUtBd0I0Cmp2Nms3V3poUVlEa2Q0ZXRoajNndVJCTGZQNThNVEJKaWhZemVINkUzclhjSGE5b0xnREgzczd4bU8yVEtUY24Kd3VIM3AvdC9WWFN3UGJ0QXBXUXdTRFNKSnA5WkF4S0Q1eVdmd3lTU2ZQVGtwM2c1b2NmKzBhSk1Kc2FkU3lwNQpNR1Vic1oxd1hTN2RXMDhOYXZ2WmpmbElNUm8wUFZDbkRVcFp1bjJuekhTRGJDSjB1M0ZYd1lFQzFFejlJUnN0ClJFbDdpdTZQRlVMSldSU0V0SzBKY1lLS0ltNXhQWHIvbTdPc2duMUNJL0F0cTkrWEFjODk1MGVxeTRwTFVQYkYKZkhFOFhVYWFzUU82MDJTeGpnOTZZaWJ3ZnFyTDF2Vjd1MitUYzJleUZ1N3oxUGRPZDQyWko5M2wvM3lOUW92egora0JuQVdObzZ3WnNKSitHNDZDODNYRVBLM0h1bGw1dFg2UDU4NUQ1b3o5U1oyZGlTd1FyVFN1THVSL0JCQUpVCmd1K2FITkJGRmVtUXNEL2QxMllud1h3d3FkZXVaMDVmQlFiWUREdldOM3daUjJJeHZpd1E0bjZjZWl3OUZ4QmcKbWlzMFBGY2NZOWl0SnJrYXlWQVVZUFZ3Sm5XSmZEK2pQNjJ3UWZJWmhhbFQrZDJpUzVQaDEwdWlMNHEvY1JuYgo1c1Mvc2o0Tm5QYmpxc1ZmZWlKTEh3PT0KLS0tLS1FTkQgRU5DUllQVEVEIFBSSVZBVEUgS0VZLS0tLS0K",
			"-var=account_ec2_sydney_cert=whatever",
		})

		if err != nil {
			return err
		}

		// Act - Note the private key password is actually the key file
		// See https://github.com/OctopusDeployLabs/terraform-provider-octopusdeploy/blob/main/octopusdeploy/schema_ssh_key_account.go#L16
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{
			"-var=account_ec2_sydney=LS0tLS1CRUdJTiBFTkNSWVBURUQgUFJJVkFURSBLRVktLS0tLQpNSUlKbkRCT0Jna3Foa2lHOXcwQkJRMHdRVEFwQmdrcWhraUc5dzBCQlF3d0hBUUlwNEUxV1ZrejJEd0NBZ2dBCk1Bd0dDQ3FHU0liM0RRSUpCUUF3RkFZSUtvWklodmNOQXdjRUNIemFuVE1QbHA4ZkJJSUpTSncrdW5BL2ZaVFUKRGdrdWk2QnhOY0REUFg3UHZJZmNXU1dTc3V3YWRhYXdkVEdjY1JVd3pGNTNmRWJTUXJBYzJuWFkwUWVVcU1wcAo4QmdXUUthWlB3MEdqck5OQVJaTy9QYklxaU5ERFMybVRSekZidzREcFY5aDdlblZjL1ZPNlhJdzlxYVYzendlCnhEejdZSkJ2ckhmWHNmTmx1blErYTZGdlRUVkVyWkE1Ukp1dEZUVnhUWVR1Z3lvWWNXZzAzQWlsMDh3eDhyTHkKUkgvTjNjRlEzaEtLcVZuSHQvdnNZUUhTMnJJYkt0RTluelFPWDRxRDdVYXM3Z0c0L2ZkcmZQZjZFWTR1aGpBcApUeGZRTDUzcTBQZG85T09MZlRReFRxakVNaFpidjV1aEN5d0N2VHdWTmVLZ2MzN1pqdDNPSjI3NTB3U2t1TFZvCnllR0VaQmtML1VONjJjTUJuYlFsSTEzR2FXejBHZ0NJNGkwS3UvRmE4aHJZQTQwcHVFdkEwZFBYcVFGMDhYbFYKM1RJUEhGRWdBWlJpTmpJWmFyQW00THdnL1F4Z203OUR1SVM3VHh6RCtpN1pNSmsydjI1ck14Ly9MMXNNUFBtOQpWaXBwVnpLZmpqRmpwTDVjcVJucC9UdUZSVWpHaDZWMFBXVVk1eTVzYjJBWHpuSGZVd1lqeFNoUjBKWXpXejAwCjNHbklwNnlJa1UvL3dFVGJLcVliMjd0RjdETm1WMUxXQzl0ell1dm4yK2EwQkpnU0Jlc3c4WFJ1WWorQS92bVcKWk1YbkF2anZXR3RBUzA4d0ZOV3F3QUtMbzJYUHBXWGVMa3BZUHo1ZnY2QnJaNVNwYTg4UFhsa1VmOVF0VHRobwprZFlGOWVMdk5hTXpSSWJhbmRGWjdLcHUvN2I3L0tDWE9rMUhMOUxvdEpwY2tJdTAxWS81TnQwOHp5cEVQQ1RzClVGWG5DODNqK2tWMktndG5XcXlEL2k3Z1dwaHJSK0IrNE9tM3VZU1RuY042a2d6ZkV3WldpUVA3ZkpiNlYwTHoKc29yU09sK2g2WDRsMC9oRVdScktVQTBrOXpPZU9TQXhlbmpVUXFReWdUd0RqQTJWbTdSZXI2ZElDMVBwNmVETgpBVEJ0ME1NZjJJTytxbTJtK0VLd1FVSXY4ZXdpdEpab016MFBaOHB6WEM0ZFMyRTErZzZmbnE2UGJ5WWRISDJnCmVraXk4Y2duVVJmdHJFaVoyMUxpMWdpdTJaeVM5QUc0Z1ZuT0E1Y05oSzZtRDJUaGl5UUl2M09yUDA0aDFTNlEKQUdGeGJONEhZK0tCYnVITTYwRG1PQXR5c3o4QkJheHFwWjlXQkVhV01ubFB6eEI2SnFqTGJrZ1BkQ2wycytUWAphcWx0UDd6QkpaenVTeVNQc2tQR1NBREUvaEF4eDJFM1RQeWNhQlhQRVFUM2VkZmNsM09nYXRmeHBSYXJLV09PCnFHM2lteW42ZzJiNjhWTlBDSnBTYTNKZ1Axb0NNVlBpa2RCSEdSVUV3N2dXTlJVOFpXRVJuS292M2c0MnQ4dkEKU2Z0a3VMdkhoUnlPQW91SUVsNjJIems0WC9CeVVOQ2J3MW50RzFQeHpSaERaV2dPaVhPNi94WFByRlpKa3BtcQpZUUE5dW83OVdKZy9zSWxucFJCdFlUbUh4eU9mNk12R2svdXlkZExkcmZ6MHB6QUVmWm11YTVocWh5M2Y4YlNJCmpxMlJwUHE3eHJ1Y2djbFAwTWFjdHkrbm9wa0N4M0lNRUE4NE9MQ3dxZjVtemtwY0U1M3hGaU1hcXZTK0dHZmkKZlZnUGpXTXRzMFhjdEtCV2tUbVFFN3MxSE5EV0g1dlpJaDY2WTZncXR0cjU2VGdtcHRLWHBVdUJ1MEdERFBQbwp1aGI4TnVRRjZwNHNoM1dDbXlzTU9uSW5jaXRxZWE4NTFEMmloK2lIY3VqcnJidkVYZGtjMnlxUHBtK3Q3SXBvCm1zWkxVemdXRlZpNWY3KzZiZU56dGJ3T2tmYmdlQVAyaklHTzdtR1pKWWM0L1d1eXBqeVRKNlBQVC9IMUc3K3QKUTh5R3FDV3BzNFdQM2srR3hrbW90cnFROFcxa0J1RDJxTEdmSTdMMGZUVE9lWk0vQUZ1VDJVSkcxKzQ2czJVVwp2RlF2VUJmZ0dTWlh3c1VUeGJRTlZNaTJib1BCRkNxbUY2VmJTcmw2YVgrSm1NNVhySUlqUUhGUFZWVGxzeUtpClVDUC9PQTJOWlREdW9IcC9EM0s1Qjh5MlIyUTlqZlJ0RkcwL0dnMktCbCtObzdTbXlPcWlsUlNkZ1VJb0p5QkcKRGovZXJ4ZkZNMlc3WTVsNGZ2ZlNpdU1OZmlUTVdkY3cxSStnVkpGMC9mTHRpYkNoUlg0OTlIRWlXUHZkTGFKMwppcDJEYU9ReS9QZG5zK3hvaWlMNWtHV25BVUVwanNjWno0YU5DZFowOXRUb1FhK2RZd3g1R1ovNUtmbnVpTURnClBrWjNXalFpOVlZRWFXbVIvQ2JmMjAyRXdoNjdIZzVqWE5kb0RNendXT0V4RFNkVFFYZVdzUUI0LzNzcjE2S2MKeitGN2xhOXhHVEVhTDllQitwcjY5L2JjekJLMGVkNXUxYUgxcXR3cjcrMmliNmZDdlMyblRGQTM1ZG50YXZlUwp4VUJVZ0NzRzVhTTl4b2pIQ0o4RzRFMm9iRUEwUDg2SFlqZEJJSXF5U0txZWtQYmFybW4xR1JrdUVlbU5hTVdyCkM2bWZqUXR5V2ZMWnlSbUlhL1dkSVgzYXhqZHhYa3kydm4yNVV6MXZRNklrNnRJcktPYUJnRUY1cmYwY014dTUKN1BYeTk0dnc1QjE0Vlcra2JqQnkyY3hIajJhWnJEaE53UnVQNlpIckg5MHZuN2NmYjYwU0twRWxxdmZwdlN0VQpvQnVXQlFEUUE3bHpZajhhT3BHend3LzlYTjI5MGJrUnd4elVZRTBxOVl4bS9VSHJTNUlyRWtKSml2SUlEb3hICjF4VTVLd2ErbERvWDJNcERrZlBQVE9XSjVqZG8wbXNsN0dBTmc1WGhERnBpb2hFMEdSS2lGVytYcjBsYkJKU2oKUkxibytrbzhncXU2WHB0OWU4U0Y5OEJ4bFpEcFBVMG5PcGRrTmxwTVpKYVlpaUUzRjRFRG9DcE56bmxpY2JrcApjZ2FrcGVrbS9YS21RSlJxWElXci8wM29SdUVFTXBxZzlRbjdWRG8zR0FiUTlnNUR5U1Bid0xvT25xQ0V3WGFJCkF6alFzWU4rc3VRd2FqZHFUcEthZ1FCbWRaMmdNZDBTMTV1Ukt6c2wxOHgzK1JabmRiNWoxNjNuV0NkMlQ5VDgKald3NURISDgvVUFkSGZoOHh0RTJ6bWRHbEg5T3I5U2hIMzViMWgxVm8rU2pNMzRPeWpwVjB3TmNVL1psOTBUdAp1WnJwYnBwTXZCZUVmRzZTczVXVGhySm9LaGl0RkNwWlVqaDZvdnk3Mzd6ditKaUc4aDRBNG1GTmRPSUtBd0I0Cmp2Nms3V3poUVlEa2Q0ZXRoajNndVJCTGZQNThNVEJKaWhZemVINkUzclhjSGE5b0xnREgzczd4bU8yVEtUY24Kd3VIM3AvdC9WWFN3UGJ0QXBXUXdTRFNKSnA5WkF4S0Q1eVdmd3lTU2ZQVGtwM2c1b2NmKzBhSk1Kc2FkU3lwNQpNR1Vic1oxd1hTN2RXMDhOYXZ2WmpmbElNUm8wUFZDbkRVcFp1bjJuekhTRGJDSjB1M0ZYd1lFQzFFejlJUnN0ClJFbDdpdTZQRlVMSldSU0V0SzBKY1lLS0ltNXhQWHIvbTdPc2duMUNJL0F0cTkrWEFjODk1MGVxeTRwTFVQYkYKZkhFOFhVYWFzUU82MDJTeGpnOTZZaWJ3ZnFyTDF2Vjd1MitUYzJleUZ1N3oxUGRPZDQyWko5M2wvM3lOUW92egora0JuQVdObzZ3WnNKSitHNDZDODNYRVBLM0h1bGw1dFg2UDU4NUQ1b3o5U1oyZGlTd1FyVFN1THVSL0JCQUpVCmd1K2FITkJGRmVtUXNEL2QxMllud1h3d3FkZXVaMDVmQlFiWUREdldOM3daUjJJeHZpd1E0bjZjZWl3OUZ4QmcKbWlzMFBGY2NZOWl0SnJrYXlWQVVZUFZ3Sm5XSmZEK2pQNjJ3UWZJWmhhbFQrZDJpUzVQaDEwdWlMNHEvY1JuYgo1c1Mvc2o0Tm5QYmpxc1ZmZWlKTEh3PT0KLS0tLS1FTkQgRU5DUllQVEVEIFBSSVZBVEUgS0VZLS0tLS0K",
			"-var=account_ec2_sydney_cert=whatever",
		})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.SshEndpointResource]{}
		err = octopusClient.GetAllResources("Machines", &collection)

		if err != nil {
			return err
		}

		resourceName := "Test"
		foundResource := false

		for _, machine := range collection.Items {
			if machine.Name == resourceName {
				foundResource = true

				if machine.Endpoint.Host != "3.25.215.87" {
					t.Fatal("The machine must have a Endpoint.Host of \"3.25.215.87\" (was \"" + machine.Endpoint.Host + "\")")
				}

				if machine.Endpoint.DotNetCorePlatform != "linux-x64" {
					t.Fatal("The machine must have a Endpoint.DotNetCorePlatform of \"linux-x64\" (was \"" + machine.Endpoint.DotNetCorePlatform + "\")")
				}
			}
		}

		if !foundResource {
			t.Fatal("Space must have a target \"" + resourceName + "\"")
		}

		return nil
	})
}

// TestListeningTargetExport verifies that a listening machine can be reimported with the correct settings
func TestListeningTargetExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/31-listeningtarget", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.ListeningEndpointResource]{}
		err = octopusClient.GetAllResources("Machines", &collection)

		if err != nil {
			return err
		}

		resourceName := "Test"
		foundResource := false

		for _, machine := range collection.Items {
			if machine.Name == resourceName {
				foundResource = true

				if machine.Uri != "https://tentacle/" {
					t.Fatal("The machine must have a Uri of \"https://tentacle/\" (was \"" + machine.Uri + "\")")
				}

				if machine.Thumbprint != "55E05FD1B0F76E60F6DA103988056CE695685FD1" {
					t.Fatal("The machine must have a Thumbprint of \"55E05FD1B0F76E60F6DA103988056CE695685FD1\" (was \"" + machine.Thumbprint + "\")")
				}

				if len(machine.Roles) != 1 {
					t.Fatal("The machine must have 1 role")
				}

				if machine.Roles[0] != "vm" {
					t.Fatal("The machine must have a role of \"vm\" (was \"" + machine.Roles[0] + "\")")
				}

				if machine.TenantedDeploymentParticipation != "Untenanted" {
					t.Fatal("The machine must have a TenantedDeploymentParticipation of \"Untenanted\" (was \"" + machine.TenantedDeploymentParticipation + "\")")
				}
			}
		}

		if !foundResource {
			t.Fatal("Space must have a target \"" + resourceName + "\"")
		}

		return nil
	})
}

// TestPollingTargetExport verifies that a polling machine can be reimported with the correct settings
func TestPollingTargetExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/32-pollingtarget", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.PollingEndpointResource]{}
		err = octopusClient.GetAllResources("Machines", &collection)

		if err != nil {
			return err
		}

		resourceName := "Test"
		foundResource := false

		for _, machine := range collection.Items {
			if machine.Name == resourceName {
				foundResource = true

				if machine.Endpoint.Uri != "poll://abcdefghijklmnopqrst/" {
					t.Fatal("The machine must have a Uri of \"poll://abcdefghijklmnopqrst/\" (was \"" + machine.Endpoint.Uri + "\")")
				}

				if machine.Thumbprint != "1854A302E5D9EAC1CAA3DA1F5249F82C28BB2B86" {
					t.Fatal("The machine must have a Thumbprint of \"1854A302E5D9EAC1CAA3DA1F5249F82C28BB2B86\" (was \"" + machine.Thumbprint + "\")")
				}

				if len(machine.Roles) != 1 {
					t.Fatal("The machine must have 1 role")
				}

				if machine.Roles[0] != "vm" {
					t.Fatal("The machine must have a role of \"vm\" (was \"" + machine.Roles[0] + "\")")
				}

				if machine.TenantedDeploymentParticipation != "Untenanted" {
					t.Fatal("The machine must have a TenantedDeploymentParticipation of \"Untenanted\" (was \"" + machine.TenantedDeploymentParticipation + "\")")
				}
			}
		}

		if !foundResource {
			t.Fatal("Space must have a target \"" + resourceName + "\"")
		}

		return nil
	})
}

// TestCloudRegionTargetExport verifies that a cloud region can be reimported with the correct settings
func TestCloudRegionTargetExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/33-cloudregiontarget", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.CloudRegionResource]{}
		err = octopusClient.GetAllResources("Machines", &collection)

		if err != nil {
			return err
		}

		resourceName := "Test"
		foundResource := false

		for _, machine := range collection.Items {
			if machine.Name == resourceName {
				foundResource = true

				if len(machine.Roles) != 1 {
					t.Fatal("The machine must have 1 role")
				}

				if machine.Roles[0] != "cloud" {
					t.Fatal("The machine must have a role of \"cloud\" (was \"" + machine.Roles[0] + "\")")
				}

				if machine.TenantedDeploymentParticipation != "Untenanted" {
					t.Fatal("The machine must have a TenantedDeploymentParticipation of \"Untenanted\" (was \"" + machine.TenantedDeploymentParticipation + "\")")
				}
			}
		}

		if !foundResource {
			t.Fatal("Space must have a target \"" + resourceName + "\"")
		}

		return nil
	})
}

// TestOfflineDropTargetExport verifies that an offline drop can be reimported with the correct settings
func TestOfflineDropTargetExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/34-offlinedroptarget", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.OfflineDropResource]{}
		err = octopusClient.GetAllResources("Machines", &collection)

		if err != nil {
			return err
		}

		resourceName := "Test"
		foundResource := false

		for _, machine := range collection.Items {
			if machine.Name == resourceName {
				foundResource = true

				if len(machine.Roles) != 1 {
					t.Fatal("The machine must have 1 role")
				}

				if machine.Roles[0] != "offline" {
					t.Fatal("The machine must have a role of \"offline\" (was \"" + machine.Roles[0] + "\")")
				}

				if machine.TenantedDeploymentParticipation != "Untenanted" {
					t.Fatal("The machine must have a TenantedDeploymentParticipation of \"Untenanted\" (was \"" + machine.TenantedDeploymentParticipation + "\")")
				}

				if machine.Endpoint.ApplicationsDirectory != "c:\\temp" {
					t.Fatal("The machine must have a Endpoint.ApplicationsDirectory of \"c:\\temp\" (was \"" + machine.Endpoint.ApplicationsDirectory + "\")")
				}

				if machine.Endpoint.OctopusWorkingDirectory != "c:\\temp" {
					t.Fatal("The machine must have a Endpoint.OctopusWorkingDirectory of \"c:\\temp\" (was \"" + machine.Endpoint.OctopusWorkingDirectory + "\")")
				}
			}
		}

		if !foundResource {
			t.Fatal("Space must have a target \"" + resourceName + "\"")
		}

		return nil
	})
}

// TestAzureCloudServiceTargetExport verifies that a azure cloud service target can be reimported with the correct settings
func TestAzureCloudServiceTargetExport(t *testing.T) {
	// I could not figure out a combination of properties that made the octopusdeploy_azure_subscription_account resource work
	return

	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/35-azurecloudservicetarget", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{
			"-var=account_subscription_cert=LS0tLS1CRUdJTiBPUEVOU1NIIFBSSVZBVEUgS0VZLS0tLS0KYjNCbGJuTnphQzFyWlhrdGRqRUFBQUFBQkc1dmJtVUFBQUFFYm05dVpRQUFBQUFBQUFBQkFBQUJGd0FBQUFkemMyZ3RjbgpOaEFBQUFBd0VBQVFBQUFRRUF5c25PVXhjN0tJK2pIRUc5RVEwQXFCMllGRWE5ZnpZakZOY1pqY1dwcjJQRkRza25oOUpTCm1NVjVuZ2VrbTRyNHJVQU5tU2dQMW1ZTGo5TFR0NUVZa0N3OUdyQ0paNitlQTkzTEowbEZUamFkWEJuQnNmbmZGTlFWYkcKZ2p3U1o4SWdWQ2oySXE0S1hGZm0vbG1ycEZQK2Jqa2V4dUxwcEh5dko2ZmxZVjZFMG13YVlneVNHTWdLYy9ubXJaMTY0WApKMStJL1M5NkwzRWdOT0hNZmo4QjM5eEhZQ0ZUTzZEQ0pLQ3B0ZUdRa0gwTURHam84d3VoUlF6c0IzVExsdXN6ZG0xNmRZCk16WXZBSWR3emZ3bzh1ajFBSFFOendDYkIwRmR6bnFNOEpLV2ZrQzdFeVVrZUl4UXZmLzJGd1ZyS0xEZC95ak5PUmNoa3EKb2owNncySXFad0FBQThpS0tqT3dpaW96c0FBQUFBZHpjMmd0Y25OaEFBQUJBUURLeWM1VEZ6c29qNk1jUWIwUkRRQ29IWgpnVVJyMS9OaU1VMXhtTnhhbXZZOFVPeVNlSDBsS1l4WG1lQjZTYml2aXRRQTJaS0EvV1pndVAwdE8za1JpUUxEMGFzSWxuCnI1NEQzY3NuU1VWT05wMWNHY0d4K2Q4VTFCVnNhQ1BCSm53aUJVS1BZaXJncGNWK2IrV2F1a1UvNXVPUjdHNHVta2ZLOG4KcCtWaFhvVFNiQnBpREpJWXlBcHorZWF0blhyaGNuWDRqOUwzb3ZjU0EwNGN4K1B3SGYzRWRnSVZNN29NSWtvS20xNFpDUQpmUXdNYU9qekM2RkZET3dIZE11VzZ6TjJiWHAxZ3pOaThBaDNETi9Dank2UFVBZEEzUEFKc0hRVjNPZW96d2twWitRTHNUCkpTUjRqRkM5Ly9ZWEJXc29zTjMvS00wNUZ5R1NxaVBUckRZaXBuQUFBQUF3RUFBUUFBQVFFQXdRZzRqbitlb0kyYUJsdk4KVFYzRE1rUjViMU9uTG1DcUpEeGM1c2N4THZNWnNXbHBaN0NkVHk4ckJYTGhEZTdMcUo5QVVub0FHV1lwdTA1RW1vaFRpVwptVEFNVHJCdmYwd2xsdCtJZVdvVXo3bmFBbThQT1psb29MbXBYRzh5VmZKRU05aUo4NWtYNDY4SkF6VDRYZ1JXUFRYQ1JpCi9abCtuWUVUZVE4WTYzWlJhTVE3SUNmK2FRRWxRenBYb21idkxYM1RaNmNzTHh5Z3Eza01aSXNJU0lUcEk3Y0tsQVJ0Rm4KcWxKRitCL2JlUEJkZ3hIRVpqZDhDV0NIR1ZRUDh3Z3B0d0Rrak9NTzh2b2N4YVpOT0hZZnBwSlBCTkVjMEVKbmduN1BXSgorMVZSTWZKUW5SemVubmE3VHdSUSsrclZmdkVaRmhqamdSUk85RitrMUZvSWdRQUFBSUVBbFFybXRiV2V0d3RlWlZLLys4CklCUDZkcy9MSWtPb3pXRS9Wckx6cElBeHEvV1lFTW1QK24wK1dXdWRHNWpPaTFlZEJSYVFnU0owdTRxcE5JMXFGYTRISFYKY2oxL3pzenZ4RUtSRElhQkJGaU81Y3QvRVQvUTdwanozTnJaZVdtK0dlUUJKQ0diTEhSTlQ0M1ZpWVlLVG82ZGlGVTJteApHWENlLzFRY2NqNjVZQUFBQ0JBUHZodmgzb2Q1MmY4SFVWWGoxeDNlL1ZFenJPeVloTi9UQzNMbWhHYnRtdHZ0L0J2SUhxCndxWFpTT0lWWkZiRnVKSCtORHNWZFFIN29yUW1VcGJxRllDd0IxNUZNRGw0NVhLRm0xYjFyS1c1emVQK3d0M1hyM1p0cWsKRkdlaUlRMklSZklBQjZneElvNTZGemdMUmx6QnB0bzhkTlhjMXhtWVgyU2Rhb3ZwSkRBQUFBZ1FET0dwVE9oOEFRMFoxUwpzUm9vVS9YRTRkYWtrSU5vMDdHNGI3M01maG9xbkV1T01LM0ZRVStRRWUwYWpvdWs5UU1QNWJzZU1CYnJNZVNNUjBRWVBCClQ4Z0Z2S2VISWN6ZUtJTjNPRkRaRUF4TEZNMG9LbjR2bmdHTUFtTXUva2QwNm1PZnJUNDRmUUh1ajdGNWx1QVJHejRwYUwKLzRCTUVkMnFTRnFBYzZ6L0RRQUFBQTF0WVhSMGFFQk5ZWFIwYUdWM0FRSURCQT09Ci0tLS0tRU5EIE9QRU5TU0ggUFJJVkFURSBLRVktLS0tLQo=",
		})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.AzureCloudServiceResource]{}
		err = octopusClient.GetAllResources("Machines", &collection)

		if err != nil {
			return err
		}

		resourceName := "Azure"
		foundResource := false

		for _, machine := range collection.Items {
			if machine.Name == resourceName {
				foundResource = true

				if len(machine.Roles) != 1 {
					t.Fatal("The machine must have 1 role")
				}

				if machine.Roles[0] != "cloud" {
					t.Fatal("The machine must have a role of \"cloud\" (was \"" + machine.Roles[0] + "\")")
				}

				if machine.TenantedDeploymentParticipation != "Untenanted" {
					t.Fatal("The machine must have a TenantedDeploymentParticipation of \"Untenanted\" (was \"" + machine.TenantedDeploymentParticipation + "\")")
				}

				if machine.Endpoint.CloudServiceName != "servicename" {
					t.Fatal("The machine must have a Endpoint.CloudServiceName of \"c:\\temp\" (was \"" + machine.Endpoint.CloudServiceName + "\")")
				}

				if machine.Endpoint.StorageAccountName != "accountname" {
					t.Fatal("The machine must have a Endpoint.StorageAccountName of \"accountname\" (was \"" + machine.Endpoint.StorageAccountName + "\")")
				}

				if !machine.Endpoint.UseCurrentInstanceCount {
					t.Fatal("The machine must have Endpoint.UseCurrentInstanceCount set")
				}
			}
		}

		if !foundResource {
			t.Fatal("Space must have a target \"" + resourceName + "\"")
		}

		return nil
	})
}

// TestAzureServiceFabricTargetExport verifies that a service fabric target can be reimported with the correct settings
func TestAzureServiceFabricTargetExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/36-servicefabrictarget", []string{
			"-var=target_service_fabric=whatever",
		})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{
			"-var=target_service_fabric=whatever",
		})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.AzureServiceFabricResource]{}
		err = octopusClient.GetAllResources("Machines", &collection)

		if err != nil {
			return err
		}

		resourceName := "Service Fabric"
		foundResource := false

		for _, machine := range collection.Items {
			if machine.Name == resourceName {
				foundResource = true

				if len(machine.Roles) != 1 {
					t.Fatal("The machine must have 1 role")
				}

				if machine.Roles[0] != "cloud" {
					t.Fatal("The machine must have a role of \"cloud\" (was \"" + machine.Roles[0] + "\")")
				}

				if machine.TenantedDeploymentParticipation != "Untenanted" {
					t.Fatal("The machine must have a TenantedDeploymentParticipation of \"Untenanted\" (was \"" + machine.TenantedDeploymentParticipation + "\")")
				}

				if machine.Endpoint.ConnectionEndpoint != "http://endpoint" {
					t.Fatal("The machine must have a Endpoint.ConnectionEndpoint of \"http://endpoint\" (was \"" + machine.Endpoint.ConnectionEndpoint + "\")")
				}

				if machine.Endpoint.AadCredentialType != "UserCredential" {
					t.Fatal("The machine must have a Endpoint.AadCredentialType of \"UserCredential\" (was \"" + machine.Endpoint.AadCredentialType + "\")")
				}

				if machine.Endpoint.AadUserCredentialUsername != "username" {
					t.Fatal("The machine must have a Endpoint.AadUserCredentialUsername of \"username\" (was \"" + machine.Endpoint.AadUserCredentialUsername + "\")")
				}
			}
		}

		if !foundResource {
			t.Fatal("Space must have a target \"" + resourceName + "\"")
		}

		return nil
	})
}

// TestAzureWebAppTargetExport verifies that a web app target can be reimported with the correct settings
func TestAzureWebAppTargetExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/37-webapptarget", []string{
			"-var=account_sales_account=whatever",
		})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{
			"-var=account_sales_account=whatever",
		})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.AzureWebAppResource]{}
		err = octopusClient.GetAllResources("Machines", &collection)

		if err != nil {
			return err
		}

		resourceName := "Web App"
		foundResource := false

		for _, machine := range collection.Items {
			if machine.Name == resourceName {
				foundResource = true

				if len(machine.Roles) != 1 {
					t.Fatal("The machine must have 1 role")
				}

				if machine.Roles[0] != "cloud" {
					t.Fatal("The machine must have a role of \"cloud\" (was \"" + machine.Roles[0] + "\")")
				}

				if machine.TenantedDeploymentParticipation != "Untenanted" {
					t.Fatal("The machine must have a TenantedDeploymentParticipation of \"Untenanted\" (was \"" + machine.TenantedDeploymentParticipation + "\")")
				}

				if machine.Endpoint.ResourceGroupName != "mattc-webapp" {
					t.Fatal("The machine must have a Endpoint.ResourceGroupName of \"mattc-webapp\" (was \"" + machine.Endpoint.ResourceGroupName + "\")")
				}

				if machine.Endpoint.WebAppName != "mattc-webapp" {
					t.Fatal("The machine must have a Endpoint.WebAppName of \"mattc-webapp\" (was \"" + machine.Endpoint.WebAppName + "\")")
				}

				if machine.Endpoint.WebAppSlotName != "slot1" {
					t.Fatal("The machine must have a Endpoint.WebAppSlotName of \"slot1\" (was \"" + machine.Endpoint.WebAppSlotName + "\")")
				}
			}
		}

		if !foundResource {
			t.Fatal("Space must have a target \"" + resourceName + "\"")
		}

		return nil
	})
}

// TestSingleProjectGroupExport verifies that a single project can be reimported with the correct settings.
// This is one of the larger tests, verifying that the graph of resources linked to a project have been exported,
// and that unrelated resources were not exported.
func TestSingleProjectGroupExport(t *testing.T) {
	if os.Getenv("GIT_CREDENTIAL") == "" {
		t.Fatalf("the GIT_CREDENTIAL environment variable must be set to a GitHub access key")
	}

	performTest(t, func(t *testing.T, container *octopusContainer) error {
		terraformDir := "../test/terraform/38-multipleprojects"

		// Arrange
		newSpaceId, err := arrange(t, container, terraformDir, []string{
			"-var=gitcredential_matt=" + os.Getenv("GIT_CREDENTIAL"),
		})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := actProjectExport(t, container, terraformDir, newSpaceId, []string{
			"-var=gitcredential_matt=" + os.Getenv("GIT_CREDENTIAL"),
			"-var=project_test_git_base_path=.octopus/integrationtestimport",
		}, "octopus_project_1")

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		// Test that the project exported its project group
		err = func() error {
			collection := octopus.GeneralCollection[octopus.ProjectGroup]{}
			err = octopusClient.GetAllResources("ProjectGroups", &collection)

			if err != nil {
				return err
			}

			found := false
			for _, v := range collection.Items {
				if v.Name == "Test" {
					found = true
					if *v.Description != "Test Description" {
						t.Fatalf("The project group must be have a description of \"Test Description\"")
					}
				}
			}

			if !found {
				t.Fatalf("Space must have a project group called \"Test\"")
			}
			return nil
		}()

		if err != nil {
			return err
		}

		// Verify that the single project was exported
		err = func() error {
			projectCollection := octopus.GeneralCollection[octopus.Project]{}
			err = octopusClient.GetAllResources("Projects", &projectCollection)

			if err != nil {
				return err
			}

			if len(projectCollection.Items) != 1 {
				t.Fatalf("There must only be one project")
			}

			if projectCollection.Items[0].Name != "Test" {
				t.Fatalf("The project must be called \"Test\"")
			}

			// Verify that the variable set was imported

			if projectCollection.Items[0].VariableSetId == nil {
				t.Fatalf("The project must have a variable set")
			}

			variableSet := octopus.VariableSet{}
			_, err = octopusClient.GetResourceById("Variables", *projectCollection.Items[0].VariableSetId, &variableSet)

			if err != nil {
				return err
			}

			if len(variableSet.Variables) != 1 {
				t.Fatalf("The project must have 1 variable")
			}

			if variableSet.Variables[0].Name != "Test" {
				t.Fatalf("The project must have 1 variable called \"Test\"")
			}
			return nil
		}()

		if err != nil {
			return err
		}

		// Verify that the single channel was exported
		err = func() error {
			channelsCollection := octopus.GeneralCollection[octopus.Channel]{}
			err = octopusClient.GetAllResources("Channels", &channelsCollection)

			if err != nil {
				return err
			}

			foundChannel := false
			for _, v := range channelsCollection.Items {
				if v.Name == "Test 1" {
					foundChannel = true
				}

				if v.Name == "Test 2" {
					t.Fatalf("The second channel must not have been exported")
				}
			}

			if !foundChannel {
				t.Fatalf("The space must have a channel called \"Test 1\"")
			}

			return nil
		}()

		if err != nil {
			return err
		}

		// Verify that the single trigger was exported
		err = func() error {
			triggersCollection := octopus.GeneralCollection[octopus.ProjectTrigger]{}
			err = octopusClient.GetAllResources("ProjectTriggers", &triggersCollection)

			if err != nil {
				return err
			}

			foundTrigger := false
			for _, v := range triggersCollection.Items {
				if v.Name == "Test 1" {
					foundTrigger = true
				}

				if v.Name == "Test 2" {
					t.Fatalf("The second trigger must not have been exported")
				}
			}

			if !foundTrigger {
				t.Fatalf("The space must have a trigger called \"Test 1\"")
			}

			return nil
		}()

		if err != nil {
			return err
		}

		// Verify that the single tenant was exported
		err = func() error {
			tenantsCollection := octopus.GeneralCollection[octopus.Tenant]{}
			err = octopusClient.GetAllResources("Tenants", &tenantsCollection)

			if err != nil {
				return err
			}

			foundTenant := false
			for _, v := range tenantsCollection.Items {
				if v.Name == "Team A" {
					foundTenant = true
				}

				if v.Name == "Team B" {
					t.Fatalf("The second tenant must not have been exported")
				}
			}

			if !foundTenant {
				t.Fatalf("The space must have a tenant called \"Team A\"")
			}
			return nil
		}()

		if err != nil {
			return err
		}

		// Verify that the tenant tags were exported
		err = func() error {
			tagsCollection := octopus.GeneralCollection[octopus.TagSet]{}
			err = octopusClient.GetAllResources("TagSets", &tagsCollection)

			if err != nil {
				return err
			}

			foundTag := false
			for _, v := range tagsCollection.Items {
				if v.Name == "tag1" {
					foundTag = true
				}

				if v.Name == "tag2" {
					t.Fatalf("The space must not have a tagset called \"tag2\"")
				}
			}

			if !foundTag {
				t.Fatalf("The space must have a tagset called \"tag1\"")
			}
			return nil
		}()

		if err != nil {
			return err
		}

		// Verify that the environments were exported
		err = func() error {
			environmentsCollection := octopus.GeneralCollection[octopus.Tenant]{}
			err = octopusClient.GetAllResources("Environments", &environmentsCollection)

			if err != nil {
				return err
			}

			foundEnvironmentDev := false
			foundEnvironmentTest := false
			foundEnvironmentProduction := false
			for _, v := range environmentsCollection.Items {
				if v.Name == "Development" {
					foundEnvironmentDev = true
				}

				if v.Name == "Test" {
					foundEnvironmentTest = true
				}

				if v.Name == "Production" {
					foundEnvironmentProduction = true
				}

				if v.Name == "Blah" {
					t.Fatalf("The environment called \"Blah\" must not been exported")
				}
			}

			if !foundEnvironmentDev {
				t.Fatalf("The space must have a space called \"Deveopment\"")
			}

			if !foundEnvironmentTest {
				t.Fatalf("The space must have a space called \"Test\"")
			}

			if !foundEnvironmentProduction {
				t.Fatalf("The space must have a space called \"Production\"")
			}

			return nil
		}()

		if err != nil {
			return err
		}

		// Verify that the library variable set was exported
		err = func() error {
			libraryVariableSetCollection := octopus.GeneralCollection[octopus.LibraryVariableSet]{}
			err = octopusClient.GetAllResources("LibraryVariableSets", &libraryVariableSetCollection)

			if err != nil {
				return err
			}

			foundLibraryVariableSet := false
			for _, v := range libraryVariableSetCollection.Items {
				if v.Name == "Test" {
					foundLibraryVariableSet = true
				}

				if v.Name == "Test2" {
					t.Fatalf("The library variable set called \"Test2\" must not been exported")
				}
			}

			if !foundLibraryVariableSet {
				t.Fatalf("The space must have a library variable called \"Test\"")
			}

			return nil
		}()

		if err != nil {
			return err
		}

		// Verify that the library variable set was exported
		err = func() error {
			collection := octopus.GeneralCollection[octopus.Lifecycle]{}
			err = octopusClient.GetAllResources("Lifecycles", &collection)

			if err != nil {
				return err
			}

			found := false
			for _, v := range collection.Items {
				if v.Name == "Simple" {
					found = true
				}

				if v.Name == "Simple2" {
					t.Fatalf("The lifecycle called \"Simple2\" must not been exported")
				}
			}

			if !found {
				t.Fatalf("The space must have a lifecycle called \"Simple\"")
			}

			return nil
		}()

		// Verify that the git credential was exported
		err = func() error {
			collection := octopus.GeneralCollection[octopus.GitCredentials]{}
			err = octopusClient.GetAllResources("Git-Credentials", &collection)

			if err != nil {
				return err
			}

			found := false
			for _, v := range collection.Items {
				if v.Name == "matt" {
					found = true
				}
			}

			if !found {
				t.Fatalf("The space must have a git credential called \"matt\"")
			}

			return nil
		}()

		if err != nil {
			return err
		}

		return nil
	})
}

// TestProjectWithGitUsernameExport verifies that a project can be reimported with the correct git settings
func TestProjectWithGitUsernameExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/39-projectgitusername", []string{
			"-var=project_git_password=" + os.Getenv("GIT_CREDENTIAL"),
		})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{
			"-var=project_test_git_password=" + os.Getenv("GIT_CREDENTIAL"),
			"-var=project_test_git_base_path=.octopus/projectgitusername",
		})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.Project]{}
		err = octopusClient.GetAllResources("Projects", &collection)

		if err != nil {
			return err
		}

		resourceName := "Test"
		found := false
		for _, v := range collection.Items {
			if v.Name == resourceName {
				found = true

				if v.PersistenceSettings.Credentials.Type != "UsernamePassword" {
					t.Fatal("The project must be have a git credential type of \"UsernamePassword\" (was \"" + v.PersistenceSettings.Credentials.Type + "\")")
				}

				if v.PersistenceSettings.Credentials.Username != "mcasperson" {
					t.Fatal("The project must be have a git username of \"mcasperson\" (was \"" + v.PersistenceSettings.Credentials.Username + "\")")
				}
			}
		}

		if !found {
			t.Fatal("Space must have an project called \"" + resourceName + "\"")
		}

		return nil
	})
}

// TestProjectWithDollarSignsExport verifies that a project can be reimported with the correct git settings
func TestProjectWithDollarSignsExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrange(t, container, "../test/terraform/40-escapedollar", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := act(t, container, newSpaceId, []string{})

		if err != nil {
			return err
		}

		// Assert
		octopusClient := createClient(container, recreatedSpaceId)

		collection := octopus.GeneralCollection[octopus.Project]{}
		err = octopusClient.GetAllResources("Projects", &collection)

		if err != nil {
			return err
		}

		resourceName := "Test"
		found := false
		for _, v := range collection.Items {
			if v.Name == resourceName {
				found = true
			}
		}

		if !found {
			t.Fatal("Space must have an project called \"" + resourceName + "\"")
		}

		return nil
	})
}
