package deploy

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

type ECSDeployer struct {
	ecsClient   *ecs.Client
	ec2Client   *ec2.Client
	elbv2Client *elasticloadbalancingv2.Client
	iamClient   *iam.Client
	logsClient  *cloudwatchlogs.Client
	ctx         context.Context
}

type ECSConfig struct {
	ClusterName        string
	ServiceName        string
	TaskDefinitionName string
	VpcId              string
	SubnetIds          []string
	SecurityGroupIds   []string
	LoadBalancerName   string
	WebAppImage        string
	DatabaseImage      string
	WebAppPort         int32
	DatabasePort       int32
	WebAppMemory       int32
	WebAppCPU          int32
	DatabaseMemory     int32
	DatabaseCPU        int32
	Environment        map[string]string
}

func NewECSDeployer() (*ECSDeployer, error) {
	// Load AWS config with explicit environment variable credentials
	var cfg aws.Config
	var err error
	
	// Check if we have environment variables for AWS credentials
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	region := os.Getenv("AWS_REGION")
	
	if region == "" {
		region = "us-east-1" // Default region
	}
	
	if accessKey != "" && secretKey != "" {
		// Use static credentials from environment variables
		cfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		)
	} else {
		// Fall back to default credential chain (excluding IMDS)
		cfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(region),
			config.WithEC2IMDSClientEnableState(imds.ClientDisabled),
		)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &ECSDeployer{
		ecsClient:   ecs.NewFromConfig(cfg),
		ec2Client:   ec2.NewFromConfig(cfg),
		elbv2Client: elasticloadbalancingv2.NewFromConfig(cfg),
		iamClient:   iam.NewFromConfig(cfg),
		logsClient:  cloudwatchlogs.NewFromConfig(cfg),
		ctx:         context.Background(),
	}, nil
}

func (d *ECSDeployer) CreateCluster(clusterName string) error {
	fmt.Printf("Creating ECS cluster: %s\n", clusterName)
	
	input := &ecs.CreateClusterInput{
		ClusterName: aws.String(clusterName),
		CapacityProviders: []string{"FARGATE", "FARGATE_SPOT"},
		DefaultCapacityProviderStrategy: []types.CapacityProviderStrategyItem{
			{
				CapacityProvider: aws.String("FARGATE"),
				Weight:           aws.Int32(1),
			},
		},
	}

	_, err := d.ecsClient.CreateCluster(d.ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create ECS cluster: %w", err)
	}

	fmt.Printf("ECS cluster %s created successfully\n", clusterName)
	return nil
}

func (d *ECSDeployer) CreateTaskDefinition(config ECSConfig) error {
	fmt.Printf("Creating task definition: %s\n", config.TaskDefinitionName)

	// Create CloudWatch log groups
	webAppLogGroup := fmt.Sprintf("/ecs/%s-webapp", config.TaskDefinitionName)
	dbLogGroup := fmt.Sprintf("/ecs/%s-database", config.TaskDefinitionName)

	d.createLogGroup(webAppLogGroup)
	d.createLogGroup(dbLogGroup)

	// Task execution role ARN - this should exist or be created
	taskExecutionRoleArn := "arn:aws:iam::" + d.getAccountId() + ":role/ecsTaskExecutionRole"

	containerDefinitions := []types.ContainerDefinition{
		{
			Name:  aws.String("webapp"),
			Image: aws.String(config.WebAppImage),
			Memory: aws.Int32(config.WebAppMemory),
			Cpu:    aws.Int32(config.WebAppCPU),
			PortMappings: []types.PortMapping{
				{
					ContainerPort: aws.Int32(config.WebAppPort),
					Protocol:      types.TransportProtocolTcp,
				},
			},
			Environment: d.mapToEnvironment(config.Environment),
			LogConfiguration: &types.LogConfiguration{
				LogDriver: types.LogDriverAwslogs,
				Options: map[string]string{
					"awslogs-group":         webAppLogGroup,
					"awslogs-region":        os.Getenv("AWS_REGION"),
					"awslogs-stream-prefix": "ecs",
				},
			},
			Essential: aws.Bool(true),
		},
		{
			Name:   aws.String("database"),
			Image:  aws.String(config.DatabaseImage),
			Memory: aws.Int32(config.DatabaseMemory),
			Cpu:    aws.Int32(config.DatabaseCPU),
			PortMappings: []types.PortMapping{
				{
					ContainerPort: aws.Int32(config.DatabasePort),
					Protocol:      types.TransportProtocolTcp,
				},
			},
			Environment: []types.KeyValuePair{
				{
					Name:  aws.String("NEO4J_AUTH"),
					Value: aws.String("none"),
				},
			},
			LogConfiguration: &types.LogConfiguration{
				LogDriver: types.LogDriverAwslogs,
				Options: map[string]string{
					"awslogs-group":         dbLogGroup,
					"awslogs-region":        os.Getenv("AWS_REGION"),
					"awslogs-stream-prefix": "ecs",
				},
			},
			Essential: aws.Bool(false),
		},
	}

	input := &ecs.RegisterTaskDefinitionInput{
		Family:                  aws.String(config.TaskDefinitionName),
		NetworkMode:             types.NetworkModeAwsvpc,
		RequiresCompatibilities: []types.Compatibility{types.CompatibilityFargate},
		Cpu:                     aws.String(fmt.Sprintf("%d", config.WebAppCPU+config.DatabaseCPU)),
		Memory:                  aws.String(fmt.Sprintf("%d", config.WebAppMemory+config.DatabaseMemory)),
		ExecutionRoleArn:        aws.String(taskExecutionRoleArn),
		ContainerDefinitions:    containerDefinitions,
	}

	_, err := d.ecsClient.RegisterTaskDefinition(d.ctx, input)
	if err != nil {
		return fmt.Errorf("failed to register task definition: %w", err)
	}

	fmt.Printf("Task definition %s registered successfully\n", config.TaskDefinitionName)
	return nil
}

