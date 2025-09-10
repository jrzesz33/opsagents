# Application Configuration

## Secrets Management
    1. A Secret needs to be generated and stored for the Database and also used by the Web Application
    2. A Secret needs to be generated for the JWT secret key and used by the web application
    3. A Secret needs to be generated for the Session Key for the Session Manager
The below Secrets need to be mapped to from this applications environment variables
    4. A Secret needs to be created from this applications Environment variable called ANTHROPIC_API_KEY for the Anthropic API Key
    5. A Secret needs to be created from this applications Environment variable called GMAIL_USER for the GMail App Key for email integration
    6. A Secret needs to be created from this applications Environment variable called GMAIL_PASS for the GMail App Secret for email integration

## Configuration Management
    1. A configuration (environment) variable needs to be created for MODE and is defaulted to "prod"

## Database Configuration
    1. The Database secret should be mapped to the NEO4J_PASSWORD environment Variable for the database Container
    2. The Database container Requires the container to be mounted to a persisted volume (EFS) in the /data directory of the container
    3. The Database container is going to need to expose ports 7474 and 7687

## Web App Configuration
    1. The Database secret should be mapped to the DB_ADMIN environment variable
    2. The JWT Secret should map to the JWT_SECRET environment variable
    3. The Anthropic API Key should be mapped to the ANTHROPIC_API_KEY variable
    4. The Session Key should be mapped to the SESSION_KEY variable
    5. The GMAIL app key should be mapped to the GMAIL_USER variable
    6. The GMAIL App Secret should be mapped to the GMAIL_PASS variable