package testing

import (
	"testing"

	platform "github.com/influxdata/influxdb"
)

// TODO(goller): remove opPrefix argument
func diffPlatformErrors(name string, actual, expected error, opPrefix string, t *testing.T) {
	t.Helper()
	ErrorsEqual(t, actual, expected)
}

// ErrorsEqual checks to see if the provided errors are equivalent.
func ErrorsEqual(t *testing.T, actual, expected error) {
	t.Helper()
	if expected == nil && actual == nil {
		return
	}

	if expected == nil && actual != nil {
		t.Errorf("unexpected error %s", actual.Error())
	}

	if expected != nil && actual == nil {
		t.Errorf("expected error %s but received nil", expected.Error())
	}

	if platform.ErrorCode(expected) != platform.ErrorCode(actual) {
		t.Logf("\nexpected: %v\nactual: %v\n\n", expected, actual)
		t.Errorf("expected error code %q but received %q", platform.ErrorCode(expected), platform.ErrorCode(actual))
	}

	if platform.ErrorMessage(expected) != platform.ErrorMessage(actual) {
		t.Logf("\nexpected: %v\nactual: %v\n\n", expected, actual)
		t.Errorf("expected error message %q but received %q", platform.ErrorMessage(expected), platform.ErrorMessage(actual))
	}
}

// FloatPtr takes the ref of a float number.
func FloatPtr(f float64) *float64 {
	p := new(float64)
	*p = f
	return p
}

func idPtr(id platform.ID) *platform.ID {
	return &id
}

// MustIDBase16 is an helper to ensure a correct ID is built during testing.
func MustIDBase16(s string) platform.ID {
	id, err := platform.IDFromString(s)
	if err != nil {
		panic(err)
	}
	return *id
}
