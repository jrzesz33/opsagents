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

OpsAgents is a Claude AI-powered DevOps automation tool that deploys containerized applications to AWS ECS (Elastic Container Service) with intelligent deployment workflows using pre-built Docker images.

### Core Components

1. **Claude AI Agent** (`pkg/agent/claude.go`)
   - Uses AWS Bedrock for Claude 3 Sonnet model integration
   - Implements tool-calling interface for deployment operations
   - Natural language processing for DevOps commands

2. **CLI Interface** (`cmd/opsagents/main.go`)
   - Cobra-based CLI with three main commands: `agent`, `deploy`, `config`
   - Interactive chat mode with Claude AI agent
   - Direct command execution for automated workflows

3. **ECS Deployment System** (`pkg/deploy/ecs.go`)
   - AWS ECS container service management
   - Container deployment with health checks and load balancing
   - Service monitoring and status reporting
   - Advanced features: Secrets Manager integration, EFS persistent storage

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
- `aws.ecs`: ECS container service configuration
- `auth`: Environment variable names for credentials

# AWS Deployment Architecture

## Complete Infrastructure Overview

OpsAgents deploys a full-stack containerized application to AWS ECS with the following architecture:

### AWS Components Deployed

#### 1. **ECS (Elastic Container Service)**
- **ECS Cluster** - Container orchestration cluster (Fargate)
- **ECS Service** - Manages container instances and scaling
- **Task Definition** - Container specifications and resource allocation
- **Capacity Providers** - FARGATE and FARGATE_SPOT for serverless containers

#### 2. **Load Balancing & Networking**
- **Application Load Balancer (ALB)** - Internet-facing load balancer
- **Target Group** - Routes traffic to container instances
- **Listener** - HTTP listener on port 80
- **VPC Integration** - Auto-discovery of default VPC and subnets
- **Security Groups** - Network access control

#### 3. **Container Applications**

**Web Application Container:**
- **Image**: Configurable via `images.app_image` (default: bigfootgolf-webapp)
- **Port**: 8000 (configurable)
- **CPU**: 256 units (configurable)
- **Memory**: 512 MB (configurable)
- **Health Check**: `/health` endpoint

**Database Container (Neo4j):**
- **Image**: Configurable via `images.neo4j_image` (default: bigfootgolf-db)
- **Ports**: 7474 (HTTP), 7687 (Bolt)
- **CPU**: 256 units (configurable)
- **Memory**: 512 MB (configurable)
- **Data Persistence**: Optional EFS volume mount

#### 4. **Secrets Management (Optional)**
When `create_secrets: true`:
- **AWS Secrets Manager** - Secure storage for sensitive data
- **6 Auto-Generated Secrets**:
  - Database password (32-char auto-generated)
  - JWT secret (64-char auto-generated)
  - Session key (32-char auto-generated)
  - Anthropic API key (from environment)
  - Gmail user credential (from environment)
  - Gmail password credential (from environment)

#### 5. **Persistent Storage (Optional)**
When `create_efs: true`:
- **EFS File System** - Persistent storage for database
- **Mount Targets** - In all available subnets
- **Provisioned Throughput** - 10 MiB/s
- **Transit Encryption** - Enabled
- **Mount Point** - `/data` in Neo4j container

#### 6. **Monitoring & Logging**
- **CloudWatch Log Groups**:
  - `/ecs/{task-name}-webapp` - Web application logs
  - `/ecs/{task-name}-database` - Database logs
- **ECS Service Events** - Container start/stop events
- **Load Balancer Metrics** - Traffic and health metrics

### Container Environment Configuration

#### Web Application Environment Variables

**Standard Variables (Always Set):**
```bash
MODE=prod                           # Application mode
DB_URI=bolt://localhost:7687        # Database connection string
env=production                      # Legacy environment setting
port=8000                          # Application port
```

**Secret Variables (When create_secrets: true):**
```bash
DB_ADMIN                           # ← Database password secret
JWT_SECRET                         # ← JWT authentication secret
SESSION_KEY                        # ← Session management key
ANTHROPIC_API_KEY                  # ← AI service key
GMAIL_USER                         # ← Email integration user
GMAIL_PASS                         # ← Email integration password
```

#### Database Container Environment Variables

**When create_secrets: false:**
```bash
NEO4J_AUTH=none                    # No authentication
```

**When create_secrets: true:**
```bash
NEO4J_PASSWORD                     # ← Database password secret
```

### Networking Configuration

#### VPC & Subnets
- **Auto-Discovery**: Uses default VPC if not specified
- **Multi-AZ Deployment**: Deploys across all available subnets
- **Public IP Assignment**: Enabled for internet access

#### Security Groups
- **Default Security Group**: Used if not specified
- **Port Access**: 
  - ALB: Port 80 (HTTP)
  - Containers: Internal communication only

#### Load Balancer Configuration
- **Scheme**: Internet-facing
- **Type**: Application Load Balancer
- **Health Check**: HTTP `/health` endpoint
- **Health Check Settings**:
  - Interval: 30 seconds
  - Healthy threshold: 2 checks
  - Unhealthy threshold: 3 checks

## AWS Permissions

Required AWS IAM permissions:

### Core ECS Permissions
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecs:*",
        "ec2:DescribeVpcs",
        "ec2:DescribeSubnets", 
        "ec2:DescribeSecurityGroups",
        "elasticloadbalancing:*",
        "logs:CreateLogGroup",
        "logs:DeleteLogGroup",
        "iam:PassRole"
      ],
      "Resource": "*"
    }
  ]
}
```

### Advanced Features Permissions
```json
{
  "Effect": "Allow",
  "Action": [
    "secretsmanager:*",
    "elasticfilesystem:*"
  ],
  "Resource": "*"
}
```

### Claude AI Integration
- **Bedrock**: `bedrock:InvokeModel`, `bedrock:ListFoundationModels`

## Deployment Flow

### 1. **Basic Deployment** (`create_secrets: false`, `create_efs: false`)
```bash
./build/opsagents deploy
```

**Resources Created:**
- ECS Cluster + Service + Task Definition
- Application Load Balancer + Target Group + Listener
- CloudWatch Log Groups
- 2 Containers: Web app + Neo4j database

**Timeline:** ~3-5 minutes

### 2. **Advanced Deployment** (`create_secrets: true`, `create_efs: true`)
```bash
./build/opsagents deploy
```

**Resources Created:**
- All basic deployment resources
- 6 AWS Secrets Manager secrets
- EFS file system with mount targets
- Enhanced security and persistence

**Timeline:** ~5-8 minutes

### 3. **Auto-Discovery Features**
- **VPC/Subnets**: Automatically finds default VPC and all subnets
- **Security Groups**: Uses default security group if none specified
- **Resource Reuse**: Reuses existing load balancers and target groups
- **Health Checks**: Waits for service stability before completion

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