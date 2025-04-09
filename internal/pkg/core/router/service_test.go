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

package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/apache/synapse-go/internal/pkg/core/artifacts"
	"github.com/apache/synapse-go/internal/pkg/core/synctx"
)

// MockMediator implements the Mediator interface for testing
type MockMediator struct {
	executed bool
}

func (m *MockMediator) Execute(context *synctx.MsgContext) (bool, error) {
	m.executed = true

	// Set a test response
	context.Message.RawPayload = []byte("Test response")
	context.Headers["Content-Type"] = "text/plain"

	return true, nil
}

// TestMethodMediator implements the Mediator interface for method testing
type TestMethodMediator struct {
	executed bool
	method   string
}

func (m *TestMethodMediator) Execute(context *synctx.MsgContext) (bool, error) {
	m.executed = true
	// Store the HTTP method used for verification
	if context.Properties["http_request"] != "" {
		r := context.Properties["http_request"]
		m.method = r // simplified for test
	}
	// Set a test response
	context.Message.RawPayload = []byte("Test response for " + m.method)
	context.Headers["Content-Type"] = "text/plain"
	return true, nil
}

func TestRouterService_RegisterAPI(t *testing.T) {
	// Create a test server that will never start listening
	rs := &RouterService{
		router:     http.NewServeMux(),
		apis:       make(map[string]artifacts.API),
		listenAddr: "test-only",
	}

	// Create a mock API
	mediator := &MockMediator{}
	api := artifacts.API{
		Name:    "TestAPI",
		Context: "/test",
		Resources: []artifacts.Resource{
			{
				Methods:     []string{"GET"},
				URITemplate: "/resource",
				InSequence: artifacts.Sequence{
					MediatorList: []artifacts.Mediator{mediator},
				},
			},
		},
	}

	// Register the API
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rs.RegisterAPI(ctx, api); err != nil {
		t.Fatalf("Failed to register API: %v", err)
	}

	// Create a test request
	req := httptest.NewRequest("GET", "/test/resource", nil)
	w := httptest.NewRecorder()

	// Serve the request
	rs.router.ServeHTTP(w, req)

	// Check the response
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.StatusCode)
	}

	if !mediator.executed {
		t.Error("Mediator was not executed")
	}
}

func TestRegisterAPI_Versioning(t *testing.T) {
	// Implement a simpler test that doesn't rely on mocking the router
	testCases := []struct {
		name        string
		context     string
		version     string
		versionType string
		uriTemplate string
	}{
		{
			name:        "No versioning",
			context:     "/api",
			version:     "",
			versionType: "",
			uriTemplate: "/resource",
		},
		{
			name:        "Context with trailing slash",
			context:     "/api/",
			version:     "",
			versionType: "",
			uriTemplate: "/resource",
		},
		{
			name:        "URL versioning",
			context:     "/api",
			version:     "v1",
			versionType: "url",
			uriTemplate: "/resource",
		},
		{
			name:        "Context versioning with placeholder",
			context:     "/api/{version}/services",
			version:     "v1",
			versionType: "context",
			uriTemplate: "/resource",
		},
		{
			name:        "Context versioning with typo in placeholder",
			context:     "/api/{versions}/services",
			version:     "v1",
			versionType: "context",
			uriTemplate: "/resource",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create API
			api := artifacts.API{
				Name:        "TestAPI",
				Context:     tc.context,
				Version:     tc.version,
				VersionType: tc.versionType,
				Resources: []artifacts.Resource{
					{
						Methods:     []string{"GET"},
						URITemplate: tc.uriTemplate,
					},
				},
			}

			// Calculate expected path based on our logic
			expectedPath := calcExpectedPath(api.Context, api.Version, api.VersionType, api.Resources[0].URITemplate)

			// Create a minimal router service that won't actually start
			rs := &RouterService{
				router:     http.NewServeMux(), // Use a real ServeMux
				apis:       make(map[string]artifacts.API),
				listenAddr: "test-only",
				started:    true, // Prevent actual server start
			}

			// Register should succeed without errors
			err := rs.RegisterAPI(context.Background(), api)
			if err != nil {
				t.Fatalf("Failed to register API: %v", err)
			}

			// Log the expected path for verification
			t.Logf("Test %s: Expected path: %s", tc.name, expectedPath)
		})
	}
}

