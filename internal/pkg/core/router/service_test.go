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
				Methods:     "GET",
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
