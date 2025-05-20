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

// Package e2e contains end-to-end tests for the Synapse Go application
package e2e

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/apache/synapse-go/internal/app/synapse"
	"github.com/stretchr/testify/suite"
)

// Define constants used throughout the test suite
const (
	BaseURL        = "http://localhost:8290"
	TestServerPort = 8390 // 8290 + 100 (default offset)
	TestTimeout    = 30 * time.Second
)

// SynapseE2ESuite is the base test suite for all e2e tests
type SynapseE2ESuite struct {
	suite.Suite
	AppCtx        context.Context
	AppCancel     context.CancelFunc
	AppWaitGroup  *sync.WaitGroup
	TempDir       string
	ArtifactsPath string
	ConfigPath    string
}

// SetupSuite prepares the test environment for all tests
func (s *SynapseE2ESuite) SetupSuite() {
	// Create temp directory for test artifacts
	tempDir, err := os.MkdirTemp("", "synapse-e2e-test-*")
	s.Require().NoError(err, "Failed to create temp directory")
	s.TempDir = tempDir

	// Create necessary directory structure
	s.ArtifactsPath = filepath.Join(s.TempDir, "artifacts")
	s.ConfigPath = filepath.Join(s.TempDir, "conf")

	dirs := []string{
		s.ArtifactsPath,
		filepath.Join(s.ArtifactsPath, "APIs"),
		filepath.Join(s.ArtifactsPath, "Endpoints"),
		filepath.Join(s.ArtifactsPath, "Inbounds"),
		filepath.Join(s.ArtifactsPath, "Sequences"),
		s.ConfigPath,
	}

	for _, dir := range dirs {
		s.Require().NoError(os.MkdirAll(dir, 0755), "Failed to create directory: "+dir)
	}

	// Create basic configuration files
	s.createDeploymentConfig()
	s.createLoggerConfig()

	// Start the application with the test configuration
	s.AppWaitGroup = &sync.WaitGroup{}
	s.AppCtx, s.AppCancel = context.WithCancel(context.Background())

	// Set environment variables for testing
	os.Setenv("SYNAPSE_HOME", s.TempDir)
	os.Setenv("SYNAPSE_CONF_PATH", s.ConfigPath) // Add explicit configuration path

	// Start the application in a goroutine
	s.AppWaitGroup.Add(1)
	go func() {
		defer s.AppWaitGroup.Done()
		if err := synapse.Run(s.AppCtx); err != nil {
			fmt.Printf("ERROR: Application failed to run: %v\n", err)
		}
	}()

	// Wait for application to start
	s.waitForAppToStart()
}

// TearDownSuite cleans up the test environment after all tests
func (s *SynapseE2ESuite) TearDownSuite() {
	// Cancel the application context
	if s.AppCancel != nil {
		s.AppCancel()
	}

	// Wait for application to shut down
	if s.AppWaitGroup != nil {
		s.AppWaitGroup.Wait()
	}

	// Remove temp directory
	if s.TempDir != "" {
		os.RemoveAll(s.TempDir)
	}

	// Unset environment variables
	os.Unsetenv("SYNAPSE_HOME")
	os.Unsetenv("SYNAPSE_CONF_PATH")
}

// Helper function to create the deployment.toml file
func (s *SynapseE2ESuite) createDeploymentConfig() {
	content := `[server]
hostname = "localhost"
offset = "100"
`
	err := os.WriteFile(filepath.Join(s.ConfigPath, "deployment.toml"), []byte(content), 0644)
	s.Require().NoError(err, "Failed to create deployment.toml")
}

// Helper function to create the LoggerConfig.toml file
func (s *SynapseE2ESuite) createLoggerConfig() {
	content := `[logger]
level.default = "info"

[logger.level.packages]
mediation = "info"
router = "info"
deployers = "info"
synapse = "info"
default = "info"

[logger.handler]
format = "text"
outputPath = "stdout"
`
	err := os.WriteFile(filepath.Join(s.ConfigPath, "LoggerConfig.toml"), []byte(content), 0644)
	s.Require().NoError(err, "Failed to create LoggerConfig.toml")
}

// Helper function to wait for the application to start
func (s *SynapseE2ESuite) waitForAppToStart() {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	maxRetries := 10
	retryDelay := 500 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		time.Sleep(retryDelay)
		resp, err := client.Get(fmt.Sprintf("http://localhost:%d/livez", TestServerPort))
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	s.FailNow("Application failed to start within the expected time")
}

// Helper function to deploy a test artifact
func (s *SynapseE2ESuite) deployArtifact(artifactType, name, content string) {
	var dir string
	switch artifactType {
	case "API":
		dir = filepath.Join(s.ArtifactsPath, "APIs")
	case "Endpoint":
		dir = filepath.Join(s.ArtifactsPath, "Endpoints")
	case "Inbound":
		dir = filepath.Join(s.ArtifactsPath, "Inbounds")
	case "Sequence":
		dir = filepath.Join(s.ArtifactsPath, "Sequences")
	default:
		s.FailNow("Invalid artifact type: " + artifactType)
	}

	filename := filepath.Join(dir, name+".xml")
	err := os.WriteFile(filename, []byte(content), 0644)
	s.Require().NoError(err, "Failed to write artifact file: "+filename)

	// Give some time for the artifact to be deployed
	time.Sleep(1 * time.Second)
}

// Helper function to make HTTP requests
func (s *SynapseE2ESuite) makeHTTPRequest(method, url string, body io.Reader) (*http.Response, error) {
	client := &http.Client{
		Timeout: TestTimeout,
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if method == http.MethodPost || method == http.MethodPut {
		req.Header.Set("Content-Type", "application/json")
	}

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	req = req.WithContext(ctx)
	return client.Do(req)
}

// CreateTestFile creates a file in the specified directory with the given content
func (s *SynapseE2ESuite) CreateTestFile(dir, filename, content string) string {
	fullPath := filepath.Join(dir, filename)
	err := os.WriteFile(fullPath, []byte(content), 0644)
	s.Require().NoError(err, "Failed to create test file: "+fullPath)
	return fullPath
}

// RunE2ETests runs the test suite with the provided test cases
func RunE2ETests(t *testing.T, testCases ...suite.TestingSuite) {
	for _, testCase := range testCases {
		suite.Run(t, testCase)
	}
}
