package schema

func ConditionalDefault(condition bool, defaultValue interface{}) interface{} {
	if !condition {
		return nil
	}
	return defaultValue
}
