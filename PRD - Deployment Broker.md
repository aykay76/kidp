# PRD: Deployment Broker

## Overview
Deployment Brokers are stateless workers that execute deployments on specific cloud providers and regions. They translate abstract resource specifications into cloud-specific API calls.

## Objectives
- Execute cloud-specific resource provisioning
- Provide consistent webhook API interface
- Report status back to management cluster
- Support multiple cloud providers with pluggable architecture

## Architecture

### Stateless Design
- **No local persistence** - all state in management cluster
- **Idempotent operations** - safe to retry any deployment
- **Horizontal scaling** - multiple broker instances per region
- **Cloud-specific logic** - encapsulate provider APIs

### Communication Pattern
```
Operator → Webhook API → Broker → Cloud API
Cloud Status → Broker → Callback → Operator Status
```

## Webhook API Implementation

### Base Requirements
- **HTTP/2 support** for performance
- **TLS 1.3** with cert-manager integration
- **JWT authentication** with signature verification
- **Rate limiting** to prevent abuse
- **Request validation** with OpenAPI schema

### Endpoints

#### POST /api/v1/deploy
**Purpose**: Initiate resource deployment

**Request Validation**:
- Valid JWT token with deploy scope
- Resource spec matches schema for resource type
- Target matches broker's assigned region/cloud
- No duplicate deployment for same resource UID

**Processing**:
1. Generate unique deploymentId
2. Validate cloud quota and permissions
3. Translate spec to cloud-specific parameters
4. Initiate cloud API call (async)
5. Return deploymentId immediately

**Response SLA**: < 500ms

#### GET /api/v1/deployments/{deploymentId}
**Purpose**: Check deployment progress

**Response**:
- Current status and phase
- Progress percentage (if available)
- Error messages if failed
- Estimated completion time

**Caching**: 30 second cache for completed deployments

#### DELETE /api/v1/resources/{resourceType}/{resourceName}
**Purpose**: Delete cloud resource

**Processing**:
1. Lookup cloud resource by metadata tags
2. Initiate deletion via cloud API
3. Wait for deletion confirmation
4. Clean up associated resources (secrets, backups)
5. Send completion callback

## Cloud Provider Plugins

### Plugin Interface
```go
type CloudProvider interface {
    // Deploy creates or updates a resource
    Deploy(ctx context.Context, spec ResourceSpec) (DeploymentResult, error)
    
    // GetStatus checks deployment progress
    GetStatus(ctx context.Context, deploymentId string) (DeploymentStatus, error)
    
    // Delete removes a resource
    Delete(ctx context.Context, resourceId string) error
    
    // ValidateQuota checks if deployment is possible
    ValidateQuota(ctx context.Context, spec ResourceSpec) error
}
```

### Azure Plugin
**Supported Resources**:
- Azure SQL Database / Managed Instance
- Azure Cache for Redis
- Azure Container Apps
- Azure Service Bus Topics
- Azure Storage Accounts

**Implementation**:
- Azure SDK for Go
- Managed Identity for authentication
- Resource tags for tracking
- ARM template validation

### AWS Plugin
**Supported Resources**:
- RDS (PostgreSQL, MySQL, Aurora)
- ElastiCache (Redis, Memcached)
- ECS/Fargate services
- MSK (Kafka) topics
- S3 buckets

**Implementation**:
- AWS SDK v2 for Go
- IAM role authentication
- CloudFormation for complex resources
- Resource tagging strategy

### GCP Plugin
**Supported Resources**:
- Cloud SQL
- Memorystore (Redis)
- Cloud Run services
- Pub/Sub topics
- Cloud Storage buckets

**Implementation**:
- Google Cloud Go SDK
- Service account authentication
- Deployment Manager templates
- Labels for resource tracking

### On-Premises Plugin
**Supported Resources**:
- Kubernetes deployments (via kubectl)
- PostgreSQL via operator (zalando/postgres-operator)
- Redis via operator (spotahome/redis-operator)
- Kafka via Strimzi operator

**Implementation**:
- Kubernetes client-go library
- ServiceAccount authentication
- CRD-based resource management

## Status Callback Mechanism

### Callback Requirements
- **Retry logic**: Exponential backoff (1s, 2s, 4s, 8s, 16s)
- **Timeout**: 30 seconds per attempt
- **Max retries**: 5 attempts
- **Dead letter queue**: Failed callbacks logged for manual investigation

