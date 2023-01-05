package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"k8s.io/utils/strings/slices"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
)

const API_KEY = "API-ABCDEFGHIJKLMNOPQURTUVWXYZ12345"

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
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	// Display the container logs
	container.StartLogProducer(ctx)
	if err != nil {
		// do something with err
	}
	g := TestLogConsumer{}
	container.FollowOutput(&g)

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
	req := testcontainers.ContainerRequest{
		Image:        "octopusdeploy/octopusdeploy",
		ExposedPorts: []string{"8080/tcp"},
		Env: map[string]string{
			"ACCEPT_EULA":                   "Y",
			"DB_CONNECTION_STRING":          connString,
			"ADMIN_API_KEY":                 API_KEY,
			"DISABLE_DIND":                  "Y",
			"ADMIN_USERNAME":                "admin",
			"ADMIN_PASSWORD":                "Password01!",
			"OCTOPUS_SERVER_BASE64_LICENSE": os.Getenv("LICENSE"),
		},
		Privileged: false,
		WaitingFor: wait.ForLog("Listening for HTTP requests on").WithStartupTimeout(10 * time.Minute),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	// Display the container logs
	container.StartLogProducer(ctx)
	if err != nil {
		// do something with err
	}
	g := TestLogConsumer{}
	container.FollowOutput(&g)

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
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	sqlServer, err := setupDatabase(ctx)
	if err != nil {
		t.Fatal(err)
	}

	octopusContainer, err := setupOctopus(ctx, "Server=172.17.0.1,"+sqlServer.port+";Database=OctopusDeploy;User=sa;Password=Password01!")
	if err != nil {
		t.Fatal(err)
	}

	// Clean up the container after the test is complete
	defer func() {
		octoTerminateErr := octopusContainer.Terminate(ctx)
		sqlTerminateErr := sqlServer.Terminate(ctx)

		if octoTerminateErr != nil || sqlTerminateErr != nil {
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

	// perform the test
	err = testFunc(t, octopusContainer)

	if err != nil {
		t.Fatalf(err.Error())
	}
}

// initialiseOctopus uses Terraform to populate the test Octopus instance, making sure to clean up
// any files generated during previous Terraform executions to avoid conflicts and locking issues.
func initialiseOctopus(t *testing.T, container *octopusContainer, terraformDir string, spaceName string, populateVars []string) error {
	path, err := os.Getwd()
	if err != nil {
		return err
	}
	t.Log("Working dir: " + path)

	// First loop initialises the new space, second populates the space
	spaceId := "Spaces-1"
	for i, p := range []string{"/space_creation", "/space_population"} {

		os.Remove(terraformDir + p + "/.terraform.lock.hcl")
		os.Remove(terraformDir + p + "/terraform.tfstate")
		os.Remove(terraformDir + p + "/.terraform")

		cmnd := exec.Command(
			"terraform",
			"init",
			"-no-color")
		cmnd.Dir = terraformDir + p
		out, err := cmnd.Output()

		if err != nil {
			t.Log(string(err.(*exec.ExitError).Stderr))
			return err
		}

		t.Log(string(out))

		// when initialising the new space, we need to define a new space name as a variable
		vars := []string{}
		if i == 0 {
			vars = []string{"-var=octopus_space_name=" + spaceName}
		} else {
			vars = populateVars
		}

		newArgs := append([]string{
			"apply",
			"-auto-approve",
			"-no-color",
			"-var=octopus_server=" + container.URI,
			"-var=octopus_apikey=" + API_KEY,
			"-var=octopus_space_id=" + spaceId,
		}, vars...)

		cmnd = exec.Command("terraform", newArgs...)
		cmnd.Dir = terraformDir + p
		out, err = cmnd.Output()

		if err != nil {
			t.Log(string(err.(*exec.ExitError).Stderr))
			return err
		}

		t.Log(string(out))

		// get the ID of any new space created, which will be used in the subsequent Terraform executions
		spaceId, err = getOutputVariable(t, terraformDir+p, "octopus_space_id")

		if err != nil {
			t.Log(string(err.(*exec.ExitError).Stderr))
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
		t.Log(string(err.(*exec.ExitError).Stderr))
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
		ApiKey: API_KEY,
	}
}

// arrangeTest initialises Octopus and MSSQL
func arrangeTest(t *testing.T, container *octopusContainer, terraformDir string) (string, error) {
	err := initialiseOctopus(t, container, terraformDir, "Test2", []string{})

	if err != nil {
		return "", err
	}

	return getOutputVariable(t, terraformDir+"/space_creation", "octopus_space_id")
}

// actTest exports the Octopus configuration as Terraform, and reimports it as a new space
func actTest(t *testing.T, container *octopusContainer, newSpaceId string, populateVars []string) (string, error) {
	tempDir := getTempDir()
	defer os.Remove(tempDir)

	err := ConvertToTerraform(container.URI, newSpaceId, API_KEY, tempDir, true)

	if err != nil {
		return "", err
	}

	err = initialiseOctopus(t, container, tempDir, "Test3", populateVars)

	if err != nil {
		return "", err
	}

	return getOutputVariable(t, tempDir+"/space_creation", "octopus_space_id")
}

// TestSpaceExport verifies that a space can be reimported with the correct settings
func TestSpaceExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrangeTest(t, container, "../test/terraform/1-singlespace")

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := actTest(t, container, newSpaceId, []string{})

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

		if *space.Name != "Test3" {
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
		newSpaceId, err := arrangeTest(t, container, "../test/terraform/2-projectgroup")

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := actTest(t, container, newSpaceId, []string{})

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
			if *v.Name == "Test" {
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
		newSpaceId, err := arrangeTest(t, container, "../test/terraform/3-awsaccount")

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := actTest(t, container, newSpaceId, []string{"-var=account_aws_account=secretgoeshere"})

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

// TestAzureAccountExport verifies that an AWS account can be reimported with the correct settings
func TestAzureAccountExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrangeTest(t, container, "../test/terraform/4-azureaccount")

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := actTest(t, container, newSpaceId, []string{"-var=account_azure=secretgoeshere"})

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
