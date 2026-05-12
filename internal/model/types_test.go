package model

import "testing"

func TestVisibilityShort(t *testing.T) {
	cases := []struct {
		vis  Visibility
		want string
	}{
		{VisibilityPrivate, "PRIVATE"},
		{VisibilityProtected, "PROTECTED"},
		{VisibilityPublic, "PUBLIC"},
		{"UNKNOWN", "PRIVATE"}, // default branch
	}
	for _, c := range cases {
		got := c.vis.Short()
		if got != c.want {
			t.Errorf("Visibility(%q).Short() = %q, want %q", c.vis, got, c.want)
		}
	}
}

func TestVisibilityNext(t *testing.T) {
	cases := []struct {
		vis  Visibility
		want Visibility
	}{
		{VisibilityPrivate, VisibilityProtected},
		{VisibilityProtected, VisibilityPublic},
		{VisibilityPublic, VisibilityPrivate},
		{"UNKNOWN", VisibilityPrivate}, // default branch
	}
	for _, c := range cases {
		got := c.vis.Next()
		if got != c.want {
			t.Errorf("Visibility(%q).Next() = %q, want %q", c.vis, got, c.want)
		}
	}
}

func TestVisibilityNextCycles(t *testing.T) {
	v := VisibilityPrivate
	v = v.Next() // PROTECTED
	v = v.Next() // PUBLIC
	v = v.Next() // back to PRIVATE
	if v != VisibilityPrivate {
		t.Errorf("full cycle did not return to PRIVATE, got %q", v)
	}
}
