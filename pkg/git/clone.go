package git

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type GitClient struct {
	WorkingDir string
	Token      string
}

func New(workingDir, token string) *GitClient {
	return &GitClient{
		WorkingDir: workingDir,
		Token:      token,
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

	// Add authentication to the URL if token is provided
	authenticatedURL, err := g.addAuthToURL(repoURL)
	if err != nil {
		return fmt.Errorf("failed to add authentication to URL: %w", err)
	}

	fmt.Printf("Cloning repository %s to %s...\n", repoURL, fullPath)
	cmd := exec.Command("git", "clone", authenticatedURL, fullPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	// Set environment variables for Git credentials
	cmd.Env = append(os.Environ(), g.getGitEnv()...)

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
	
	// Set environment variables for Git credentials
	cmd.Env = append(os.Environ(), g.getGitEnv()...)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull latest changes: %w", err)
	}

	fmt.Println("Repository updated successfully")
	return nil
}

func (g *GitClient) addAuthToURL(repoURL string) (string, error) {
	if g.Token == "" {
		return repoURL, nil
	}

	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse repository URL: %w", err)
	}

	// Only add auth for HTTPS URLs
	if parsedURL.Scheme == "https" && strings.Contains(parsedURL.Host, "github.com") {
		parsedURL.User = url.UserPassword("token", g.Token)
	}

	return parsedURL.String(), nil
}

func (g *GitClient) getGitEnv() []string {
	env := []string{}
	
	if g.Token != "" {
		// Set Git credentials for HTTPS
		env = append(env, "GIT_ASKPASS=echo")
		env = append(env, fmt.Sprintf("GIT_USERNAME=token"))
		env = append(env, fmt.Sprintf("GIT_PASSWORD=%s", g.Token))
	}
	
	return env
}