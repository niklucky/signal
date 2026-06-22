package templates

// statusEmoji returns an emoji matching the alert status.
func statusEmoji(status string) string {
	switch status {
	case "resolved":
		return "✅"
	default:
		return "🔥"
	}
}

// firstNonEmpty returns the first non-empty string from the provided values.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
