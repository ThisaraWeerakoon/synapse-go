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

package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/apache/synapse-go/internal/app/core/domain"
	"github.com/apache/synapse-go/internal/app/core/ports"
	"github.com/apache/synapse-go/internal/pkg/core/synctx"
	"github.com/apache/synapse-go/internal/pkg/loggerfactory"
)

const (
	componentName = "http"
)

// HTTPInboundEndpoint handles http-based inbound operations
type HTTPInboundEndpoint struct {
	config    domain.InboundConfig
	mediator  ports.InboundMessageMediator
	IsRunning bool
	server    *http.Server
	router    *http.ServeMux
	logger    *slog.Logger
}

// NewHTTPInboundEndpoint creates a new HTTPInboundEndpoint instance
func NewHTTPInboundEndpoint(
	config domain.InboundConfig,
	mediator ports.InboundMessageMediator,
) *HTTPInboundEndpoint {
	h := &HTTPInboundEndpoint{
		config: config,
		router: http.NewServeMux(),
	}
	h.logger = loggerfactory.GetLogger(componentName, h)
	return h
}

func (h *HTTPInboundEndpoint) Start(ctx context.Context, mediator ports.InboundMessageMediator) error {
	// Check if context is already canceled before proceeding
	select {
	case <-ctx.Done():
		// Context already canceled, don't decrement WaitGroup
		return ctx.Err()
	default:
		// Context still valid, proceed with normal operation
	}

	h.IsRunning = true
	h.mediator = mediator

	// Set up the HTTP handler for the root path
	h.router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Create message context
		msgContext := synctx.CreateMsgContext()

		// Set request into message context properties
		msgContext.Properties["http_request"] = r

		// Mediate the inbound message
		if err := h.mediator.MediateInboundMessage(ctx, h.config.SequenceName, msgContext); err != nil {
			h.logger.Error("Error mediating inbound message", "error", err)
		}
	})

	port := h.config.Parameters["inbound.http.port"]

	// Ensure the port has the proper format with colon prefix
	listenAddr := ":" + port
	if port[0] == ':' {
		listenAddr = port // Port already has colon prefix
	}

	// Create a new HTTP server
	h.server = &http.Server{
		Addr:    listenAddr,
		Handler: h.router,
	}

	// Start the server in a goroutine
	go func() {
		h.logger.Info("Starting HTTP Inbound listener", "address", listenAddr)
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			h.logger.Error("HTTP Inbound listener error", "error", err)
		}
	}()

	// Start a goroutine to monitor context cancellation and shut down server
	go func() {
		<-ctx.Done()
		h.logger.Info("Shutting down HTTP server...")
		// Shutdown the server gracefully
		if err := h.server.Shutdown(ctx); err != nil {
			h.logger.Error("Error shutting down HTTP server", "error", err.Error())
		}
	}()
	return nil
}

// call this using a channel
func (h *HTTPInboundEndpoint) Stop() error {
	h.IsRunning = false
	return nil
}

func (h *HTTPInboundEndpoint) UpdateLogger() {
	h.logger = loggerfactory.GetLogger(componentName, h)
}
