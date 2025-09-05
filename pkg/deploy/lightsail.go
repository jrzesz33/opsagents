package deploy

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lightsail"
	"github.com/aws/aws-sdk-go-v2/service/lightsail/types"
)

type LightsailDeployer struct {
	client *lightsail.Client
	ctx    context.Context
}

func NewLightsailDeployer() (*LightsailDeployer, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := lightsail.NewFromConfig(cfg)
	
	return &LightsailDeployer{
		client: client,
		ctx:    context.Background(),
	}, nil
}

type ContainerServiceConfig struct {
	ServiceName   string
	Power         types.ContainerServicePowerName
	Scale         int32
	PublicDomain  string
	ContainerName string
	ImageName     string
	Ports         map[string]int32
	Environment   map[string]string
}

func (d *LightsailDeployer) CreateContainerService(config ContainerServiceConfig) error {
	fmt.Printf("Creating Lightsail container service: %s\n", config.ServiceName)
	
	input := &lightsail.CreateContainerServiceInput{
		ServiceName: aws.String(config.ServiceName),
		Power:       config.Power,
		Scale:       aws.Int32(config.Scale),
		PublicDomainNames: map[string][]string{
			config.ContainerName: {config.PublicDomain},
		},
	}

	_, err := d.client.CreateContainerService(d.ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create container service: %w", err)
	}

	fmt.Printf("Container service %s created successfully\n", config.ServiceName)
	return nil
}

func (d *LightsailDeployer) DeployContainer(serviceName string, config ContainerServiceConfig) error {
	fmt.Printf("Deploying container to service: %s\n", serviceName)
	
	ports := make(map[string]types.ContainerServiceProtocol)
	for port, portNum := range config.Ports {
		ports[port] = types.ContainerServiceProtocol(fmt.Sprintf("%d/tcp", portNum))
	}

	containers := map[string]types.Container{
		config.ContainerName: {
			Image: aws.String(config.ImageName),
			Ports: ports,
			Environment: config.Environment,
		},
	}

	publicEndpoint := &types.EndpointRequest{
		ContainerName: aws.String(config.ContainerName),
		ContainerPort: aws.Int32(8080),
		HealthCheck: &types.ContainerServiceHealthCheckConfig{
			HealthyThreshold:   aws.Int32(2),
			UnhealthyThreshold: aws.Int32(2),
			TimeoutSeconds:     aws.Int32(5),
			IntervalSeconds:    aws.Int32(30),
			Path:               aws.String("/health"),
		},
	}

	input := &lightsail.CreateContainerServiceDeploymentInput{
		ServiceName: aws.String(serviceName),
		Containers:  containers,
		PublicEndpoint: publicEndpoint,
	}

	_, err := d.client.CreateContainerServiceDeployment(d.ctx, input)
	if err != nil {
		return fmt.Errorf("failed to deploy container: %w", err)
	}

	fmt.Printf("Container deployed successfully to service: %s\n", serviceName)
	return nil
}

func (d *LightsailDeployer) GetContainerServiceState(serviceName string) (*types.ContainerService, error) {
	input := &lightsail.GetContainerServicesInput{
		ServiceName: aws.String(serviceName),
	}

	output, err := d.client.GetContainerServices(d.ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get container service state: %w", err)
	}

	if len(output.ContainerServices) == 0 {
		return nil, fmt.Errorf("container service %s not found", serviceName)
	}

	return &output.ContainerServices[0], nil
}

func (d *LightsailDeployer) WaitForServiceReady(serviceName string) error {
	fmt.Printf("Waiting for service %s to be ready...\n", serviceName)
	
	for {
		service, err := d.GetContainerServiceState(serviceName)
		if err != nil {
			return err
		}

		if service.State == types.ContainerServiceStateReady {
			fmt.Printf("Service %s is ready!\n", serviceName)
			if service.Url != nil {
				fmt.Printf("Service URL: %s\n", *service.Url)
			}
			break
		}

		fmt.Printf("Service state: %s, waiting...\n", service.State)
	}

	return nil
}