package keycloak

import "testing"

func TestSanitizePersonName(t *testing.T) {
	cases := map[string]string{
		"Anna":               "Anna",
		"Jürgen Müller-Weiß": "Jürgen Müller-Weiß",
		"O'Brien":            "O'Brien",
		"Kai (Hamburg)":      "Kai Hamburg",
		"Peter/Paul":         "Peter Paul",
		"  Lisa\tMarie\n":    "Lisa Marie",
		"A&B; <c> \"d\" #1!": "A B c d 1",
		"x$%?§*~\\|^=[]{}y":  "x y",
		"()":                 "",
		"Ion (Nelu) Popescu": "Ion Nelu Popescu",
	}
	for in, want := range cases {
		if got := sanitizePersonName(in); got != want {
			t.Errorf("sanitizePersonName(%q) = %q, want %q", in, got, want)
		}
	}
}
