package parser

import (
	"fmt"
	"strings"
)

const (
	xpathPrefix      = "xpath:"
	jsonpathPrefix   = "jsonpath:"
	extractAsJSONPipe = "extractAsJSON"
	extractAsXMLPipe  = "extractAsXML"
)

// ExpressionEngine parses and evaluates expressions against payloads.
type ExpressionEngine struct {
	// Potentially cache compiled expressions if expressions are often reused
	// For PoC, we re-evaluate prefixes each time.
	payloadFactory *PayloadFactory // To create intermediate payloads for mixed content
}

func NewEngine() *ExpressionEngine {
	return &ExpressionEngine{
		payloadFactory: NewPayloadFactory(),
	}
}

// Evaluate processes the full expression string, handling prefixes and pipes.
func (ee *ExpressionEngine) Evaluate(currentPayload PayloadObject, fullExpression string) (QueryResult, error) {
	parts := strings.Split(fullExpression, "|")
	var currentResult QueryResult
	var err error
	
	// Initial payload for the first part of the expression
	activePayload := currentPayload 

	for i, part := range parts {
		trimmedPart := strings.TrimSpace(part)
		if i == 0 { // First part is always an expression
			currentResult, err = ee.evaluateSingleExpression(activePayload, trimmedPart)
			if err != nil {
				return QueryResult{}, fmt.Errorf("error in expression part '%s': %w", trimmedPart, err)
			}
		} else { // Subsequent parts are transformations or chained expressions
			// Ensure previous result was a string to be re-parsed
			prevResultStr, ok := currentResult.Value.(string)
			if !ok {
				return QueryResult{}, &ErrEvaluationFailed{
					Expression: fullExpression,
					Reason:     fmt.Sprintf("pipe operation '%s' requires string input from previous step, got %T", trimmedPart, currentResult.Value),
				}
			}

			pipeAndNextExpr := strings.Fields(trimmedPart) // Simple split by space
			if len(pipeAndNextExpr) < 2 {
				return QueryResult{}, &ErrEvaluationFailed{Expression: fullExpression, Reason: fmt.Sprintf("invalid pipe operation format: %s", trimmedPart)}
			}
			
			pipeOperation := pipeAndNextExpr[0]
			nextExpression := strings.Join(pipeAndNextExpr[1:], " ")


			switch pipeOperation {
			case "extractAsJSON":
				// Create a new JSONPayload from the string result of the previous step
				intermediatePayload, err := ee.payloadFactory.CreatePayload([]byte(prevResultStr), "application/json")
				if err != nil {
					return QueryResult{}, &ErrEvaluationFailed{
						Expression: fullExpression,
						Reason:     fmt.Sprintf("failed to create intermediate JSON payload for pipe '%s'", pipeOperation),
						InnerError: err,
					}
				}
				activePayload = intermediatePayload
			case "extractAsXML":
				// Create a new XMLPayload from the string result
				intermediatePayload, err := ee.payloadFactory.CreatePayload([]byte(prevResultStr), "application/xml")
				if err != nil {
					return QueryResult{}, &ErrEvaluationFailed{
						Expression: fullExpression,
						Reason:     fmt.Sprintf("failed to create intermediate XML payload for pipe '%s'", pipeOperation),
						InnerError: err,
					}
				}
				activePayload = intermediatePayload
			default:
				return QueryResult{}, &ErrUnsupportedExpression{Expression: fmt.Sprintf("unknown pipe operation: %s", pipeOperation)}
			}
			
			// Evaluate the next part of the expression on the new intermediate payload
			currentResult, err = ee.evaluateSingleExpression(activePayload, nextExpression)
			if err != nil {
				return QueryResult{}, fmt.Errorf("error in chained expression part '%s' after pipe '%s': %w", nextExpression, pipeOperation, err)
			}
		}
	}
	return currentResult, nil
}


// evaluateSingleExpression evaluates a simple, non-piped expression part.
func (ee *ExpressionEngine) evaluateSingleExpression(pld PayloadObject, expressionPart string) (QueryResult, error) {
	if strings.HasPrefix(expressionPart, xpathPrefix) {
		if pld.GetContentType() != "application/xml" && pld.GetContentType() != "text/xml" {
			return QueryResult{}, &ErrInvalidPayloadForOperation{Operation: "XPath", PayloadType: pld.GetContentType(), Reason: "XPath requires XML payload"}
		}
		actualExpr := strings.TrimPrefix(expressionPart, xpathPrefix)
		return pld.Query(actualExpr)
	} else if strings.HasPrefix(expressionPart, jsonpathPrefix) {
		if pld.GetContentType() != "application/json" {
				return QueryResult{}, &ErrInvalidPayloadForOperation{Operation: "JSONPath", PayloadType: pld.GetContentType(), Reason: "JSONPath requires JSON payload"}
		}
		actualExpr := strings.TrimPrefix(expressionPart, jsonpathPrefix)
		return pld.Query(actualExpr)
	}
	// Add other expression types (regex, etc.) here
	return QueryResult{}, &ErrUnsupportedExpression{Expression: expressionPart}
}
