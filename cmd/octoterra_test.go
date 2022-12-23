package main

import (
	"context"
	"fmt"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"net/http"
	"testing"
	"time"
)

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
			"ACCEPT_EULA":          "Y",
			"DB_CONNECTION_STRING": connString,
			"ADMIN_API_KEY":        "API-ABCDEFGHIJKLMNOPQURTUVWXYZ12345",
			"DISABLE_DIND":         "Y",
			"ADMIN_USERNAME":       "admin",
			"ADMIN_PASSWORD":       "Password01!",
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

	uri := fmt.Sprintf("http://%s:%s/api", ip, mappedPort.Port())

	return &octopusContainer{Container: container, URI: uri}, nil
}

func TestOctopusExportAndRecreate(t *testing.T) {
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

		resp, err := http.Get(octopusContainer.URI)
		if err == nil && resp.StatusCode == http.StatusOK {
			success = true
			t.Log("Successfully contacted the Octopus API")
			break
		}

		time.Sleep(10 * time.Second)
	}

	if !success {
		t.Fatal("Failed to access the Octopus API")
	}

}
