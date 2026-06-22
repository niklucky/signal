package notifier

import (
	"github.com/niklucky/signal/internal/config"
)

// Matrix sends messages to a Matrix room (reserved for future use).
type Matrix struct {
	cfg config.MatrixConfig
}

// NewMatrix creates a new Matrix notifier placeholder.
func NewMatrix(cfg config.MatrixConfig) *Matrix {
	return &Matrix{cfg: cfg}
}

// Send is not implemented yet.
func (m *Matrix) Send(text string) error {
	// TODO: implement Matrix alerting in the next iteration.
	return nil
}
