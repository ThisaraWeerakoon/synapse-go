/*
 *  Licensed to the Apache Software Foundation (ASF) under one
 *  or more contributor license agreements.  See the NOTICE file
 *  distributed with this work for additional information
 *  regarding copyright ownership.  The ASF licenses this file
 *  to you under the Apache License, Version 2.0 (the
 *  "License"); you may not use this file except in compliance
 *  with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing,
 *  software distributed under the License is distributed on an
 *   * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 *  KIND, either express or implied.  See the License for the
 *  specific language governing permissions and limitations
 *  under the License.
 */

// Package router provides HTTP routing capabilities for Synapse APIs.
//
// The RouterService is the main component of this package, providing:
// - API registration with automatic route creation from resources
// - HTTP server lifecycle management with automatic start/stop
// - Request handling with conversion to/from Synapse message contexts
// - Method-based routing for RESTful APIs
//
// Usage:
//
//	// Create a router service
//	rs := router.NewRouterService(":8290")
//
//	// Register an API with the router
//	rs.RegisterAPI(ctx, myAPI)
//
//	// Later, gracefully shut down
//	rs.Shutdown(ctx)
package router

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"encoding/json"

	"github.com/apache/synapse-go/internal/pkg/core/artifacts"
	"github.com/apache/synapse-go/internal/pkg/core/synctx"
	"gopkg.in/yaml.v2"
)

// RouterService manages API routing and server lifecycle
type RouterService struct {
	server     *http.Server
	router     *http.ServeMux
	listenAddr string
	mu         sync.RWMutex
	started    bool
}

// NewRouterService creates a new router service with the given listen address
func NewRouterService(listenAddr string) *RouterService {
	return &RouterService{
		router:     http.NewServeMux(),
		listenAddr: listenAddr,
		started:    false,
	}
}

// RegisterAPI registers a new API with the router service
func (rs *RouterService) RegisterAPI(ctx context.Context, api artifacts.API) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Determine base path based on context and version
	basePath := api.Context

	// Remove trailing slash from context if present
	if len(basePath) > 1 && basePath[len(basePath)-1] == '/' {
		basePath = basePath[:len(basePath)-1]
	}

	// Handle versioning based on versionType
	if api.Version != "" && api.VersionType != "" {
		switch api.VersionType {
		case "url":
			// For URL type, add version as a path segment
			basePath = basePath + "/" + api.Version
		case "context":
			// For context type, replace {version} placeholder if it exists
			versionPattern := "{version}"
			basePath = strings.Replace(basePath, versionPattern, api.Version, 1)
		}
	}

	// Register swagger documentation handlers with appropriate versioning in URL
	// If version is not empty, register at /<API_NAME>/<API_VERSION>
	// If version is empty, register at /<API_NAME>
	swaggerBasePath := "/" + api.Name
	if api.Version != "" {
		swaggerBasePath = swaggerBasePath + "/" + api.Version
	}

	rs.router.HandleFunc(swaggerBasePath, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Has("swagger.yaml") {
			rs.serveSwaggerYAML(w, r, api)
			return
		} else if query.Has("swagger.json") {
			rs.serveSwaggerJSON(w, r, api)
			return
		} else if query.Has("swagger.html") {
			rs.serveSwaggerHTML(w, r, api)
			return
		}
		http.NotFound(w, r)
	})

	// Create a subrouter for this API
	apiHandler := http.NewServeMux()

	// Register each resource in the API
	for _, resource := range api.Resources {
		// Register a handler for each HTTP method in the resource
		for _, method := range resource.Methods {
			// Construct the full pattern: "METHOD /path/to/resource"
			pattern := method + " " + resource.URITemplate
			apiHandler.HandleFunc(pattern, rs.createResourceHandler(resource))
			fmt.Printf("Registered route for API: '%s': %s\n", api.Name, pattern)
			// No need to register explicit OPTIONS handlers when using rs/cors package
			// The CORSMiddleware already handles OPTIONS preflight requests automatically
		}
	}

	// Apply CORS middleware to the entire API subrouter if enabled
	var handler http.Handler = apiHandler
	if api.CORSConfig.Enabled {
		handler = CORSMiddleware(handler, api.CORSConfig)
	}

	// Register the API handler with the main router
	rs.router.Handle(basePath+"/", http.StripPrefix(basePath, handler))

	// Start the server if it hasn't been started yet
	if !rs.started {
		if err := rs.startServer(ctx); err != nil {
			return fmt.Errorf("failed to start server: %w", err)
		}
	}

	return nil
}

