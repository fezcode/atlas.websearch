package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Target struct {
	OS   string
	Arch string
}

func main() {
	appName := "atlas-websearch"
	targets := []Target{
		// Linux
		{"linux", "386"},
		{"linux", "amd64"},
		{"linux", "arm"},
		{"linux", "arm64"},
		// Windows
		{"windows", "386"},
		{"windows", "amd64"},
		{"windows", "arm"},
		{"windows", "arm64"},
		// macOS (Darwin)
		{"darwin", "amd64"},
		{"darwin", "arm64"},
	}

	buildDir := "build"

	// Ensure build directory exists
	if _, err := os.Stat(buildDir); os.IsNotExist(err) {
		err := os.Mkdir(buildDir, 0755)
		if err != nil {
			fmt.Printf("Error creating build directory: %v\n", err)
			return
		}
	}

	for _, t := range targets {
		outputName := fmt.Sprintf("%s-%s-%s", appName, t.OS, t.Arch)
		if t.OS == "windows" {
			outputName += ".exe"
		}

		outputPath := filepath.Join(buildDir, outputName)
		fmt.Printf("Building for %s/%s -> %s\n", t.OS, t.Arch, outputName)

		cmd := exec.Command("go", "build", "-o", outputPath, "main.go")
		cmd.Env = append(os.Environ(),
			"GOOS="+t.OS,
			"GOARCH="+t.Arch,
			"CGO_ENABLED=0",
		)

		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("  Error building for %s/%s: %v\n", t.OS, t.Arch, err)
			fmt.Printf("  Output: %s\n", string(out))
		}
	}
}
