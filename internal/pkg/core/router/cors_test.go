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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/apache/synapse-go/internal/pkg/core/artifacts"
	"github.com/stretchr/testify/assert"
)

// createTestHandler creates a simple HTTP handler for testing
func createTestHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})
}

// TestCORSMiddleware_WithDisabledCORS tests the middleware behavior when CORS is disabled
func TestCORSMiddleware_WithDisabledCORS(t *testing.T) {
	// Create config with CORS disabled
	config := artifacts.DefaultCORSConfig()
	config.Enabled = false

	// Create test handler and wrap with middleware
	testHandler := createTestHandler()
	handler := CORSMiddleware(testHandler, config)

	// Create a test request with Origin header
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8290", nil)
	req.Header.Set("Origin", "http://client.example.com")

	// Create response recorder
	resp := httptest.NewRecorder()

	// Serve the request
	handler.ServeHTTP(resp, req)

	// Verify response has no CORS headers and original handler was called
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "test response", resp.Body.String())
	assert.Equal(t, "", resp.Header().Get("Access-Control-Allow-Origin"))
}

// TestCORSMiddleware_WithAllowAllOrigins tests wildcard origin configuration
func TestCORSMiddleware_WithAllowAllOrigins(t *testing.T) {
	// Create config with CORS enabled and all origins allowed
	config := artifacts.CORSConfig{
		Enabled:       true,
		AllowOrigins:  []string{"*"},
		AllowMethods:  []string{"GET", "POST", "PUT"},
		AllowHeaders:  []string{"Content-Type", "Authorization"},
		ExposeHeaders: []string{"X-Custom-Header"},
	}

	// Create test handler and wrap with middleware
	testHandler := createTestHandler()
	handler := CORSMiddleware(testHandler, config)

	// Create a test request with Origin header
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8290", nil)
	req.Header.Set("Origin", "http://client.example.com")

	// Create response recorder
	resp := httptest.NewRecorder()

	// Serve the request
	handler.ServeHTTP(resp, req)

	// Verify CORS headers are set correctly
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "test response", resp.Body.String())
	assert.Equal(t, "*", resp.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "X-Custom-Header", resp.Header().Get("Access-Control-Expose-Headers"))
	assert.Equal(t, "", resp.Header().Get("Access-Control-Allow-Credentials"))
}

// TestCORSMiddleware_WithSpecificOrigins tests specific allowed origins
func TestCORSMiddleware_WithSpecificOrigins(t *testing.T) {
	testCases := []struct {
		name               string
		allowedOrigins     []string
		requestOrigin      string
		allowCredentials   bool
		expectedStatusCode int
		expectedOrigin     string
	}{
		{
			name:               "Allowed specific origin",
			allowedOrigins:     []string{"http://allowed.example.com"},
			requestOrigin:      "http://allowed.example.com",
			allowCredentials:   false,
			expectedStatusCode: http.StatusOK,
			expectedOrigin:     "http://allowed.example.com",
		},
		{
			name:               "Disallowed specific origin",
			allowedOrigins:     []string{"http://allowed.example.com"},
			requestOrigin:      "http://disallowed.example.com",
			allowCredentials:   false,
			expectedStatusCode: http.StatusForbidden,
			expectedOrigin:     "",
		},
		{
			name:               "Multiple allowed origins - match",
			allowedOrigins:     []string{"http://first.example.com", "http://second.example.com"},
			requestOrigin:      "http://second.example.com",
			allowCredentials:   false,
			expectedStatusCode: http.StatusOK,
			expectedOrigin:     "http://second.example.com",
		},
		{
			name:               "With credentials - specific origin",
			allowedOrigins:     []string{"http://allowed.example.com"},
			requestOrigin:      "http://allowed.example.com",
			allowCredentials:   true,
			expectedStatusCode: http.StatusOK,
			expectedOrigin:     "http://allowed.example.com", // Must be specific with credentials
		},
		{
			name:               "With wildcard and credentials",
			allowedOrigins:     []string{"*"},
			requestOrigin:      "http://any.example.com",
			allowCredentials:   true,
			expectedStatusCode: http.StatusOK,
			expectedOrigin:     "http://any.example.com", // Must be specific with credentials
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create config
			config := artifacts.CORSConfig{
				Enabled:          true,
				AllowOrigins:     tc.allowedOrigins,
				AllowMethods:     []string{"GET", "POST"},
				AllowHeaders:     []string{"Content-Type"},
				AllowCredentials: tc.allowCredentials,
			}

			// Create test handler and wrap with middleware
			testHandler := createTestHandler()
			handler := CORSMiddleware(testHandler, config)

			// Create a test request with Origin header
			req := httptest.NewRequest(http.MethodGet, "http://localhost:8290", nil)
			req.Header.Set("Origin", tc.requestOrigin)

			// Create response recorder
			resp := httptest.NewRecorder()

			// Serve the request
			handler.ServeHTTP(resp, req)

			// Verify response
			assert.Equal(t, tc.expectedStatusCode, resp.Code)

			if tc.expectedStatusCode == http.StatusOK {
				assert.Equal(t, tc.expectedOrigin, resp.Header().Get("Access-Control-Allow-Origin"))

				if tc.allowCredentials {
					assert.Equal(t, "true", resp.Header().Get("Access-Control-Allow-Credentials"))
				}
			}
		})
	}
}

