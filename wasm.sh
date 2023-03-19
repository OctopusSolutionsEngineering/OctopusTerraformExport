#!/bin/bash

GOOS=js GOARCH=wasm go build -o wasm/convert_project.wasm cmd/convert_project_js_wasm.go