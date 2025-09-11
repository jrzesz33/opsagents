# OpsAgents Configuration Specification

This document specifies all configuration requirements for OpsAgents AWS ECS deployments.

## Container Applications

### Web Application Container
- **Image**: `images.app_image` (configurable in config.yaml)
- **Default Port**: 8000
- **CPU**: 256 units (configurable)
- **Memory**: 512 MB (configurable)
- **Health Check**: Expects `/health` endpoint
- **Essential**: Yes (service fails if this container stops)

### Neo4j Database Container  
- **Image**: `images.neo4j_image` (configurable in config.yaml)
- **Ports**: 
  - 7474 (HTTP interface)
  - 7687 (Bolt protocol)
- **CPU**: 256 units (configurable)
- **Memory**: 512 MB (configurable)
- **Essential**: No (allows graceful degradation)
- **Persistence**: Optional EFS volume mount to `/data`

## Environment Variables Configuration

### Web Application Environment Variables

#### Always Set (Basic & Advanced Deployments)
| Variable | Value | Description |
|----------|-------|-------------|
| `MODE` | `"prod"` | Application mode (prod/dev/test) |
| `DB_URI` | `"bolt://localhost:7687"` | Neo4j connection string |
| `env` | From config.yaml `environment.env` | Legacy environment setting |
| `port` | From config.yaml `environment.port` | Application port |

#### Secret Variables (Advanced Deployment Only)
When `create_secrets: true`, these are injected from AWS Secrets Manager:

| Variable | Source Secret | Description |
|----------|---------------|-------------|
| `DB_ADMIN` | Database password secret | Neo4j authentication |
| `JWT_SECRET` | JWT secret | Authentication token signing |
| `SESSION_KEY` | Session key | Session management |
| `ANTHROPIC_API_KEY` | Anthropic API secret | AI service integration |
| `GMAIL_USER` | Gmail user secret | Email integration user |
| `GMAIL_PASS` | Gmail password secret | Email integration password |

### Database Container Environment Variables

#### Basic Deployment (`create_secrets: false`)
| Variable | Value | Description |
|----------|-------|-------------|
| `NEO4J_AUTH` | `"none"` | Disables authentication |

#### Advanced Deployment (`create_secrets: true`)
| Variable | Source | Description |
|----------|--------|-------------|
| `NEO4J_PASSWORD` | Database password secret | Enables secure authentication |

## AWS Secrets Management

### Secret Creation (create_secrets: true)

#### Auto-Generated Secrets
| Secret Name | Type | Length | Usage |
|-------------|------|--------|-------|
| `{service-name}-db-password` | Auto-generated | 32 chars | Neo4j database password |
| `{service-name}-jwt-secret` | Auto-generated | 64 chars | JWT token signing |
| `{service-name}-session-key` | Auto-generated | 32 chars | Session management |

#### Environment-Sourced Secrets
| Secret Name | Environment Variable | Required For Deployment |
|-------------|---------------------|------------------------|
| `{service-name}-anthropic-key` | `ANTHROPIC_API_KEY` | Optional |
| `{service-name}-gmail-user` | `GMAIL_USER` | Optional |  
| `{service-name}-gmail-pass` | `GMAIL_PASS` | Optional |

**Note**: Environment-sourced secrets are only created if the corresponding environment variable is set during deployment.

### Secret Naming Convention
```
{service-name}-{secret-type}
```
Examples:
- `bigfootgolf-service-db-password`
- `bigfootgolf-service-jwt-secret`

## Persistent Storage Configuration

### EFS File System (create_efs: true)
- **Performance Mode**: General Purpose
- **Throughput Mode**: Provisioned (10 MiB/s)
- **Transit Encryption**: Enabled
- **Mount Targets**: Created in all available subnets
- **Mount Point**: `/data` in Neo4j container
- **Purpose**: Persistent storage for Neo4j database files

### Volume Configuration
```yaml
volumes:
  - name: neo4j-data
    efsVolumeConfiguration:
      fileSystemId: {auto-created-efs-id}
      rootDirectory: "/"
      transitEncryption: ENABLED
```

## Networking Configuration

### VPC & Subnets
- **Auto-Discovery**: Uses default VPC if `vpc_id` not specified
- **Multi-AZ**: Deploys across all available subnets
- **Public IP**: Enabled for internet access

### Load Balancer Configuration
- **Type**: Application Load Balancer (ALB)
- **Scheme**: Internet-facing
- **Ports**: 80 (HTTP)
- **Health Check**:
  - Path: `/health`
  - Interval: 30 seconds
  - Healthy threshold: 2
  - Unhealthy threshold: 3

### Security Groups
- **Default**: Uses VPC default security group if not specified
- **Ports**: 
  - ALB: 80 (HTTP inbound from internet)
  - Containers: Internal communication only

## Configuration Deployment Matrix

| Feature | Basic | Advanced |
|---------|-------|----------|
| **Containers** | ✅ Web + DB | ✅ Web + DB |
| **Load Balancer** | ✅ ALB + Target Group | ✅ ALB + Target Group |
| **Secrets Manager** | ❌ | ✅ 6 secrets |
| **EFS Persistence** | ❌ | ✅ (optional) |
| **Database Auth** | None | Password-based |
| **Environment Variables** | 4 standard | 4 standard + 6 secrets |
| **Deployment Time** | ~3-5 min | ~5-8 min |

## Required Environment Variables for Deployment

### AWS Credentials (Required)
```bash
AWS_ACCESS_KEY_ID="your-access-key"
AWS_SECRET_ACCESS_KEY="your-secret-key" 
AWS_REGION="us-east-1"
```

### Advanced Features (Optional)
```bash
ANTHROPIC_API_KEY="your-anthropic-key"  # For AI integration
GMAIL_USER="your-gmail-user"            # For email features
GMAIL_PASS="your-gmail-app-password"    # For email features
```

## Configuration File Structure

### config.yaml Template
```yaml
aws:
  ecs:
    # Basic Configuration
    cluster_name: "your-cluster"
    service_name: "your-service" 
    task_definition_name: "your-task"
    webapp_port: 8000
    database_port: 7687
    database_http_port: 7474
    
    # Resource Allocation
    webapp_memory: 512
    webapp_cpu: 256
    database_memory: 512
    database_cpu: 256
    
    # Advanced Features
    create_secrets: false    # Enable AWS Secrets Manager
    create_efs: false       # Enable EFS persistent storage
    mode: "prod"            # Application mode
    
    # Custom Environment Variables
    environment:
      ENV: "production"
      PORT: "8000"

images:
  app_image: "your-registry/app:tag"
  neo4j_image: "your-registry/neo4j:tag"
```