package parser

import (
	"fmt"
	"strings"
	// No aliasing needed here if no conflicts
)

// PayloadFactory creates PayloadObjects based on content type.
type PayloadFactory struct{}

func NewPayloadFactory() *PayloadFactory {
	return &PayloadFactory{}
}

// CreatePayload inspects content type and returns the appropriate PayloadObject.
// For PoC, parsing happens within the NewXYZPayload constructors.
func (pf *PayloadFactory) CreatePayload(raw []byte, contentType string) (PayloadObject, error) {
	// Normalize content type (e.g., "application/json; charset=utf-8" -> "application/json")
	normalizedContentType := strings.ToLower(strings.Split(contentType, ";")[0])

	switch normalizedContentType {
	case "application/xml", "text/xml":
		return NewXMLPayload(raw)
	case "application/json":
		return NewJSONPayload(raw)
	// Add cases for other types here
	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}
}