// createHandlerFunc creates an HTTP handler function for the given API resource
func (rs *RouterService) createResourceHandler(resource artifacts.Resource) http.HandlerFunc {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Create message context
		msgContext := synctx.CreateMsgContext()

		// Store the *http.Request in the message context properties.
		if msgContext.Properties == nil {
			msgContext.Properties = make(map[string]string)
		}
		//Store pointer to request as string representation
		msgContext.Properties["http_request"] = fmt.Sprintf("%v", r)

		// Process through mediation pipeline
		success := resource.Mediate(msgContext)

		// Write response
		if success {
			for name, value := range msgContext.Headers {
				w.Header().Set(name, value)
			}
			if msgContext.Message.RawPayload != nil {
				w.Write(msgContext.Message.RawPayload)
			}
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
	return handler
}

// startServer starts the HTTP server
func (rs *RouterService) startServer(ctx context.Context) error {
	rs.server = &http.Server{
		Addr:    rs.listenAddr,
		Handler: rs.router,
	}

	// Start the server in a goroutine
	go func() {
		fmt.Printf("Starting HTTP server on %s\n", rs.listenAddr)
		if err := rs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	// Start a goroutine to monitor context cancellation and shut down server
	go func() {
		<-ctx.Done()
		fmt.Println("Shutting down HTTP server...")
		if err := rs.Shutdown(ctx); err != nil {
			fmt.Printf("Error shutting down HTTP server: %v\n", err)
		} else {
			fmt.Println("HTTP server stopped gracefully")
		}
	}()

	rs.started = true
	return nil
}

// Shutdown gracefully shuts down the server
func (rs *RouterService) Shutdown(ctx context.Context) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.server != nil && rs.started {
		fmt.Println("Shutting down HTTP server")
		return rs.server.Shutdown(ctx)
	}
	return nil
}

// serveSwaggerYAML serves the swagger.yaml documentation for the API
func (rs *RouterService) serveSwaggerYAML(w http.ResponseWriter, r *http.Request, api artifacts.API) {
	swagger := rs.generateSwaggerDoc(api)
	yamlData, err := yaml.Marshal(swagger)
	if err != nil {
		http.Error(w, "Failed to generate YAML", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/yaml")
	w.Write(yamlData)
}

// serveSwaggerJSON serves the swagger.json documentation for the API
func (rs *RouterService) serveSwaggerJSON(w http.ResponseWriter, r *http.Request, api artifacts.API) {
	swagger := rs.generateSwaggerDoc(api)
	jsonData, err := json.MarshalIndent(swagger, "", "  ")
	if err != nil {
		http.Error(w, "Failed to generate JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

// serveSwaggerHTML serves the swagger.html documentation for the API
func (rs *RouterService) serveSwaggerHTML(w http.ResponseWriter, r *http.Request, api artifacts.API) {
	// HTML template for Swagger UI
	htmlTemplate := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Swagger UI - %s</title>
  <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@4.5.0/swagger-ui.css" />
  <script src="https://unpkg.com/swagger-ui-dist@4.5.0/swagger-ui-bundle.js" charset="UTF-8"></script>
  <style>
    html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin: 0; background: #fafafa; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script>
    window.onload = function() {
      SwaggerUIBundle({
        url: "%s?swagger.json",
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIBundle.SwaggerUIStandalonePreset
        ],
        layout: "BaseLayout"
      });
    }
  </script>
</body>
</html>`

	var swaggerJsonUrl string
	if api.Version != "" {
		swaggerJsonUrl = fmt.Sprintf("/%s/%s", api.Name, api.Version)
	} else {
		swaggerJsonUrl = fmt.Sprintf("/%s", api.Name)
	}

	htmlContent := fmt.Sprintf(htmlTemplate, api.Name, swaggerJsonUrl)
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(htmlContent))
}

// generateSwaggerDoc creates a swagger/OpenAPI representation of the API
func (rs *RouterService) generateSwaggerDoc(api artifacts.API) map[string]interface{} {
	// Determine base path based on context and version
	basePath := api.Context
	if len(basePath) > 1 && basePath[len(basePath)-1] == '/' {
		basePath = basePath[:len(basePath)-1]
	}

	// Handle versioning based on versionType
	if api.Version != "" && api.VersionType != "" {
		switch api.VersionType {
		case "url":
			basePath = basePath + "/" + api.Version
		case "context":
			versionPattern := "{version}"
			basePath = strings.Replace(basePath, versionPattern, api.Version, 1)
		}
	}

	// Create basic swagger document
	swagger := map[string]interface{}{
		"openapi": "3.0.3",
		"info": map[string]interface{}{
			"title":       api.Name,
			"description": "API automatically generated from Synapse API definition",
			"version":     api.Version,
		},
		"servers": []map[string]interface{}{
			{
				"url": basePath,
			},
		},
		"paths": map[string]interface{}{},
	}

	paths := swagger["paths"].(map[string]interface{})

	// Add paths and methods from API resources
	for _, resource := range api.Resources {
		pathItem := map[string]interface{}{}

		for _, method := range resource.Methods {
			methodLower := strings.ToLower(method)

			// Skip OPTIONS method as it's handled by CORS
			if methodLower == "options" {
				continue
			}

			operation := map[string]interface{}{
				"summary":     fmt.Sprintf("%s %s", method, resource.URITemplate),
				"description": fmt.Sprintf("Operation for %s %s", method, resource.URITemplate),
				"operationId": fmt.Sprintf("%s_%s", methodLower, strings.ReplaceAll(strings.Trim(resource.URITemplate, "/"), "/", "_")),
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Successful operation",
					},
					"500": map[string]interface{}{
						"description": "Internal server error",
					},
				},
			}

			// Extract path parameters
			params := rs.extractPathParams(resource.URITemplate)
			if len(params) > 0 {
				paramDefs := []map[string]interface{}{}
				for _, param := range params {
					paramDefs = append(paramDefs, map[string]interface{}{
						"name":        param,
						"in":          "path",
						"required":    true,
						"description": fmt.Sprintf("Parameter %s", param),
						"schema": map[string]interface{}{
							"type": "string",
						},
					})
				}
				operation["parameters"] = paramDefs
			}

			pathItem[methodLower] = operation
		}

		paths[resource.URITemplate] = pathItem
	}

	// Add components section with security schemes if needed
	if api.CORSConfig.Enabled {
		// Check if there's any security related headers in CORS config
		for _, header := range api.CORSConfig.AllowHeaders {
			if strings.ToLower(header) == "authorization" {
				swagger["components"] = map[string]interface{}{
					"securitySchemes": map[string]interface{}{
						"bearerAuth": map[string]interface{}{
							"type":         "http",
							"scheme":       "bearer",
							"bearerFormat": "JWT",
						},
					},
				}
				break
			}
		}
	}

	return swagger
}

// extractPathParams extracts path parameters from a URI template
func (rs *RouterService) extractPathParams(uriTemplate string) []string {
	params := []string{}
	segments := strings.Split(uriTemplate, "/")

	for _, segment := range segments {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			// Extract parameter name without braces
			paramName := segment[1 : len(segment)-1]
			params = append(params, paramName)
		}
	}

	return params
}
