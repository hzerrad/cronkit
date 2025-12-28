package cronx_test

import (
	"testing"

	"github.com/hzerrad/cronic/internal/cronx"
	"github.com/stretchr/testify/assert"
)

// TestParseValue tests the parseValue function indirectly through field parsing
// Since parseValue is not exported, we test it through parseField which uses it
func TestParseValue_Unparseable(t *testing.T) {
	parser := cronx.NewParser()

	// Test that parseValue returns 0 for unparseable values
	// This happens when a field contains a value that's neither numeric nor a valid symbol
	// We test this through a range that includes an invalid symbol
	_, err := parser.Parse("0 0 * * MON-INVALID")
	// This should fail because "INVALID" is not a valid day name
	assert.Error(t, err, "Parse should fail for invalid symbol in range")

	// Test with a list containing invalid value
	_, err = parser.Parse("0 0 * * MON,INVALID")
	assert.Error(t, err, "Parse should fail for invalid symbol in list")
}
