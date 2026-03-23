package utils

import (
	"testing"

	"github.com/jackc/pgx/v5/tracelog"
)

// Check that traceLogger implements tracelog.Logger
// This test would fail to compile otherwise.
func TestTraceLoggerIsPgxLogger(t *testing.T) {
	tl := traceLogger{}
	_ = tracelog.Logger(&tl)
}
