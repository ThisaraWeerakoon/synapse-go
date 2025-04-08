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

	"github.com/apache/synapse-go/internal/pkg/core/artifacts"
	"github.com/apache/synapse-go/internal/pkg/core/synctx"
)

// RouterService manages API routing and server lifecycle
type RouterService struct {
	server     *http.Server
	router     *http.ServeMux
	apis       map[string]artifacts.API
	listenAddr string
	mu         sync.RWMutex
	started    bool
}

// NewRouterService creates a new router service with the given listen address
func NewRouterService(listenAddr string) *RouterService {
	return &RouterService{
		router:     http.NewServeMux(),
		apis:       make(map[string]artifacts.API),
		listenAddr: listenAddr,
		started:    false,
	}
}

// RegisterAPI registers a new API with the router service
func (rs *RouterService) RegisterAPI(ctx context.Context, api artifacts.API) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Store the API
	rs.apis[api.Name] = api

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

	// Register each resource in the API
	for _, resource := range api.Resources {
		// Construct the full pattern: "METHOD /path/to/resource"
		pattern := resource.Methods + " " + basePath + resource.URITemplate
		rs.router.HandleFunc(pattern, rs.createHandlerFunc(resource))
	}

	// Start the server if it hasn't been started yet
	if !rs.started {
		if err := rs.startServer(ctx); err != nil {
			return fmt.Errorf("failed to start server: %w", err)
		}
	}

	return nil
}

// createHandlerFunc creates an HTTP handler function for the given API resource
func (rs *RouterService) createHandlerFunc(resource artifacts.Resource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
