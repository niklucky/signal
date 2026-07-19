// Package static provides the web UI static assets embedded into the binary.
package static

import "embed"

// FS contains the compiled CSS and JavaScript assets served under /static/.
//
//go:embed css/main.css js
var FS embed.FS
