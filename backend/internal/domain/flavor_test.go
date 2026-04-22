package domain

import "testing"

func TestParseFlavor_Valid(t *testing.T) {
	cases := []struct {
		in   string
		want Flavor
	}{
		{"vanilla", FlavorVanilla},
		{"chocolate", FlavorChocolate},
		{"matcha", FlavorMatcha},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, err := ParseFlavor(c.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Fatalf("want %q, got %q", c.want, got)
			}
		})
	}
}

func TestParseFlavor_Invalid(t *testing.T) {
	cases := []string{"", "strawberry", "VANILLA", "choco"}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			if _, err := ParseFlavor(in); err == nil {
				t.Fatalf("expected error for %q, got nil", in)
			}
		})
	}
}

func TestFlavor_String(t *testing.T) {
	if got := FlavorVanilla.String(); got != "vanilla" {
		t.Fatalf("want vanilla, got %s", got)
	}
}
