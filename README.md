# OpsAgents - Build and Deploy Automation Tool

A CLI tool that orchestrates the building and deployment of applications to AWS Lightsail. This agent automates the entire workflow from Git repository management to containerized deployment.

## Features

- **Git Repository Management**: Automatically clone and pull the latest changes from GitHub repositories
- **Go Binary Building**: Cross-compile Go applications with optimized settings
- **Docker Image Creation**: Build container images using secure Chainguard base images
- **AWS Lightsail Deployment**: Deploy containers to cost-efficient AWS Lightsail services
- **Configuration Management**: Flexible YAML-based configuration with environment variable support

## Quick Start

### 1. Build the Tool

```bash
go build -o build/opsagents cmd/opsagents/main.go
```

### 2. Generate Configuration

```bash
./build/opsagents config
```

This creates a `config.yaml` file with default settings for the BigFoot Golf application.

### 3. Build Application

```bash
./build/opsagents build
```

This will:
- Clone the repository from GitHub
- Build the Go binary
- Create Docker images for the application and Neo4j database

### 4. Deploy to AWS

```bash
./build/opsagents deploy
```

This will deploy the containers to AWS Lightsail.

## Commands

### `opsagents build`
Executes the complete build pipeline:
- Clones/pulls from the configured Git repository
- Builds the Go binary with cross-compilation (Linux/AMD64)
- Creates Dockerfiles using Chainguard base images
- Builds Docker images for both the application and Neo4j database

### `opsagents deploy`
Deploys the application to AWS Lightsail:
- Creates or updates the Lightsail container service
- Deploys the application container with health checks
- Waits for the service to become ready
- Provides the service URL when deployment is complete

### `opsagents config`
Generates a default `config.yaml` file with sensible defaults for the BigFoot Golf project.

## Configuration

The tool uses a `config.yaml` file for configuration. Here's the structure:

```yaml
agent_name: bigfootgolf-agent
port: 8080
log_level: info

git:
  repository: https://github.com/jrzesz33/bigfootgolf
  branch: main
  working_dir: ./workspace

build:
  app_name: bigfootgolf
  output_dir: ./build

aws:
  region: us-east-1
  lightsail:
    service_name: bigfootgolf-service
    power: nano          # Most cost-efficient option
    scale: 1
    public_domain: bigfootgolf.example.com
    container_name: bigfootgolf-app
    environment:
      ENV: production
      PORT: "8080"
```

### Configuration Options

- **git.repository**: GitHub repository URL to clone
- **git.branch**: Git branch to use (default: main)
- **git.working_dir**: Local directory for cloned repository
- **build.app_name**: Name of the Go binary and Docker images
- **aws.lightsail.power**: Container size (nano, micro, small, medium, large)
- **aws.lightsail.scale**: Number of container instances

## Prerequisites

### Required Tools
- Go 1.24+
- Docker
- Git
- AWS CLI configured with appropriate credentials

### AWS Permissions
Your AWS credentials need the following Lightsail permissions:
- `lightsail:CreateContainerService`
- `lightsail:CreateContainerServiceDeployment`
- `lightsail:GetContainerServices`

## Docker Images

The tool creates two Docker images:

### Application Image (`bigfootgolf-app`)
- Base: `cgr.dev/chainguard/static:latest` (minimal, secure)
- Contains the compiled Go binary
- Exposes port 8080

### Neo4j Database Image (`bigfootgolf-neo4j`)
- Base: `neo4j:5-community`
- Pre-configured with authentication and database settings
- Includes volume mounts for data persistence

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Git Clone     │───▶│   Go Build      │───▶│  Docker Build   │
│                 │    │                 │    │                 │
│ • Clone repo    │    │ • Cross-compile │    │ • App image     │
│ • Pull latest   │    │ • Linux/AMD64   │    │ • Neo4j image   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                       │
                                                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Service Ready │◀───│  Wait & Monitor │◀───│ AWS Lightsail   │
│                 │    │                 │    │                 │
│ • Health checks │    │ • Service state │    │ • Create service│
│ • Public URL    │    │ • Deployment    │    │ • Deploy images │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Project Structure

```
opsagents/
├── cmd/opsagents/          # CLI application entry point
│   └── main.go
├── pkg/                    # Public packages
│   ├── agent/             # Agent management
│   ├── builder/           # Build functionality
│   │   ├── docker.go      # Docker image building
│   │   └── golang.go      # Go binary building
│   ├── deploy/            # Deployment functionality
│   │   └── lightsail.go   # AWS Lightsail deployment
│   └── git/               # Git repository management
│       └── clone.go
├── internal/              # Private packages
│   └── config/            # Configuration management
│       └── config.go
├── docs/                  # Documentation
│   ├── README.md         # Project overview
│   └── reqs/             # Requirements specifications
├── .devcontainer/        # Development container config
├── .github/              # GitHub workflows
├── go.mod                # Go module definition
├── go.sum                # Go module checksums
└── config.yaml           # Runtime configuration
```

## Development

### Running in Development Container

This project includes a devcontainer configuration for consistent development:

```bash
# Open in VS Code with Dev Containers extension
code .
```

The devcontainer includes:
- Go 1.24
- AWS CLI
- Docker-in-Docker
- GitHub CLI

### Building from Source

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Build binary
go build -o build/opsagents cmd/opsagents/main.go

# Build with optimizations
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o build/opsagents cmd/opsagents/main.go
```

## Troubleshooting

### Build Issues
- Ensure Docker is running and accessible
- Verify Git repository access and credentials
- Check Go version compatibility (1.24+)

### Deployment Issues
- Verify AWS credentials are configured (`aws configure`)
- Check AWS region settings match your configuration
- Ensure Lightsail service limits aren't exceeded
- Verify container registry access for base images

### Configuration Issues
- Run `opsagents config` to regenerate default configuration
- Check YAML syntax in `config.yaml`
- Verify environment variables are set correctly

## Security

- Uses Chainguard distroless images for minimal attack surface
- Follows AWS IAM best practices with minimal required permissions
- Secrets and credentials managed through AWS/environment variables
- No hardcoded secrets in configuration files

## Cost Optimization

- Uses AWS Lightsail nano instances (most cost-effective)
- Single container instance by default
- Minimal base images reduce storage and transfer costs
- Automated scaling can be configured based on needs

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For issues and questions:
- Create an issue in the GitHub repository
- Check the troubleshooting section above
- Review the AWS Lightsail documentation for deployment-specific issues