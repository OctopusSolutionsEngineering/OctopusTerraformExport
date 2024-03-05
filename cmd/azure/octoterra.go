package main

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/entry"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/logger"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"go.uber.org/zap"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type AzureFunctionRequestDataReq struct {
	Body string `json:"Body"`
}

type AzureFunctionRequestData struct {
	Req AzureFunctionRequestDataReq `json:"req"`
}

type AzureFunctionRequest struct {
	Data AzureFunctionRequestData `json:"Data"`
}

func octoterraHandler(w http.ResponseWriter, r *http.Request) {
	respBytes, err := io.ReadAll(r.Body)

	if err != nil {
		handleError(err, w)
		return
	}

	//var requestBody AzureFunctionRequest
	//err = json.Unmarshal(respBytes, &requestBody)
	//
	//if err != nil {
	//	handleError(err, w)
	//	return
	//}

	file, err := os.CreateTemp("", "*.json")

	if err != nil {
		handleError(err, w)
		return
	}

	err = os.WriteFile(file.Name(), []byte(respBytes), 0644)

	if err != nil {
		handleError(err, w)
		return
	}

	// Clean up the file when we are done
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			zap.L().Error(err.Error())
		}
	}(file.Name())

	filename := filepath.Base(file.Name())
	extension := filepath.Ext(filename)
	filenameWithoutExtension := filename[0 : len(filename)-len(extension)]

	webArgs, _, err := args.ParseArgs([]string{"-configFile", filenameWithoutExtension, "-configPath", filepath.Dir(file.Name())})

	if err != nil {
		handleError(err, w)
		return
	}

	files, err := entry.Entry(webArgs)

	if err != nil {
		handleError(err, w)
		return
	}

	var sb strings.Builder
	for _, str := range strutil.UnEscapeDollarInMap(files) {
		sb.WriteString(str + "\n\n")
	}

	w.Header()["Content-Type"] = []string{"text/plain; charset=utf-8"}
	w.WriteHeader(200)
	if _, err := w.Write([]byte(sb.String())); err != nil {
		zap.L().Error(err.Error())
	}
}

func handleError(err error, w http.ResponseWriter) {
	zap.L().Error(err.Error())
	w.WriteHeader(500)
	if _, err := w.Write([]byte(err.Error())); err != nil {
		zap.L().Error(err.Error())
	}
}

func main() {
	logger.BuildLogger()

	listenAddr := ":8080"
	if val, ok := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT"); ok {
		listenAddr = ":" + val
	}
	http.HandleFunc("/api/octoterra", octoterraHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header()["Content-Type"] = []string{"application/json; charset=utf-8"}
		w.WriteHeader(200)
		w.Write([]byte("{\"Hello\": \"" + r.RequestURI + "\"}"))
	})
	log.Printf("About to listen on %s. Go to https://127.0.0.1%s/", listenAddr, listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
