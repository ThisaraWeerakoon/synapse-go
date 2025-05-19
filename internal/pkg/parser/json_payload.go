package parser

import (
	"fmt"
	"github.com/tidwall/gjson"
)

// JSONPayload handles JSON data.
type JSONPayload struct {
	rawContent  []byte
	jsonResult  gjson.Result // Store the parsed gjson.Result
	contentType string
}

// NewJSONPayload creates a new JSONPayload.
// gjson parses on demand, so no explicit parsing step here for the whole doc,
// but we validate it.
func NewJSONPayload(content []byte) (*JSONPayload, error) {
	if !gjson.ValidBytes(content) {
		return nil, &ErrEvaluationFailed{Reason: "Invalid JSON content"}
	}
	return &JSONPayload{
		rawContent:  content,
		jsonResult:  gjson.ParseBytes(content), // Parse it once
		contentType: "application/json",
	}, nil
}

func (jp *JSONPayload) GetRawBytes() []byte {
	return jp.rawContent
}

func (jp *JSONPayload) GetContentType() string {
	return jp.contentType
}

// Query evaluates a JSONPath expression (simplified to gjson paths) against the JSON payload.
func (jp *JSONPayload) Query(expression string) (QueryResult, error) {
	// gjson.Path directly uses the raw JSON string/bytes.
	// result := gjson.GetBytes(jp.rawContent, expression)
	result := jp.jsonResult.Get(expression) // Use the parsed result

	if !result.Exists() {
		// Check if the path was intended to return a null that exists vs a path that doesn't exist
		// gjson distinction: result.Type == gjson.Null vs !result.Exists()
		// For simplicity, if it doesn't exist, we treat it as "not found".
		return QueryResult{Value: nil, Type: UnknownResult}, &ErrEvaluationFailed{Expression: expression, Reason: "path not found or value does not exist"}
	}

	var qr QueryResult
	switch result.Type {
	case gjson.String:
		qr = QueryResult{Value: result.String(), Type: StringResult}
	case gjson.Number:
		qr = QueryResult{Value: result.Float(), Type: NumberResult}
	case gjson.True, gjson.False:
		qr = QueryResult{Value: result.Bool(), Type: BooleanResult}
	case gjson.JSON: // This means it's an object or array
		if result.IsArray() {
			var arr []interface{}
			result.ForEach(func(key, value gjson.Result) bool {
				arr = append(arr, value.Value()) // gjson.Result.Value() gives basic types
				return true
			})
			qr = QueryResult{Value: arr, Type: ArrayResult}
		} else if result.IsObject() {
			// Convert map[string]gjson.Result to map[string]interface{}
			objMap := make(map[string]interface{})
			result.ForEach(func(key, value gjson.Result) bool {
				objMap[key.String()] = value.Value()
				return true
			})
			qr = QueryResult{Value: objMap, Type: ObjectResult}
		} else {
			// Should not happen if IsArray/IsObject are comprehensive
			qr = QueryResult{Value: result.Raw, Type: UnknownResult} // Fallback to raw
		}
	case gjson.Null:
		qr = QueryResult{Value: nil, Type: ScalarResult} // Or a specific NullResult type
	default: // Should not be reached if gjson types are handled
		return QueryResult{}, &ErrEvaluationFailed{Expression: expression, Reason: fmt.Sprintf("unexpected gjson result type: %s", result.Type.String())}
	}
	return qr, nil
}

func (jp *JSONPayload) AsString() (string, error) {
	return string(jp.rawContent), nil
}

func (jp *JSONPayload) GetUnderlying() interface{} {
	return jp.jsonResult // Return the gjson.Result
}
