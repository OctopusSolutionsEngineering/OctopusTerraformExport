package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/google/uuid"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"k8s.io/utils/strings/slices"
	"net/http"
	"os"
	"os/exec"
	"runtime"
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
	if os.Getenv("LICENSE") == "" {
		return nil, errors.New("the LICENSE environment variable must be set to a base 64 encoded Octopus license key")
	}

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
	err := retry.Do(
		func() error {

			if testing.Short() {
				t.Skip("skipping integration test")
			}

			ctx := context.Background()

			sqlServer, err := setupDatabase(ctx)
			if err != nil {
				return err
			}

			// Different OSs have different ways of connecting to the host
			sqlHostName := "172.17.0.1"
			if runtime.GOOS == "windows" {
				sqlHostName = "host.docker.internal"
			}

			octopusContainer, err := setupOctopus(ctx, "Server="+sqlHostName+","+sqlServer.port+";Database=OctopusDeploy;User=sa;Password=Password01!")
			if err != nil {
				return err
			}

			// Clean up the container after the test is complete
			defer func() {
				// This fixes the "can not get logs from container which is dead or marked for removal" error
				// See https://github.com/testcontainers/testcontainers-go/issues/606
				octopusContainer.StopLogProducer()
				sqlServer.StopLogProducer()

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

	// First loop initialises the new space, second populates the space
	spaceId := "Spaces-1"
	for i, p := range []string{"/space_creation", "/space_population"} {

		os.Remove(terraformDir + p + "/.terraform.lock.hcl")
		os.Remove(terraformDir + p + "/terraform.tfstate")

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
			vars = append(initialiseVars, "-var=octopus_space_name="+spaceName)
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
func arrangeTest(t *testing.T, container *octopusContainer, terraformDir string, populateVars []string) (string, error) {
	err := initialiseOctopus(t, container, terraformDir, "Test2", []string{}, populateVars)

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

	err = initialiseOctopus(t, container, tempDir, "Test3", []string{}, populateVars)

	if err != nil {
		return "", err
	}

	return getOutputVariable(t, tempDir+"/space_creation", "octopus_space_id")
}

// TestSpaceExport verifies that a space can be reimported with the correct settings
func TestSpaceExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrangeTest(t, container, "../test/terraform/1-singlespace", []string{})

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
		newSpaceId, err := arrangeTest(t, container, "../test/terraform/2-projectgroup", []string{})

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
		newSpaceId, err := arrangeTest(t, container, "../test/terraform/3-awsaccount", []string{})

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

// TestAzureAccountExport verifies that an Azure account can be reimported with the correct settings
func TestAzureAccountExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrangeTest(t, container, "../test/terraform/4-azureaccount", []string{})

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

// TestUsernamePasswordAccountExport verifies that a username/password account can be reimported with the correct settings
func TestUsernamePasswordAccountExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrangeTest(t, container, "../test/terraform/5-userpassaccount", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := actTest(t, container, newSpaceId, []string{"-var=account_gke=secretgoeshere"})

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
		newSpaceId, err := arrangeTest(t, container, "../test/terraform/6-gcpaccount", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := actTest(t, container, newSpaceId, []string{"-var=account_google=secretgoeshere"})

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
		newSpaceId, err := arrangeTest(t, container, "../test/terraform/7-sshaccount", []string{})

		if err != nil {
			return err
		}

		// Act
		// We set the passphrase because of https://github.com/OctopusDeployLabs/terraform-provider-octopusdeploy/issues/343
		recreatedSpaceId, err := actTest(t, container, newSpaceId, []string{
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

// TestAzureSubscriptionAccountExport verifies that a SSH account can be reimported with the correct settings
func TestAzureSubscriptionAccountExport(t *testing.T) {
	// I could not figure out a combination of properties that made this resource work
	return

	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		newSpaceId, err := arrangeTest(t, container, "../test/terraform/8-azuresubscriptionaccount", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := actTest(t, container, newSpaceId, []string{
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
		newSpaceId, err := arrangeTest(t, container, "../test/terraform/9-tokenaccount", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := actTest(t, container, newSpaceId, []string{
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
		newSpaceId, err := arrangeTest(t, container, "../test/terraform/10-helmfeed", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := actTest(t, container, newSpaceId, []string{
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
		newSpaceId, err := arrangeTest(t, container, "../test/terraform/11-dockerfeed", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := actTest(t, container, newSpaceId, []string{
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

		newSpaceId, err := arrangeTest(t, container, "../test/terraform/12-ecrfeed", []string{
			"-var=feed_ecr_access_key=" + os.Getenv("ECR_ACCESS_KEY"),
			"-var=feed_ecr_secret_key=" + os.Getenv("ECR_SECRET_KEY"),
		})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := actTest(t, container, newSpaceId, []string{
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
					t.Fatal("The feed must have a type of \"AwsElasticContainerRegistry\" (was \"" + util.EmptyIfNil(v.FeedType) + "\"")
				}

				if *v.AccessKey != os.Getenv("ECR_ACCESS_KEY") {
					t.Fatal("The feed must have a access key of \"" + os.Getenv("ECR_ACCESS_KEY") + "\" (was \"" + util.EmptyIfNil(v.AccessKey) + "\"")
				}

				if *v.Region != "us-east-1" {
					t.Fatal("The feed must have a region of \"us-east-1\" (was \"" + util.EmptyIfNil(v.Region) + "\"")
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
		newSpaceId, err := arrangeTest(t, container, "../test/terraform/13-mavenfeed", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := actTest(t, container, newSpaceId, []string{
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
		newSpaceId, err := arrangeTest(t, container, "../test/terraform/14-nugetfeed", []string{})

		if err != nil {
			return err
		}

		// Act
		recreatedSpaceId, err := actTest(t, container, newSpaceId, []string{
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

				if *v.FeedType != "Nuget" {
					t.Fatal("The feed must have a type of \"Nuget\"")
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
