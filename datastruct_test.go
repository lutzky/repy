package repy

import (
	"encoding/json"
	"testing"
)

func TestMarshalGroupType(t *testing.T) {
	testCases := []struct {
		gt   GroupType
		want string
	}{
		{Lecture, "\"lecture\""},
		{Tutorial, "\"tutorial\""},
		{Lab, "\"lab\""},
	}

	for _, tc := range testCases {
		t.Run(tc.gt.String(), func(t *testing.T) {
			b, err := json.Marshal(tc.gt)
			if err != nil {
				t.Errorf("Failed to marshal %+v: %v", tc.gt, err)
			}
			if string(b) != tc.want {
				t.Errorf("json.Marshal(%+v) = %q; want %q", tc.gt, b, tc.want)
			}

			var reverse GroupType
			if err := json.Unmarshal(b, &reverse); err != nil {
				t.Fatalf("json.Unmarshal(%q) failed: %v", b, err)
			}
			if reverse != tc.gt {
				t.Errorf("json.Unmarshal(%q) = %+v; want %+v", b, reverse, tc.gt)
			}
		})
	}
}
