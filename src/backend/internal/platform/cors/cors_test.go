package cors

import "testing"

func TestAllowed(t *testing.T) {
	allowed := []string{"https://app.example.com"}

	cases := []struct {
		origin string
		want   bool
	}{
		{"https://app.example.com", true},
		{"https://app.example.com/", true}, // trailing slash tolerated
		{"https://evil.example.com", false},
		{"", false},
		{"http://localhost:3000", true},      // localhost always allowed
		{"http://127.0.0.1:5173", true},      // loopback always allowed
		{"http://[::1]:3000", true},          // ipv6 loopback
		{"http://localhost.evil.com", false}, // not actually loopback
	}
	for _, tc := range cases {
		if got := Allowed(tc.origin, allowed); got != tc.want {
			t.Errorf("Allowed(%q) = %v, want %v", tc.origin, got, tc.want)
		}
	}
}

func TestIsLocalhost(t *testing.T) {
	if IsLocalhost("https://example.com") {
		t.Fatal("example.com should not be localhost")
	}
	if !IsLocalhost("http://localhost:1234") {
		t.Fatal("localhost should be localhost")
	}
	if IsLocalhost("not a url") {
		t.Fatal("invalid url should not be localhost")
	}
}
