package server

import "testing"

func TestValidateExportMaxRows(t *testing.T) {
	for _, value := range []int{1000, 100000, 1000000} {
		validated, err := validateExportMaxRows(value, 1000000)
		if err != nil || validated != value {
			t.Fatalf("value %d: validated=%d err=%v", value, validated, err)
		}
	}
	for _, value := range []int{0, 999, 1000001} {
		if _, err := validateExportMaxRows(value, 1000000); err == nil {
			t.Fatalf("value %d was accepted", value)
		}
	}
}
