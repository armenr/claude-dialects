package app

import (
	"reflect"
	"testing"
)

func TestSupportedPlatform(t *testing.T) {
	for _, test := range []struct {
		goos, goarch string
		want         bool
	}{
		{"darwin", "arm64", true},
		{"darwin", "amd64", true},
		{"linux", "arm64", true},
		{"linux", "amd64", true},
		{"windows", "amd64", false},
		{"linux", "386", false},
	} {
		if got := supportedPlatform(test.goos, test.goarch); got != test.want {
			t.Errorf("supportedPlatform(%q, %q) = %v, want %v", test.goos, test.goarch, got, test.want)
		}
	}
}

func TestBrowserCommand(t *testing.T) {
	const rawURL = "http://127.0.0.1:1234"
	for _, test := range []struct {
		goos string
		name string
	}{
		{"darwin", "open"},
		{"linux", "xdg-open"},
	} {
		name, args, err := browserCommand(test.goos, rawURL)
		if err != nil {
			t.Fatalf("browserCommand(%q): %v", test.goos, err)
		}
		if name != test.name || !reflect.DeepEqual(args, []string{rawURL}) {
			t.Errorf("browserCommand(%q) = %q, %v", test.goos, name, args)
		}
	}
	if _, _, err := browserCommand("windows", rawURL); err == nil {
		t.Fatal("expected unsupported-platform error")
	}
}
