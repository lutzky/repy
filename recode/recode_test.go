package recode

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/text/encoding/charmap"
)

func TestRecode(t *testing.T) {
	testCases := []struct {
		name        string
		from, to    *charmap.Charmap
		input, want []byte
	}{
		{
			"english_stays_english",
			charmap.ISO8859_1, charmap.ISO8859_8,
			[]byte("hello"), []byte("hello"),
		},
		{
			"hebrew_recoding",
			charmap.CodePage862, charmap.ISO8859_8,
			[]byte{0x99, 0x8c, 0x85, 0x8d}, []byte{0xf9, 0xec, 0xe5, 0xed},
		},
		{
			"hebrew_inverse_recoding",
			charmap.ISO8859_8, charmap.CodePage862,
			[]byte{0xf9, 0xec, 0xe5, 0xed}, []byte{0x99, 0x8c, 0x85, 0x8d},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Recode(tc.from, tc.to, tc.input)
			if err != nil {
				t.Fatal("Got error", err)
			}
			if d := cmp.Diff(got, tc.want); d != "" {
				t.Errorf("-got +want:\n%s", d)
			}
		})
	}
}
