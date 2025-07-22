package utils

// getString safely extracts a string value from a map[string]interface{}
// Returns empty string if key doesn't exist or value is not a string
func GetString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// getMap safely extracts a map[string]interface{} value from a map[string]interface{}
// Returns nil if key doesn't exist or value is not a map
func GetMap(m map[string]interface{}, key string) map[string]interface{} {
	if val, ok := m[key].(map[string]interface{}); ok {
		return val
	}
	return nil
}