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

package artifacts

import (
	"bytes"
	"fmt"
	"io"

	"github.com/apache/synapse-go/internal/pkg/core/synctx"
)

type LogMediator struct {
	Category string
	Message  string
	Position Position
}

func (lm LogMediator) Execute(context *synctx.MsgContext) (bool, error) {
	// Log the message
	fmt.Println(lm.Category + " : " + lm.Message)

	// Check if http_request_body exists in properties
	if bodyObj, exists := context.Properties["http_request_body"]; exists {
		// Read the request body (io.ReadCloser)
		if requestBody, ok := bodyObj.(io.ReadCloser); ok {
			// Read the body data
			bodyBytes, err := io.ReadAll(requestBody)
			if err == nil {
				// Log the body content
				fmt.Printf("%s : HTTP Request Body: %s\n", lm.Category, string(bodyBytes))

				// Important: Create a new ReadCloser and put it back in the context
				// so other mediators can also read it
				context.Properties["http_request_body"] = io.NopCloser(bytes.NewBuffer(bodyBytes))
			} else {
				fmt.Printf("%s : Error reading request body: %v\n", lm.Category, err)
			}
		}
	}

	return true, nil
}