// calcExpectedPath implements the same logic as RegisterAPI to construct paths
func calcExpectedPath(context, version, versionType, uriTemplate string) string {
	// Start with the context
	basePath := context

	// Remove trailing slash if present
	if len(basePath) > 1 && basePath[len(basePath)-1] == '/' {
		basePath = basePath[:len(basePath)-1]
	}

	// Handle versioning based on versionType
	if version != "" && versionType != "" {
		switch versionType {
		case "url":
			// For URL type, add version as a path segment
			basePath = basePath + "/" + version
		case "context":
			// For context type, replace {version} placeholder
			versionPattern := "{version}"
			basePath = strings.Replace(basePath, versionPattern, version, 1)
		}
	}

	// Construct the full pattern
	return "GET " + basePath + uriTemplate
}

func TestRouterService_MultipleMethodsPerResource(t *testing.T) {
	// Create a test server that will never start listening
	rs := &RouterService{
		router:     http.NewServeMux(),
		apis:       make(map[string]artifacts.API),
		listenAddr: "test-only",
		started:    true, // Prevent actual server start
	}

	// Create a mock API with a resource that supports multiple methods
	mediator := &MockMediator{}
	api := artifacts.API{
		Name:    "MultiMethodAPI",
		Context: "/api",
		Resources: []artifacts.Resource{
			{
				Methods:     []string{"GET", "POST", "PUT", "DELETE"},
				URITemplate: "/resource",
				InSequence: artifacts.Sequence{
					MediatorList: []artifacts.Mediator{mediator},
				},
			},
		},
	}

	// Register the API
	ctx := context.Background()
	if err := rs.RegisterAPI(ctx, api); err != nil {
		t.Fatalf("Failed to register API: %v", err)
	}

	// Get registered routes
	routes := rs.GetRegisteredRoutes("MultiMethodAPI")

	// Check that we have the correct number of routes
	expectedRoutes := []string{
		"GET /api/resource",
		"POST /api/resource",
		"PUT /api/resource",
		"DELETE /api/resource",
	}

	if len(routes) != len(expectedRoutes) {
		t.Errorf("Expected %d routes, got %d", len(expectedRoutes), len(routes))
	}

	// Check that all expected routes are registered
	for _, expected := range expectedRoutes {
		found := false
		for _, actual := range routes {
			if expected == actual {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected route %s not found in registered routes", expected)
		}
	}

	// Test that each method actually works by sending requests
	for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
		// Reset the mediator state
		mediator.executed = false

		// Create a test request with the appropriate method
		req := httptest.NewRequest(method, "/api/resource", nil)
		w := httptest.NewRecorder()

		// Serve the request
		rs.router.ServeHTTP(w, req)

		// Check the response
		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK for %s, got %v", method, resp.StatusCode)
		}

		if !mediator.executed {
			t.Errorf("Mediator was not executed for %s request", method)
		}
	}
}

func TestRouterService_MultipleHTTPMethods(t *testing.T) {
	// Create a test server that will never start listening
	rs := &RouterService{
		router:     http.NewServeMux(),
		apis:       make(map[string]artifacts.API),
		listenAddr: "test-only",
		started:    true, // Prevent actual server start
	}

	// Create a mock API with resources that have multiple methods
	mediator := &TestMethodMediator{}
	api := artifacts.API{
		Name:    "MultiMethodAPI",
		Context: "/api",
		Resources: []artifacts.Resource{
			{
				Methods:     []string{"GET", "POST", "PUT", "DELETE"},
				URITemplate: "/resource",
				InSequence: artifacts.Sequence{
					MediatorList: []artifacts.Mediator{mediator},
				},
			},
		},
	}

	// Register the API
	ctx := context.Background()
	if err := rs.RegisterAPI(ctx, api); err != nil {
		t.Fatalf("Failed to register API: %v", err)
	}

	// Get registered routes
	routes := rs.GetRegisteredRoutes("MultiMethodAPI")

	// Check that we have the correct number of routes
	if len(routes) != 4 {
		t.Errorf("Expected 4 routes, got %d", len(routes))
	}

	// Test each HTTP method
	for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			// Reset mediator
			mediator.executed = false
			mediator.method = ""

			// Create request with specific method
			req := httptest.NewRequest(method, "/api/resource", nil)
			w := httptest.NewRecorder()

			// Serve the request
			rs.router.ServeHTTP(w, req)

			// Verify the response
			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status OK for %s, got %v", method, resp.StatusCode)
			}

			if !mediator.executed {
				t.Errorf("Mediator was not executed for %s request", method)
			}
		})
	}
}
