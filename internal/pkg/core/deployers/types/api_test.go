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
package types

import (
	"testing"

	"github.com/apache/synapse-go/internal/pkg/core/artifacts"
	"github.com/stretchr/testify/assert"
)

func TestAPI_Unmarshal(t *testing.T) {
	xmlData := `
	<api context="/test" name="TestAPI">
		<resource methods="GET" uri-template="/resource1">
			<inSequence>
				<sequence>
					<log level="full"/>
				</sequence>
			</inSequence>
			<faultSequence>
				<sequence>
					<log level="error"/>
				</sequence>
			</faultSequence>
		</resource>
	</api>`

	position := artifacts.Position{
		FileName: "testfile.xml",
		LineNo:   1,
	}

	api := &API{}
	result, err := api.Unmarshal(xmlData, position)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if result.Context != "/test" {
		t.Errorf("Expected context '/test', got %s", result.Context)
	}
	if result.Name != "TestAPI" {
		t.Errorf("Expected name 'TestAPI', got %s", result.Name)
	}
	if len(result.Resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(result.Resources))
	}

	resource := result.Resources[0]
	if resource.Methods != "GET" {
		t.Errorf("Expected methods 'GET', got %s", resource.Methods)
	}
	if resource.URITemplate != "/resource1" {
		t.Errorf("Expected uri-template '/resource1', got %s", resource.URITemplate)
	}

	assert.Equal(t, "TestAPI->/resource1->inSequence->sequence", resource.InSequence.Position.Hierarchy)
	assert.Equal(t, 4, resource.InSequence.Position.LineNo)

	logMediator := resource.InSequence.MediatorList[0].(artifacts.LogMediator)
	assert.Equal(t, "TestAPI->/resource1->inSequence->sequence->log", logMediator.Position.Hierarchy)
	assert.Equal(t, 6, logMediator.Position.LineNo)
}

func TestAPI_Unmarshal_EmptyAPI(t *testing.T) {
	xmlData := `<api context="" name=""></api>`

	position := artifacts.Position{
		FileName: "testfile.xml",
		LineNo:   1,
	}

	api := &API{}
	result, err := api.Unmarshal(xmlData, position)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if result.Context != "" {
		t.Errorf("Expected empty context, got %s", result.Context)
	}
	if result.Name != "" {
		t.Errorf("Expected empty name, got %s", result.Name)
	}
	if len(result.Resources) != 0 {
		t.Fatalf("Expected 0 resources, got %d", len(result.Resources))
	}
}

func TestAPI_UnmarshalWithNestedSequence(t *testing.T) {
	xmlData := `
	<api context="/test" name="TestAPI">
		<resource methods="GET" uri-template="/resource1">
			<inSequence>
				<sequence>
					<log level="full"/>
				</sequence>
			</inSequence>
			<faultSequence>
				<log level="error"/>
			</faultSequence>
		</resource>
	</api>`

	position := artifacts.Position{
		FileName: "testfile.xml",
		LineNo:   1,
	}

	api := &API{}
	result, err := api.Unmarshal(xmlData, position)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if result.Context != "/test" {
		t.Errorf("Expected context '/test', got %s", result.Context)
	}
	if result.Name != "TestAPI" {
		t.Errorf("Expected name 'TestAPI', got %s", result.Name)
	}
	if len(result.Resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(result.Resources))
	}

	resource := result.Resources[0]
	if resource.Methods != "GET" {
		t.Errorf("Expected methods 'GET', got %s", resource.Methods)
	}
	if resource.URITemplate != "/resource1" {
		t.Errorf("Expected uri-template '/resource1', got %s", resource.URITemplate)
	}

	// Check nested sequence in inSequence
	assert.Equal(t, "TestAPI->/resource1->inSequence->sequence", resource.InSequence.Position.Hierarchy)
	assert.Equal(t, 4, resource.InSequence.Position.LineNo)

	logMediator := resource.InSequence.MediatorList[0].(artifacts.LogMediator)
	assert.Equal(t, "TestAPI->/resource1->inSequence->sequence->log", logMediator.Position.Hierarchy)
	assert.Equal(t, 6, logMediator.Position.LineNo)

	// Check direct mediator in faultSequence
	assert.Equal(t, "TestAPI->/resource1->faultSequence", resource.FaultSequence.Position.Hierarchy)
	assert.Equal(t, 9, resource.FaultSequence.Position.LineNo)

	faultLogMediator := resource.FaultSequence.MediatorList[0].(artifacts.LogMediator)
	assert.Equal(t, "TestAPI->/resource1->faultSequence->log", faultLogMediator.Position.Hierarchy)
	assert.Equal(t, 9, faultLogMediator.Position.LineNo)
}
