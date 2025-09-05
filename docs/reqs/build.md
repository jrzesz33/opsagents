# Pull and Build Application

## Steps this Agent Conducts

1. Pull from the Github Repo https://github.com/jrzesz33/bigfootgolf
    Within the main branch, the system shall utilize the Github Remote MCP server to Pull the latest files

2. Build the Web Go App
    The system shall build the Web Go App as a binary

3. Build the Docker Images
    - The system shall use template Dockerfiles for the images
    - The system shall build and manage the configuration and secrets for these systems running
    - A docker image utilizing the latest base images from Chainguard for Go applications with the Web Binary
    - A docker image that is running neo4j within a container

## Technology Stacks
- Chainguard for Base Images
