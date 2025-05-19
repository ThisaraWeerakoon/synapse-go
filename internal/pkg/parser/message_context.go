package parser

import (
	"fmt"
	"sync"
)

// MessageContext holds the message payload and provides methods to interact with it.
type MessageContext struct {
	RawPayload       []byte
	ContentType      string
	processedPayload PayloadObject // Cached parsed payload
	payloadLock      sync.RWMutex
	engine           *ExpressionEngine // Reference to the expression engine
	payloadFactory   *PayloadFactory        // To create the initial payload object
}

func NewMessageContext(rawPayload []byte, contentType string, engine *ExpressionEngine) *MessageContext {
	return &MessageContext{
		RawPayload:     rawPayload,
		ContentType:    contentType,
		engine:         engine,
		payloadFactory: NewPayloadFactory(), // Each context gets its own factory instance or use a global one
	}
}

// ensurePayloadParsed lazily parses the payload if not already done.
// This is a helper for EvaluateExpression.
func (mc *MessageContext) ensurePayloadParsed() error {
	mc.payloadLock.RLock()
	if mc.processedPayload != nil {
		mc.payloadLock.RUnlock()
		return nil
	}
	mc.payloadLock.RUnlock() // Release read lock before acquiring write lock

	mc.payloadLock.Lock()
	defer mc.payloadLock.Unlock()
	// Double check after acquiring write lock
	if mc.processedPayload != nil {
		return nil
	}

	var err error
	mc.processedPayload, err = mc.payloadFactory.CreatePayload(mc.RawPayload, mc.ContentType)
	if err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}
	fmt.Printf("DEBUG: Payload (type: %s) parsed and cached.\n", mc.ContentType) // For demo
	return nil
}

// EvaluateExpression evaluates an expression against the message payload.
// It handles lazy parsing of the payload.
func (mc *MessageContext) EvaluateExpression(fullExpression string) (QueryResult, error) {
	if err := mc.ensurePayloadParsed(); err != nil {
		return QueryResult{}, err
	}
	// The engine's Evaluate method now takes the PayloadObject directly
	return mc.engine.Evaluate(mc.processedPayload, fullExpression)
}

// GetProcessedPayload returns the processed payload object, ensuring it's parsed.
// Useful if other parts of the system need direct access to the PayloadObject.
func (mc *MessageContext) GetProcessedPayload() (PayloadObject, error) {
    if err := mc.ensurePayloadParsed(); err != nil {
        return nil, err
    }
    return mc.processedPayload, nil
}
