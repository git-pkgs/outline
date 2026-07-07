package outline

import (
	"os"
	"testing"
)

// TestMain disables the per-file parse timeout for the test suite. The
// timeout exists to keep pathological real-world inputs from dominating a
// Pack run; test fixtures are tiny and should never hit it. On slow CI
// runners (macOS under -race) the Swift grammar has been observed to brush
// the 1s default and intermittently trip ParseStoppedEarly, turning a
// correctness test into a timing test.
func TestMain(m *testing.M) {
	SetParseTimeout(0)
	os.Exit(m.Run())
}
