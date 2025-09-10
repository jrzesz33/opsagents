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

# Clean up all AWS resources
./build/opsagents cleanup
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

The Claude AI agent has access to three primary tools:
- `deploy_application`: Deploys pre-built containers to AWS ECS
- `get_deployment_status`: Retrieves current deployment status
- `cleanup_resources`: Removes all AWS ECS resources including services, clusters, load balancers, and log groups

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

## Advanced Configuration Features

OpsAgents supports advanced AWS configuration including secrets management, persistent storage, and enhanced security features as specified in `docs/reqs/configs.md`.

### Secrets Management (AWS Secrets Manager)

When `create_secrets: true` is enabled, OpsAgents automatically creates and manages the following secrets:

1. **Database Password** - Auto-generated 32-character password for Neo4j
2. **JWT Secret** - Auto-generated 64-character secret for authentication  
3. **Session Key** - Auto-generated 32-character key for session management
4. **Anthropic API Key** - From `ANTHROPIC_API_KEY` environment variable
5. **Gmail Credentials** - From `GMAIL_USER` and `GMAIL_PASS` environment variables

**Environment Variable Mapping:**
- **Web Application Container:**
  - `DB_ADMIN` ← Database password secret
  - `JWT_SECRET` ← JWT secret  
  - `SESSION_KEY` ← Session key
  - `ANTHROPIC_API_KEY` ← Anthropic API key
  - `GMAIL_USER` ← Gmail user credential
  - `GMAIL_PASS` ← Gmail password credential
  - `MODE` ← Application mode (prod/dev/test)

- **Database Container:**  
  - `NEO4J_PASSWORD` ← Database password secret

### Persistent Storage (AWS EFS)

When `create_efs: true` is enabled, OpsAgents creates an EFS file system with:

- **Provisioned throughput** (10 MiB/s)
- **Transit encryption** enabled
- **Mount targets** in all available subnets
- **Automatic mounting** to `/data` directory in Neo4j container

**Database Configuration:**
- Container exposes ports **7474** (HTTP) and **7687** (Bolt)
- EFS volume mounted to `/data` for persistent Neo4j storage
- Secure password-based authentication via secrets

### Configuration Options

Update your `config.yaml` to enable advanced features:

```yaml
aws:
  ecs:
    create_secrets: true      # Enable AWS Secrets Manager
    create_efs: true         # Enable EFS persistent storage  
    mode: "prod"             # Application mode
    environment:
      ENV: production
      PORT: "8000"
```

**Required Environment Variables** (for secrets creation):
```bash
export ANTHROPIC_API_KEY="your-anthropic-key"
export GMAIL_USER="your-gmail-user"  
export GMAIL_PASS="your-gmail-app-password"
```

## Cleanup and Resource Management

OpsAgents provides comprehensive cleanup functionality to remove all AWS ECS resources when they're no longer needed.

### Cleanup Methods

**1. CLI Command:**
```bash
./build/opsagents cleanup
```
- Interactive confirmation required
- Lists all resources that will be deleted
- Safely handles resource dependencies

**2. Claude AI Agent:**
Ask the agent to clean up resources:
- "cleanup resources" 
- "remove all AWS resources"
- "delete the deployment"

**3. Programmatic Cleanup:**
```go
deployer, _ := deploy.NewECSDeployer()
err := deployer.Cleanup(ecsConfig)
```

### Resources Cleaned Up

The cleanup process removes the following AWS resources in the correct order:

1. **ECS Service** - Scales down to 0 and deletes the service
2. **Task Definitions** - Deregisters all revisions of the task definition
3. **Load Balancer Resources:**
   - Application Load Balancer listeners
   - Application Load Balancer  
   - Target Groups (after load balancer is deleted)
4. **CloudWatch Log Groups** - Removes webapp and database log groups
5. **AWS Secrets Manager Secrets** (if `create_secrets: true`):
   - Database password secret
   - JWT secret
   - Session key secret
   - Anthropic API key secret  
   - Gmail user/password secrets
6. **EFS File System** (if `create_efs: true`):
   - EFS mount targets (all subnets)
   - EFS file system with all data
7. **ECS Cluster** - Deletes cluster only if empty (no other services/tasks)

### Safety Features

- **Confirmation Required**: CLI prompts for user confirmation
- **Agent Confirmation**: Claude agent requires explicit `confirm: true` parameter
- **Dependency Handling**: Resources are deleted in the correct order
- **Error Handling**: Continues cleanup even if some resources fail to delete
- **Cluster Protection**: Only deletes ECS cluster if it has no other resources

### Example Usage

**CLI:**
```bash
$ ./build/opsagents cleanup
This will delete the following resources:
  - ECS Service: bigfootgolf-service
  - ECS Cluster: bigfootgolf-cluster (if empty)
  - Task Definition: bigfootgolf-task (all revisions)
  - Load Balancer: bigfootgolf-service-alb
  - Target Group: bigfootgolf-service-tg
  - CloudWatch Log Groups

Are you sure you want to proceed? (yes/no): yes
✅ Cleanup completed successfully!
```

**Claude Agent:**
```
You: cleanup all AWS resources
Claude: I'll clean up all the AWS ECS resources for you. This will remove the service, load balancers, and associated infrastructure.
✅ Successfully cleaned up all AWS ECS resources for service 'bigfootgolf-service'!
```