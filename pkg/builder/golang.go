package builder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type GoBuilder struct {
	SourceDir string
	OutputDir string
}

func NewGoBuilder(sourceDir, outputDir string) *GoBuilder {
	return &GoBuilder{
		SourceDir: sourceDir,
		OutputDir: outputDir,
	}
}

func (b *GoBuilder) BuildBinary(appName string) error {
	if err := os.MkdirAll(b.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	binaryPath := filepath.Join(b.OutputDir, appName)
	
	fmt.Printf("Building Go binary: %s -> %s\n", b.SourceDir, binaryPath)
	
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = b.SourceDir
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS=linux",
		"GOARCH=amd64",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build Go binary: %w", err)
	}

	fmt.Printf("Go binary built successfully: %s\n", binaryPath)
	return nil
}

func (b *GoBuilder) RunTests() error {
	fmt.Printf("Running tests in %s...\n", b.SourceDir)
	
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = b.SourceDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}

	fmt.Println("All tests passed successfully")
	return nil
}