### Callback Timing
- **Immediate**: On deployment acceptance
- **Progress**: Every 30 seconds during deployment
- **Completion**: On success or failure
- **Error**: On any error condition

### Callback Payload
```json
{
  "deploymentId": "dep_abc123",
  "status": "ready",
  "connectionInfo": {
    "endpoint": "...",
    "secretRef": {...}
  },
  "metadata": {
    "cloudResourceId": "...",
    "cost": {...}
  }
}
```

## Resource Tracking

### Metadata Tags/Labels
All cloud resources created by broker must include:
```
platform.company.com/managed-by: kidp
platform.company.com/deployment-id: dep_abc123
platform.company.com/resource-uid: 550e8400-e29b-41d4-a716-446655440000
platform.company.com/team: backend-team
platform.company.com/application: user-service
```

### Resource Discovery
- Periodic scan of cloud resources with KIDP tags
- Detect orphaned resources (no matching CRD)
- Alert on untagged resources in KIDP-managed resource groups

## Secret Management

### Connection Credentials
1. **Generate** strong passwords/keys during provisioning
2. **Store** in cloud-native secret store (Azure Key Vault, AWS Secrets Manager)
3. **Reference** in callback via secretRef
4. **Rotate** credentials on schedule (90 days)

### Secret Synchronization
- Broker creates K8s Secret in management cluster namespace
- ExternalSecrets operator syncs to target clusters
- Automatic rotation updates both cloud and K8s secrets

## Cost Tracking

### Resource Cost Attribution
```json
{
  "cost": {
    "estimated_monthly": 150.00,
    "currency": "USD",
    "breakdown": {
      "compute": 100.00,
      "storage": 30.00,
      "network": 20.00
    }
  }
}
```

### Cost Reporting
- Include in every status callback
- Aggregate by team/application labels
- Export to central cost management system
- Alert on unexpected cost increases (>20% week-over-week)

## Security

### Authentication
- **Inbound**: JWT tokens from management cluster operators
- **Outbound**: Cloud provider credentials via managed identity
- **Secret storage**: Encrypted environment variables or K8s secrets

### Network Security
```yaml
# Only allow traffic from management cluster
NetworkPolicy:
  ingress:
    - from:
      - ipBlock:
          cidr: 10.0.0.0/16  # Management cluster CIDR
      ports:
        - port: 8443
```

### Audit Logging
- All API calls logged with request/response
- Correlation IDs for tracing
- Failed authentication attempts
- Cloud API calls and responses

## Observability

### Metrics
```
kidp_broker_requests_total{method, status}
kidp_broker_request_duration_seconds{method}
kidp_broker_cloud_api_calls_total{provider, resource_type, operation}
kidp_broker_callback_attempts_total{status}
kidp_broker_active_deployments
```

### Health Checks
```
GET /health
  - Cloud API connectivity
  - Management cluster reachability
  - Resource quota availability
  
GET /ready
  - Webhook server ready to accept requests
```

### Logging
- Structured JSON logs
- Log levels: DEBUG, INFO, WARN, ERROR
- Request correlation IDs
- Cloud API response details (sanitized)

## Deployment

### Kubernetes Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kidp-broker-azure
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: broker
        image: kidp/broker-azure:v1.0.0
        resources:
          requests:
            cpu: 500m
            memory: 1Gi
          limits:
            cpu: 2
            memory: 4Gi
        env:
        - name: CLOUD_PROVIDER
          value: "azure"
        - name: REGION
          value: "westus2"
```

### Configuration
```yaml
# ConfigMap
config:
  webhook:
    port: 8443
    tlsCert: /etc/certs/tls.crt
    tlsKey: /etc/certs/tls.key
  management:
    clusterUrl: https://management.company.com
    callbackPath: /api/v1/status
  cloud:
    quotaCheckEnabled: true
    defaultTimeout: 30m
    retryAttempts: 3
```

## Testing

### Unit Tests
- Cloud provider plugin interfaces
- Request validation logic
- Callback retry mechanisms

### Integration Tests
- Real cloud API calls (test accounts)
- Full deployment lifecycle
- Error handling scenarios

### Load Tests
- 100 concurrent deployments
- Sustained load over 1 hour
- Measure latency and error rates

## Success Metrics
- **API latency**: < 500ms for 95% of requests
- **Deployment success rate**: > 98%
- **Callback success rate**: > 99%
- **Cloud API error rate**: < 2%

## Future Enhancements
- Multi-region failover
- Cost optimization recommendations
- Predictive quota management
- Advanced deployment strategies (blue/green, canary)