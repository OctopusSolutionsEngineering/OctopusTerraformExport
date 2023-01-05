package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
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
func initialiseOctopus(t *testing.T, container *octopusContainer, terraformDir string, vars []string) error {
	path, err := os.Getwd()
	if err != nil {
		return err
	}
	t.Log("Working dir: " + path)

	os.Remove(terraformDir + "/.terraform.lock.hcl")
	os.Remove(terraformDir + "/terraform.tfstate")
	os.Remove(terraformDir + "/.terraform")

	cmnd := exec.Command(
		"terraform",
		"init",
		"-no-color")
	cmnd.Dir = terraformDir
	out, err := cmnd.Output()

	if err != nil {
		t.Log(string(err.(*exec.ExitError).Stderr))
		return err
	}

	t.Log(string(out))

	newArgs := append([]string{
		"apply",
		"-auto-approve",
		"-no-color",
		"-var=octopus_server=" + container.URI,
		"-var=octopus_apikey=" + API_KEY,
	}, vars...)

	cmnd = exec.Command("terraform", newArgs...)
	cmnd.Dir = terraformDir
	out, err = cmnd.Output()

	if err != nil {
		t.Log(string(err.(*exec.ExitError).Stderr))
		return err
	}

	t.Log(string(out))

	return nil
}

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

func getTempDir() string {
	return os.TempDir() + string(os.PathSeparator) + uuid.New().String() + string(os.PathSeparator)
}

func createClient(container *octopusContainer, space string) *client.OctopusClient {
	return &client.OctopusClient{
		Url:    container.URI,
		Space:  space,
		ApiKey: API_KEY,
	}
}

func TestSpaceExport(t *testing.T) {
	performTest(t, func(t *testing.T, container *octopusContainer) error {
		// Arrange
		terraformDir := "../test/terraform/1-singlespace"
		err := initialiseOctopus(t, container, terraformDir, []string{})

		if err != nil {
			return err
		}

		newSpaceId, err := getOutputVariable(t, terraformDir, "octopus_space_id")

		// Act
		tempDir := getTempDir()
		defer os.Remove(tempDir)

		err = ConvertToTerraform(container.URI, newSpaceId, API_KEY, tempDir, true)

		if err != nil {
			return err
		}

		err = initialiseOctopus(t, container, tempDir, []string{"-var=octopus_space_test_name=Test2"})

		recreatedSpaceId, err := getOutputVariable(t, tempDir, "octopus_space_id")

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

		if *space.Name != "Test2" {
			t.Fatalf("New space must have the name Test2")
		}

		return nil
	})
}
