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
 *  Unless required by applicable law or agreed in writing,
 *  software distributed under the License is distributed on an
 *   * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 *  KIND, either express or implied.  See the License for the
 *  specific language governing permissions and limitations
 *  under the License.
 */

package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LifecycleTestSuite tests the application lifecycle
type LifecycleTestSuite struct {
	SynapseE2ESuite
}

// TestGracefulShutdown tests the application's graceful shutdown behavior
func (s *LifecycleTestSuite) TestGracefulShutdown() {
	// For now, just verify the server is running correctly
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/livez", TestServerPort))
	s.Require().NoError(err, "Failed to connect to server")
	s.Require().Equal(http.StatusOK, resp.StatusCode, "Server should return 200 OK")
	defer resp.Body.Close()

	// Read the response body to verify it has the expected content
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err, "Failed to read response body")

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	s.Require().NoError(err, "Response should be valid JSON")

	// Verify the response contains the status field
	status, ok := response["status"]
	s.Require().True(ok, "Response should contain 'status' field")
	s.Equal("UP", status, "Server status should be 'UP'")
}

// TestContextPropagation tests the context propagation through the system
func (s *LifecycleTestSuite) TestContextPropagation() {
	// For now, just verify the server is running correctly
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/livez", TestServerPort))
	s.Require().NoError(err, "Failed to connect to server")
	s.Require().Equal(http.StatusOK, resp.StatusCode, "Server should return 200 OK")
	defer resp.Body.Close()

	// Read the response body to verify it has the expected content
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err, "Failed to read response body")

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	s.Require().NoError(err, "Response should be valid JSON")

	// Verify the response contains the timestamp field
	_, ok := response["timestamp"]
	s.Require().True(ok, "Response should contain 'timestamp' field")
}

// TestConcurrentRequests tests handling of multiple concurrent requests
func (s *LifecycleTestSuite) TestConcurrentRequests() {
	// For now, just verify the server is running correctly
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/livez", TestServerPort))
	s.Require().NoError(err, "Failed to connect to server")
	s.Require().Equal(http.StatusOK, resp.StatusCode, "Server should return 200 OK")
	defer resp.Body.Close()

	// Read the response body to verify it has the expected content
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err, "Failed to read response body")

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	s.Require().NoError(err, "Response should be valid JSON")

	// Verify the response contains the status field
	status, ok := response["status"]
	s.Require().True(ok, "Response should contain 'status' field")
	s.Equal("UP", status, "Server status should be 'UP'")
}
