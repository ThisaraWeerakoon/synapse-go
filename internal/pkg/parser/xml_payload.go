package parser

import (
	"bytes"
	"fmt"
	// Using antchfx/xpath as it's a common choice.
	// xmlquery is based on antchfx/xpath and provides a slightly higher-level API.
	// For direct XPath 1.0, antchfx/xpath is fine.
	"github.com/antchfx/xpath"
	"github.com/antchfx/xmlquery" // xmlquery uses antchfx/xpath underneath
)

// XMLPayload handles XML data.
type XMLPayload struct {
	rawContent  []byte
	parsedDoc   *xmlquery.Node // Using xmlquery's Node for easier navigation if needed
	contentType string
}

// NewXMLPayload creates a new XMLPayload.
// It parses the XML content upon creation.
func NewXMLPayload(content []byte) (*XMLPayload, error) {
	doc, err := xmlquery.Parse(bytes.NewReader(content))
	if err != nil {
		return nil, &ErrEvaluationFailed{Reason: "XML parsing failed", InnerError: err}
	}
	return &XMLPayload{
		rawContent:  content,
		parsedDoc:   doc,
		contentType: "application/xml",
	}, nil
}

func (xp *XMLPayload) GetRawBytes() []byte {
	return xp.rawContent
}

func (xp *XMLPayload) GetContentType() string {
	return xp.contentType
}

// Query evaluates an XPath expression against the XML payload.
func (xp *XMLPayload) Query(expression string) (QueryResult, error) {
	if xp.parsedDoc == nil {
		return QueryResult{}, &ErrEvaluationFailed{Expression: expression, Reason: "XML document not parsed"}
	}

	// Compile the XPath expression
	exprCompiled, err := xpath.Compile(expression)
	if err != nil {
		return QueryResult{}, &ErrEvaluationFailed{Expression: expression, Reason: "XPath compilation failed", InnerError: err}
	}

	// Evaluate the expression
	// The antchfx/xpath navigator works on an xmlquery.Node
	nav := xmlquery.CreateXPathNavigator(xp.parsedDoc)
	val := exprCompiled.Evaluate(nav)

	switch result := val.(type) {
	case string:
		return QueryResult{Value: result, Type: StringResult}, nil
	case float64:
		return QueryResult{Value: result, Type:NumberResult}, nil
	case bool:
		return QueryResult{Value: result, Type: BooleanResult}, nil
	case *xpath.NodeIterator:
		var results []string // For simplicity, collecting text content of nodes
		var nodes []*xmlquery.Node
		for result.MoveNext() {
			node := result.Current().(*xmlquery.NodeNavigator).Current()
			nodes = append(nodes, node)
			// For text(), it's often better to get it directly via XPath string() or text()
			// If the XPath itself returns a string (e.g. /a/b/text()), it's handled above.
			// If it returns nodes, we might want to return the nodes or their string representations.
			// For this PoC, if it's a nodeset, we'll try to get the InnerText.
			results = append(results, node.InnerText())
		}
		if len(nodes) == 0 {
			 // XPath selected nothing, which is not an error but an empty result.
			 // Depending on strictness, could return nil or empty string/slice.
			 return QueryResult{Value: nil, Type: NodeSetResult}, nil // Or specific type like StringResult with ""
		}
		// If the original XPath was like "/a/b/text()", it would directly be a string.
		// If it was "/a/b", it's a nodeset.
		// Let's return the first node's text if only one, or slice of texts if multiple.
		// This heuristic might need refinement based on desired behavior for node-set results.
		if len(results) == 1 {
			// If the XPath was specific and returned one node, give its text.
			// If the XPath was like "/a/b[1]/text()", it would be string.
			// If it was "/a/b[1]", this is reasonable.
			return QueryResult{Value: results[0], Type: StringResult}, nil
		}
		return QueryResult{Value: results, Type: NodeSetResult}, nil
	default:
		// This case might occur if XPath evaluates to something unexpected by this simplified switch
		return QueryResult{}, &ErrEvaluationFailed{Expression: expression, Reason: fmt.Sprintf("unexpected XPath result type: %T", val)}
	}
}

func (xp *XMLPayload) AsString() (string, error) {
	if xp.parsedDoc != nil {
		return xp.parsedDoc.OutputXML(true), nil // true for pretty print
	}
	return string(xp.rawContent), nil
}

func (xp *XMLPayload) GetUnderlying() interface{} {
	return xp.parsedDoc
}
