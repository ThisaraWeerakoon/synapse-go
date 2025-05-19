package parser

// ResultType defines the type of the query result.
type ResultType string

const (
	ScalarResult  ResultType = "scalar"  // Single value (string, number, boolean)
	NodeSetResult ResultType = "nodeset" // A collection of nodes (e.g., from XPath)
	ObjectResult  ResultType = "object"  // A JSON object
	ArrayResult   ResultType = "array"   // A JSON array
	StringResult  ResultType = "string"
	BooleanResult ResultType = "boolean"
	NumberResult  ResultType = "number"
	UnknownResult ResultType = "unknown"
)

// QueryResult holds the outcome of an expression evaluation.
type QueryResult struct {
	Value interface{} // Can be string, float64, bool, []interface{}, map[string]interface{}, or a custom Node type
	Type  ResultType  // Type of the result
}

// PayloadObject is the interface for different payload types (XML, JSON, etc.).
type PayloadObject interface {
	GetRawBytes() []byte
	GetContentType() string
	Query(expression string) (QueryResult, error)
	AsString() (string, error)      // Get the whole payload as a string
	GetUnderlying() interface{} // Access to the raw parsed object (e.g., *xmlquery.Node, gjson.Result)
}
