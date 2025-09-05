package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type GitClient struct {
	WorkingDir string
}

func New(workingDir string) *GitClient {
	return &GitClient{
		WorkingDir: workingDir,
	}
}

func (g *GitClient) CloneRepository(repoURL, targetDir string) error {
	fullPath := filepath.Join(g.WorkingDir, targetDir)
	
	if _, err := os.Stat(fullPath); err == nil {
		fmt.Printf("Directory %s already exists, removing...\n", fullPath)
		if err := os.RemoveAll(fullPath); err != nil {
			return fmt.Errorf("failed to remove existing directory: %w", err)
		}
	}

	fmt.Printf("Cloning repository %s to %s...\n", repoURL, fullPath)
	cmd := exec.Command("git", "clone", repoURL, fullPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	fmt.Println("Repository cloned successfully")
	return nil
}

func (g *GitClient) PullLatest(repoDir string) error {
	fullPath := filepath.Join(g.WorkingDir, repoDir)
	
	fmt.Printf("Pulling latest changes in %s...\n", fullPath)
	cmd := exec.Command("git", "pull", "origin", "main")
	cmd.Dir = fullPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull latest changes: %w", err)
	}

	fmt.Println("Repository updated successfully")
	return nil
}