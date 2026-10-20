package gemini_client

import "encoding/json"

// supportedSchemaKeys is the JSON schema subset accepted in function declarations.
var supportedSchemaKeys = map[string]bool{
	"type": true, "description": true, "properties": true, "required": true, "items": true,
	"enum": true, "format": true, "minimum": true, "maximum": true, "nullable": true,
}

// SanitizeSchema keeps only the supported keys of a JSON schema, such as the input schema of an MCP tool.
func SanitizeSchema(schema any) any {
	b, err := json.Marshal(schema)
	if err != nil {
		return nil
	}
	var generic any
	if err := json.Unmarshal(b, &generic); err != nil {
		return nil
	}
	return sanitize(generic)
}

func sanitize(node any) any {
	object, ok := node.(map[string]any)
	if !ok {
		return node
	}
	out := make(map[string]any, len(object))
	for key, value := range object {
		if !supportedSchemaKeys[key] {
			continue
		}
		switch key {
		case "properties":
			if properties, ok := value.(map[string]any); ok {
				clean := make(map[string]any, len(properties))
				for name, property := range properties {
					clean[name] = sanitize(property)
				}
				value = clean
			}
		case "items":
			value = sanitize(value)
		}
		out[key] = value
	}
	return out
}