func (d *ECSDeployer) CreateService(config ECSConfig) error {
	fmt.Printf("Creating ECS service: %s\n", config.ServiceName)

	// Create target group for load balancer
	targetGroupArn, err := d.createTargetGroup(config)
	if err != nil {
		return fmt.Errorf("failed to create target group: %w", err)
	}

	input := &ecs.CreateServiceInput{
		ServiceName:    aws.String(config.ServiceName),
		Cluster:        aws.String(config.ClusterName),
		TaskDefinition: aws.String(config.TaskDefinitionName),
		DesiredCount:   aws.Int32(1),
		LaunchType:     types.LaunchTypeFargate,
		NetworkConfiguration: &types.NetworkConfiguration{
			AwsvpcConfiguration: &types.AwsVpcConfiguration{
				Subnets:        config.SubnetIds,
				SecurityGroups: config.SecurityGroupIds,
				AssignPublicIp: types.AssignPublicIpEnabled,
			},
		},
		LoadBalancers: []types.LoadBalancer{
			{
				TargetGroupArn: aws.String(targetGroupArn),
				ContainerName:  aws.String("webapp"),
				ContainerPort:  aws.Int32(config.WebAppPort),
			},
		},
	}

	_, err = d.ecsClient.CreateService(d.ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create ECS service: %w", err)
	}

	fmt.Printf("ECS service %s created successfully\n", config.ServiceName)
	return nil
}

func (d *ECSDeployer) GetServiceStatus(clusterName, serviceName string) (string, error) {
	input := &ecs.DescribeServicesInput{
		Cluster:  aws.String(clusterName),
		Services: []string{serviceName},
	}

	output, err := d.ecsClient.DescribeServices(d.ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to describe service: %w", err)
	}

	if len(output.Services) == 0 {
		return "", fmt.Errorf("service %s not found", serviceName)
	}

	service := output.Services[0]
	status := fmt.Sprintf("Service: %s\nStatus: %s\nRunning: %d\nPending: %d\nDesired: %d",
		*service.ServiceName,
		service.Status,
		service.RunningCount,
		service.PendingCount,
		service.DesiredCount)

	return status, nil
}

func (d *ECSDeployer) WaitForServiceStable(clusterName, serviceName string) error {
	fmt.Printf("Waiting for service %s to be stable...\n", serviceName)
	
	waiter := ecs.NewServicesStableWaiter(d.ecsClient)
	input := &ecs.DescribeServicesInput{
		Cluster:  aws.String(clusterName),
		Services: []string{serviceName},
	}

	err := waiter.Wait(d.ctx, input, 10*time.Minute)
	if err != nil {
		return fmt.Errorf("failed waiting for service to be stable: %w", err)
	}

	fmt.Printf("Service %s is now stable!\n", serviceName)
	return nil
}

func (d *ECSDeployer) createTargetGroup(config ECSConfig) (string, error) {
	fmt.Printf("Creating target group for service: %s\n", config.ServiceName)

	input := &elasticloadbalancingv2.CreateTargetGroupInput{
		Name:                       aws.String(fmt.Sprintf("%s-tg", config.ServiceName)),
		Protocol:                   elbv2types.ProtocolEnumHttp,
		Port:                       aws.Int32(config.WebAppPort),
		VpcId:                      aws.String(config.VpcId),
		TargetType:                 elbv2types.TargetTypeEnumIp,
		HealthCheckProtocol:        elbv2types.ProtocolEnumHttp,
		HealthCheckPath:            aws.String("/health"),
		HealthCheckIntervalSeconds: aws.Int32(30),
		HealthyThresholdCount:      aws.Int32(2),
		UnhealthyThresholdCount:    aws.Int32(3),
	}

	output, err := d.elbv2Client.CreateTargetGroup(d.ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to create target group: %w", err)
	}

	return *output.TargetGroups[0].TargetGroupArn, nil
}

func (d *ECSDeployer) createLogGroup(logGroupName string) error {
	input := &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(logGroupName),
	}

	_, err := d.logsClient.CreateLogGroup(d.ctx, input)
	if err != nil {
		// Log group might already exist, which is fine
		fmt.Printf("Log group %s might already exist: %v\n", logGroupName, err)
	}

	return nil
}

func (d *ECSDeployer) mapToEnvironment(envMap map[string]string) []types.KeyValuePair {
	var env []types.KeyValuePair
	for key, value := range envMap {
		env = append(env, types.KeyValuePair{
			Name:  aws.String(key),
			Value: aws.String(value),
		})
	}
	return env
}

func (d *ECSDeployer) getAccountId() string {
	// This is a placeholder - in a real implementation, you'd get this from STS
	// For now, return the account ID from your .env or a default
	return "944945738659" // Replace with actual account ID retrieval
}