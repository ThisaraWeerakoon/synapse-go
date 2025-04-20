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
	"fmt"
	"net/http"

	"github.com/apache/synapse-go/internal/app/core/domain"
	"github.com/apache/synapse-go/internal/app/core/ports"
	"github.com/apache/synapse-go/internal/pkg/core/synctx"
)

// FileInboundEndpoint handles file-based inbound operations
type HTTPInboundEndpoint struct {
	config          domain.InboundConfig
	mediator        ports.InboundMessageMediator
	IsRunning       bool
}

// NewHTTPInboundEndpoint creates a new HTTPInboundEndpoint instance
func NewHTTPInboundEndpoint(
	config domain.InboundConfig,
	mediator ports.InboundMessageMediator,
) *HTTPInboundEndpoint {
	return &HTTPInboundEndpoint{
		config:   config,
	}
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

	h.mediator = mediator
	h.IsRunning = true
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Create message context
		msgContext := synctx.CreateMsgContext()

		// Store the *http.Request in the message context properties.
		if msgContext.Properties == nil {
			msgContext.Properties = make(map[string]string)
		}
		//Store pointer to request as string representation
		msgContext.Properties["http_request"] = fmt.Sprintf("%v", r)

		h.mediator.MediateInboundMessage(ctx, h.config.SequenceName, msgContext)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Message received"))
	})
	port := h.config.Parameters["inbound.http.port"]
	fmt.Printf("Server starting on port %s...\n", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
		return err
	}
	return nil

}

// call this using a channel
func (adapter *HTTPInboundEndpoint) Stop() error {
	adapter.IsRunning = false
	return nil
}

