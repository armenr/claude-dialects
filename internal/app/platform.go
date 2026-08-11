package app

import (
	"fmt"
	"os/exec"
	"runtime"
)

func supportedPlatform(goos, goarch string) bool {
	return (goos == "darwin" || goos == "linux") && (goarch == "amd64" || goarch == "arm64")
}

func browserCommand(goos, rawURL string) (string, []string, error) {
	switch goos {
	case "darwin":
		return "open", []string{rawURL}, nil
	case "linux":
		return "xdg-open", []string{rawURL}, nil
	default:
		return "", nil, fmt.Errorf("opening a browser is not supported on %s; use --no-browser and open the printed URL manually", goos)
	}
}

func openBrowser(rawURL string) error {
	name, args, err := browserCommand(runtime.GOOS, rawURL)
	if err != nil {
		return err
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("browser opener %q was not found; use --no-browser and open the printed URL manually: %w", name, err)
	}
	return exec.Command(path, args...).Start()
}
