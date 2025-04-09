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
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/apache/synapse-go/internal/pkg/core/artifacts"
)

// CORSMiddleware applies CORS headers based on the provided configuration
func CORSMiddleware(handler http.Handler, config artifacts.CORSConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip CORS handling if disabled
		if !config.Enabled {
			handler.ServeHTTP(w, r)
			return
		}

		origin := r.Header.Get("Origin")

		// If no Origin is present, continue as normal
		if origin == "" {
			handler.ServeHTTP(w, r)
			return
		}

		// Check if the origin is allowed
		if !config.IsOriginAllowed(origin) {
			http.Error(w, "CORS: Origin not allowed", http.StatusForbidden)
			return
		}

		// Set CORS headers
		// Always use the actual origin (not "*") when credentials are allowed
		if config.AllowCredentials && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			// If AllowOrigins contains "*", use "*", otherwise use the specific origin
			if slices.Contains(config.AllowOrigins, "*") {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
		}

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			// Set allowed methods
			if len(config.AllowMethods) > 0 {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
			}

			// Set allowed headers - if client requested specific headers, respond to those
			requestHeaders := r.Header.Get("Access-Control-Request-Headers")
			if requestHeaders != "" {
				w.Header().Set("Access-Control-Allow-Headers", requestHeaders)
			} else if len(config.AllowHeaders) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
			}

			// Set max age
			if config.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
			}

			// Set allow credentials
			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Respond to preflight with 204 No Content
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// For actual requests, set expose headers if any
		if len(config.ExposeHeaders) > 0 {
			w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))
		}

		// Set allow credentials
		if config.AllowCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Continue to the actual handler
		handler.ServeHTTP(w, r)
	})
}

// CreateOptionsHandler creates a handler for OPTIONS requests
func CreateOptionsHandler(methods []string, config artifacts.CORSConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If CORS is not enabled, just respond with 200 OK
		if !config.Enabled {
			w.WriteHeader(http.StatusOK)
			return
		}

		origin := r.Header.Get("Origin")
		if origin == "" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Check if the origin is allowed
		if !config.IsOriginAllowed(origin) {
			http.Error(w, "CORS: Origin not allowed", http.StatusForbidden)
			return
		}

		// Set Access-Control-Allow-Origin
		if config.AllowCredentials && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			if slices.Contains(config.AllowOrigins, "*") {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
		}

		// Set Access-Control-Allow-Methods
		if len(methods) > 0 {
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ", "))
		} else if len(config.AllowMethods) > 0 {
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
		}

		// Set Access-Control-Allow-Headers
		requestHeaders := r.Header.Get("Access-Control-Request-Headers")
		if requestHeaders != "" {
			w.Header().Set("Access-Control-Allow-Headers", requestHeaders)
		} else if len(config.AllowHeaders) > 0 {
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
		}

		// Set Access-Control-Max-Age
		if config.MaxAge > 0 {
			w.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
		}

		// Set Access-Control-Allow-Credentials
		if config.AllowCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Response to preflight with 204 No Content
		w.WriteHeader(http.StatusNoContent)
	}
}

// Log CORS configuration for debugging
func LogCORSConfig(config artifacts.CORSConfig) {
	if !config.Enabled {
		fmt.Println("CORS is disabled for this API")
		return
	}

	fmt.Println("CORS Configuration:")
	fmt.Println("  Enabled:", config.Enabled)
	fmt.Println("  Allow Origins:", strings.Join(config.AllowOrigins, ", "))
	fmt.Println("  Allow Methods:", strings.Join(config.AllowMethods, ", "))
	fmt.Println("  Allow Headers:", strings.Join(config.AllowHeaders, ", "))
	fmt.Println("  Expose Headers:", strings.Join(config.ExposeHeaders, ", "))
	fmt.Println("  Allow Credentials:", config.AllowCredentials)
	fmt.Println("  Max Age:", config.MaxAge)
}
