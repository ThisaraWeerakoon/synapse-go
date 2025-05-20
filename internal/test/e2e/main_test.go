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
	"testing"
)

// TestE2E is the main entry point for all end-to-end tests
func TestE2E(t *testing.T) {
	// Run all end-to-end test suites
	RunE2ETests(t,
		&HTTPInboundTestSuite{},
		&FileInboundTestSuite{},
		&LifecycleTestSuite{},
	)
}
