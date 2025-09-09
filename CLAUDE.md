# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Build and Development
```bash
# Build the application
go build -o build/opsagents cmd/opsagents/main.go

# Build with optimizations for production
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o build/opsagents cmd/opsagents/main.go

# Run tests
go test ./...

# Install dependencies
go mod download

# Generate default configuration
./build/opsagents config
```

### Application Usage
```bash
# Start Claude AI agent (interactive mode)
./build/opsagents agent

# Deploy directly (non-interactive)
./build/opsagents deploy
```

## Architecture Overview

OpsAgents is a Claude AI-powered DevOps automation tool that integrates AWS Bedrock with AWS Lightsail for intelligent deployment workflows using pre-built Docker images.

### Core Components

1. **Claude AI Agent** (`pkg/agent/claude.go`)
   - Uses AWS Bedrock for Claude 3 Sonnet model integration
   - Implements tool-calling interface for deployment operations
   - Natural language processing for DevOps commands

2. **CLI Interface** (`cmd/opsagents/main.go`)
   - Cobra-based CLI with three main commands: `agent`, `deploy`, `config`
   - Interactive chat mode with Claude AI agent
   - Direct command execution for automated workflows

3. **Deployment System** (`pkg/deploy/lightsail.go`)
   - AWS Lightsail container service management
   - Container deployment with health checks
   - Service monitoring and status reporting

4. **Configuration Management** (`internal/config/config.go`)
   - YAML-based configuration with Viper
   - Environment variable integration
   - Claude AI, AWS service, and Docker registry settings

### Tool Integration

The Claude AI agent has access to two primary tools:
- `deploy_application`: Deploys pre-built containers to AWS Lightsail
- `get_deployment_status`: Retrieves current deployment status

### Environment Variables

Required environment variables:
```bash
AWS_ACCESS_KEY_ID="xxx"     # AWS credentials
AWS_SECRET_ACCESS_KEY="xxx" # AWS credentials  
AWS_DEFAULT_REGION="us-east-1"
```

Alternative AWS authentication:
```bash
AWS_PROFILE="profile-name"  # AWS Profile for authentication
```

## Project Structure

```
opsagents/
├── cmd/opsagents/          # CLI entry point
│   └── main.go            # Cobra CLI with agent integration
├── pkg/                    # Public packages
│   ├── agent/             # Claude AI agent implementation
│   └── deploy/            # AWS Lightsail deployment
├── internal/              # Private packages
│   └── config/            # Configuration management
├── docs/                  # Documentation and requirements
└── config.yaml           # Runtime configuration
```

## Configuration

The application uses `config.yaml` for configuration. Generate with:
```bash
./build/opsagents config
```

Key configuration sections:
- `claude`: AWS Bedrock and model settings
- `images`: Docker registry and image specifications
- `aws.lightsail`: Container service configuration
- `auth`: Environment variable names for credentials

## AWS Permissions

Required AWS IAM permissions:
- **Bedrock**: `bedrock:InvokeModel`, `bedrock:ListFoundationModels`
- **Lightsail**: `lightsail:CreateContainerService`, `lightsail:CreateContainerServiceDeployment`, `lightsail:GetContainerServices`

## Docker Images

Deploys pre-built Docker images from external registries:
1. **Application Image**: Configured via `images.app_image` setting
2. **Neo4j Database Image**: Configured via `images.neo4j_image` setting (default: neo4j:5-community)

## Testing and Linting

The project uses Go's built-in testing framework. Run tests with:
```bash
go test ./...
```

For linting, use standard Go tools:
```bash
go vet ./...
gofmt -l .
```