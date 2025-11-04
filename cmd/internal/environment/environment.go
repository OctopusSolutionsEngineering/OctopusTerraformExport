package environment

import (
	"encoding/json"
	"os"
	"strings"

	"go.uber.org/zap"
)

func GetPort() string {
	// Get the port from the environment variable
	port := os.Getenv("FUNCTIONS_CUSTOMHANDLER_PORT")
	if port == "" {
		port = os.Getenv("OCTOTERRA_FUNCTIONS_CUSTOMHANDLER_PORT")
		if port == "" {
			port = "8080" // Default port
		}
	}
	return port
}

func GetRedirectionBypass() []string {
	hostnames := []string{}
	hostnamesJson := os.Getenv("REDIRECTION_BYPASS")
	if hostnamesJson == "" {
		return []string{} // Default to empty slice if not set
	}

	err := json.Unmarshal([]byte(hostnamesJson), &hostnames)
	if err != nil {
		zap.L().Error("Error parsing redirection bypass JSON:", zap.Error(err))
		return []string{}
	}

	return hostnames
}

func GetRedirectionForce() bool {
	redirectionForce := os.Getenv("REDIRECTION_FORCE")
	return strings.ToLower(redirectionForce) == "true"
}

func GetRedirectionDisable() bool {
	redirectionForce := os.Getenv("REDIRECTION_DISABLE")
	return strings.ToLower(redirectionForce) == "true"
}
