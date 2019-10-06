package bidi

import "testing"

func TestReverse(t *testing.T) {
	testCases := []struct {
		s, want string
	}{
		{"hello", "olleh"},
		{"הסדנה", "הנדסה"},
		{"םלוע םולש", "שלום עולם"},
		{"group 20 hello", "olleh 20 puorg"},
		{"םולש 20 הצובק", "קבוצה 20 שלום"},
		{"(הלאכ) םיירגוס", "סוגריים (כאלה)"},
		{"3.14 יאפ", "פאי 3.14"},
		{"1970-01-02 ךיראת", "תאריך 1970-01-02"},
		{"19-ה האמה", "המאה ה-19"},
		{"ב-2 הסרג", "גרסה 2-ב"},
		{"-ףקמב", "במקף-"},
	}

	for _, tc := range testCases {
		got := Reverse(tc.s)
		if got != tc.want {
			t.Errorf("Reverse(a%qa) = a%qa; want a%qa", tc.s, got, tc.want)
		}
	}
}
