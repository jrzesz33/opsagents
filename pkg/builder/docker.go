package builder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type DockerBuilder struct {
	SourceDir string
}

func NewDockerBuilder(sourceDir string) *DockerBuilder {
	return &DockerBuilder{
		SourceDir: sourceDir,
	}
}

func (d *DockerBuilder) BuildImage(imageName, dockerfilePath string) error {
	fmt.Printf("Building Docker image: %s using %s\n", imageName, dockerfilePath)
	
	cmd := exec.Command("docker", "build", "-t", imageName, "-f", dockerfilePath, ".")
	cmd.Dir = d.SourceDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	fmt.Printf("Docker image built successfully: %s\n", imageName)
	return nil
}

func (d *DockerBuilder) CreateDockerfiles() error {
	if err := d.createGoAppDockerfile(); err != nil {
		return err
	}
	
	if err := d.createNeo4jDockerfile(); err != nil {
		return err
	}
	
	return nil
}

func (d *DockerBuilder) createGoAppDockerfile() error {
	dockerfile := `FROM cgr.dev/chainguard/static:latest

WORKDIR /app

COPY build/bigfootgolf /app/bigfootgolf

EXPOSE 8080

ENTRYPOINT ["/app/bigfootgolf"]
`
	
	dockerfilePath := filepath.Join(d.SourceDir, "Dockerfile.app")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		return fmt.Errorf("failed to create Go app Dockerfile: %w", err)
	}
	
	fmt.Printf("Go app Dockerfile created: %s\n", dockerfilePath)
	return nil
}

func (d *DockerBuilder) createNeo4jDockerfile() error {
	dockerfile := `FROM neo4j:5-community

ENV NEO4J_AUTH=neo4j/password
ENV NEO4J_dbms_default__database=bigfoot
ENV NEO4J_dbms_security_procedures_unrestricted=apoc.*
ENV NEO4J_dbms_security_procedures_allowlist=apoc.*

EXPOSE 7474 7687

VOLUME /data
VOLUME /logs
VOLUME /import
VOLUME /plugins
`
	
	dockerfilePath := filepath.Join(d.SourceDir, "Dockerfile.neo4j")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		return fmt.Errorf("failed to create Neo4j Dockerfile: %w", err)
	}
	
	fmt.Printf("Neo4j Dockerfile created: %s\n", dockerfilePath)
	return nil
}

func (d *DockerBuilder) TagImage(imageName, tag string) error {
	fullTag := fmt.Sprintf("%s:%s", imageName, tag)
	
	fmt.Printf("Tagging image %s as %s\n", imageName, fullTag)
	
	cmd := exec.Command("docker", "tag", imageName, fullTag)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to tag Docker image: %w", err)
	}
	
	return nil
}