// TestCORSMiddleware_PreflightRequest tests OPTIONS preflight requests
func TestCORSMiddleware_PreflightRequest(t *testing.T) {
	// Create CORS config with typical values
	config := artifacts.CORSConfig{
		Enabled:          true,
		AllowOrigins:     []string{"http://client.example.com"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"X-Response-Time"},
		AllowCredentials: true,
		MaxAge:           3600,
	}

	// Create test handler and wrap with middleware
	testHandler := createTestHandler()
	handler := CORSMiddleware(testHandler, config)

	// Create a preflight OPTIONS request
	req := httptest.NewRequest(http.MethodOptions, "http://localhost:8290/api", nil)
	req.Header.Set("Origin", "http://client.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")

	// Create response recorder
	resp := httptest.NewRecorder()

	// Serve the request
	handler.ServeHTTP(resp, req)

	// Verify preflight response
	assert.Equal(t, http.StatusNoContent, resp.Code)
	assert.Empty(t, resp.Body.String()) // No body for preflight response

	// Check CORS headers
	assert.Equal(t, "http://client.example.com", resp.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, DELETE", resp.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type, Authorization", resp.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "3600", resp.Header().Get("Access-Control-Max-Age"))
	assert.Equal(t, "true", resp.Header().Get("Access-Control-Allow-Credentials"))
}

// TestCORSMiddleware_RegularRequestWithExposedHeaders tests exposed headers in regular requests
func TestCORSMiddleware_RegularRequestWithExposedHeaders(t *testing.T) {
	// Create CORS config with exposed headers
	config := artifacts.CORSConfig{
		Enabled:       true,
		AllowOrigins:  []string{"http://client.example.com"},
		ExposeHeaders: []string{"X-Response-Time", "X-Request-ID"},
	}

	// Create test handler and wrap with middleware
	testHandler := createTestHandler()
	handler := CORSMiddleware(testHandler, config)

	// Create a regular GET request
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8290/api", nil)
	req.Header.Set("Origin", "http://client.example.com")

	// Create response recorder
	resp := httptest.NewRecorder()

	// Serve the request
	handler.ServeHTTP(resp, req)

	// Verify response
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "test response", resp.Body.String())

	// Check exposed headers
	assert.Equal(t, "X-Response-Time, X-Request-ID", resp.Header().Get("Access-Control-Expose-Headers"))
}

// TestCreateOptionsHandler tests the dedicated OPTIONS handler
func TestCreateOptionsHandler(t *testing.T) {
	testCases := []struct {
		name            string
		config          artifacts.CORSConfig
		methods         []string
		requestOrigin   string
		expectedStatus  int
		expectedHeaders map[string]string
	}{
		{
			name: "Standard preflight",
			config: artifacts.CORSConfig{
				Enabled:      true,
				AllowOrigins: []string{"http://client.example.com"},
				AllowHeaders: []string{"Content-Type", "Authorization"},
				MaxAge:       3600,
			},
			methods:        []string{"GET", "POST"},
			requestOrigin:  "http://client.example.com",
			expectedStatus: http.StatusNoContent,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "http://client.example.com",
				"Access-Control-Allow-Methods": "GET, POST",
				"Access-Control-Allow-Headers": "Content-Type, Authorization",
				"Access-Control-Max-Age":       "3600",
			},
		},
		{
			name: "CORS disabled",
			config: artifacts.CORSConfig{
				Enabled: false,
			},
			methods:        []string{"GET", "POST"},
			requestOrigin:  "http://client.example.com",
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin": "",
			},
		},
		{
			name: "No origin",
			config: artifacts.CORSConfig{
				Enabled:      true,
				AllowOrigins: []string{"http://client.example.com"},
			},
			methods:        []string{"GET", "POST"},
			requestOrigin:  "", // No origin
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin": "",
			},
		},
		{
			name: "Disallowed origin",
			config: artifacts.CORSConfig{
				Enabled:      true,
				AllowOrigins: []string{"http://allowed.example.com"},
			},
			methods:        []string{"GET", "POST"},
			requestOrigin:  "http://disallowed.example.com",
			expectedStatus: http.StatusForbidden,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin": "",
			},
		},
		{
			name: "With credentials",
			config: artifacts.CORSConfig{
				Enabled:          true,
				AllowOrigins:     []string{"http://client.example.com"},
				AllowCredentials: true,
			},
			methods:        []string{"GET", "POST"},
			requestOrigin:  "http://client.example.com",
			expectedStatus: http.StatusNoContent,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":      "http://client.example.com",
				"Access-Control-Allow-Methods":     "GET, POST",
				"Access-Control-Allow-Credentials": "true",
			},
		},
		{
			name: "With wildcard origin",
			config: artifacts.CORSConfig{
				Enabled:      true,
				AllowOrigins: []string{"*"},
			},
			methods:        []string{"GET", "POST"},
			requestOrigin:  "http://client.example.com",
			expectedStatus: http.StatusNoContent,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET, POST",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create handler
			handler := CreateOptionsHandler(tc.methods, tc.config)

			// Create request
			req := httptest.NewRequest(http.MethodOptions, "http://localhost:8290/api", nil)
			if tc.requestOrigin != "" {
				req.Header.Set("Origin", tc.requestOrigin)
			}

			// Create response recorder
			resp := httptest.NewRecorder()

			// Serve the request
			handler(resp, req)

			// Verify response
			assert.Equal(t, tc.expectedStatus, resp.Code)

			// Check headers
			for name, value := range tc.expectedHeaders {
				if value == "" {
					assert.Equal(t, "", resp.Header().Get(name))
				} else {
					assert.Equal(t, value, resp.Header().Get(name))
				}
			}
		})
	}
}
