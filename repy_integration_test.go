package repy

import "testing"

func TestReadKnownREPY(t *testing.T) {
	catalog, err := ReadFile("REPY")
	if err != nil {
		t.Fatalf("Couldn't parse REPY: %v", err)
	}

	if len(*catalog) != 8 {
		t.Errorf("len(catalog) == %d; want 8", len(*catalog))
	}
}
