# Broker API Reference

## Overview

The KIDP Deployment Broker is a stateless service that manages the actual provisioning and lifecycle of resources in Kubernetes clusters. It exposes a REST API for the manager to coordinate resource deployments.

## Base URL

```
http://broker-service:8082
```

## Authentication

*TODO: Implement mutual TLS or service account token authentication*

## API Endpoints

### Health & Readiness

#### GET /health

Returns the health status of the broker service.

**Response:**
```json
{
  "status": "healthy",
  "version": "0.1.0",
  "time": "2025-10-03T09:30:00Z"
}
```

#### GET /readiness

Checks if the broker is ready to accept requests (includes Kubernetes connectivity check).

**Response:**
```json
{
  "ready": true,
  "reason": "ok",
  "version": "0.1.0"
}
```

**Response (Not Ready):**
```json
{
  "ready": false,
  "reason": "kubernetes API not accessible: connection refused",
  "version": "0.1.0"
}
```

---

### Resource Lifecycle

#### POST /v1/provision

Provisions a new resource in the target Kubernetes cluster.

**Request Body:**
```json
{
  "resourceType": "database",
  "resourceName": "postgres-app-db",
  "namespace": "team-platform",
  "team": "platform-team",
  "owner": "user@example.com",
  "callbackUrl": "http://manager:9090/v1/callback",
  "spec": {
    "engine": "postgresql",
    "version": "15",
    "size": "medium",
    "highAvailability": true,
    "backup": {
      "enabled": true,
      "retention": "7d"
    },
    "encryption": {
      "atRest": true,
      "inTransit": true
    }
  }
}
```

**Response: 202 Accepted**
```json
{
  "status": "accepted",
  "deploymentId": "deploy-fc8fc917314e2b8b698427458cd35342",
  "message": "Provisioning request accepted for database/postgres-app-db"
}
```

**Error Response: 400 Bad Request**
```json
{
  "error": "validation_failed",
  "message": "resourceName is required",
  "code": 400
}
```

#### POST /v1/deprovision

Deprovisions a resource from the target Kubernetes cluster.

**Request Body:**
```json
{
  "deploymentId": "deploy-fc8fc917314e2b8b698427458cd35342",
  "resourceType": "database",
  "resourceName": "postgres-app-db",
  "namespace": "team-platform",
  "callbackUrl": "http://manager:9090/v1/callback"
}
```

**Response: 202 Accepted**
```json
{
  "status": "accepted",
  "message": "Deprovisioning request accepted for deployment deploy-fc8fc917314e2b8b698427458cd35342"
}
```

---

### Resource State & Drift Detection

#### GET /v1/resources

Query the actual state of resources managed by the broker.

**Query Parameters:**
- `namespace` (required) - Kubernetes namespace
- `resourceType` (optional) - Filter by type (database, cache, topic)
- `resourceName` (optional) - Filter by name
- `deploymentId` (optional) - Filter by deployment ID

**Example:**
```bash
curl "http://broker:8082/v1/resources?namespace=team-platform&resourceType=database"
```

**Response: 200 OK**
```json
{
  "resources": [
    {
      "deploymentId": "deploy-abc123",
      "resourceType": "database",
      "resourceName": "postgres-app-db",
      "namespace": "team-platform",
      "phase": "Ready",
      "healthStatus": "Healthy",
      "message": "Database is running and healthy",
      "lastChecked": "2025-10-03T09:30:00Z",
      "endpoint": "postgres-app-db.team-platform.svc.cluster.local",
      "port": 5432,
      "connectionSecret": "postgres-app-db-credentials",
      "actualSpec": {
        "engine": "postgresql",
        "version": "15.2",
        "size": "medium"
      },
      "desiredSpec": {
        "engine": "postgresql",
        "version": "15",
        "size": "medium"
      },
      "driftDetected": true,
      "driftDetails": [
        "Version mismatch: desired=15, actual=15.2 (patch update)"
      ],
      "resourceUsage": {
        "cpuUsage": "250m",
        "memoryUsage": "512Mi",
        "storageUsage": "5Gi",
        "replicas": 1
      },
      "estimatedMonthlyCost": 45.50
    }
  ],
  "total": 1,
  "namespace": "team-platform"
}
```

#### POST /v1/resources

Alternative method for querying resource state using JSON body.

**Request Body:**
```json
{
  "namespace": "team-platform",
  "resourceType": "database",
  "resourceName": "postgres-app-db"
}
```

**Response:** Same as GET /v1/resources

---

### Status Queries

#### GET /v1/status

Get the status of a specific deployment.

**Query Parameters:**
- `id` (required) - Deployment ID

**Example:**
```bash
curl "http://broker:8082/v1/status?id=deploy-abc123"
```

**Response: 200 OK**
```json
{
  "deploymentId": "deploy-abc123",
  "phase": "Ready",
  "message": "Resource provisioned successfully",
  "lastUpdated": "2025-10-03T09:30:00Z"
}
```

---

## Callbacks

The broker sends asynchronous status updates to the manager's callback URL.

### POST {callbackUrl}

**Request Body:**
```json
{
  "deploymentId": "deploy-abc123",
  "resourceType": "database",
  "resourceName": "postgres-app-db",
  "namespace": "team-platform",
  "phase": "Ready",
  "message": "Database provisioned and healthy",
  "time": "2025-10-03T09:30:00Z",
  "endpoint": "postgres-app-db.team-platform.svc.cluster.local",
  "port": 5432,
  "connectionSecret": "postgres-app-db-credentials",
  "additionalMetadata": {
    "version": "15.2",
    "engine": "postgresql"
  },
  "estimatedMonthlyCost": 45.50
}
```

**Callback Phases:**
- `Provisioning` - Resource creation in progress
- `Ready` - Resource is provisioned and healthy
- `Failed` - Provisioning failed
- `Deleting` - Resource deletion in progress
- `Deleted` - Resource successfully removed

---

## Error Handling

All errors follow a consistent format:

```json
{
  "error": "error_code",
  "message": "Human-readable error description",
  "code": 400
}
```

**Common Error Codes:**
- `validation_failed` - Request validation error
- `invalid_request` - Malformed JSON or missing fields
- `resource_not_found` - Requested resource does not exist
- `provisioning_failed` - Resource creation failed
- `kubernetes_error` - Error communicating with Kubernetes API

---

## Resource Types

### Database

**Supported Engines:**
- `postgresql` (versions: 12, 13, 14, 15, 16)
- `mysql` (versions: 5.7, 8.0, 8.4)
- `mongodb` (versions: 5.0, 6.0, 7.0)
- `redis` (versions: 6.2, 7.0, 7.2)

**Size Options:**
- `small` - Development/testing (1 CPU, 2Gi RAM)
- `medium` - Production (2 CPU, 4Gi RAM)
- `large` - High-load (4 CPU, 8Gi RAM)
- `xlarge` - Enterprise (8 CPU, 16Gi RAM)

### Cache (Coming Soon)

- Redis
- Memcached

### Message Queue (Coming Soon)

- Kafka
- RabbitMQ
- NATS

---

## Rate Limiting

*TODO: Implement rate limiting to prevent broker overload*

Suggested limits:
- 100 requests per minute per source IP
- 10 concurrent provisioning operations per namespace

---

## Monitoring & Metrics

The broker exposes Prometheus metrics on port 9090:

```
# Provisioning operations
broker_provision_requests_total{resource_type="database",status="success"} 42
broker_provision_duration_seconds{resource_type="database"} 45.2

# Resource state
broker_resources_total{namespace="team-platform",type="database",phase="Ready"} 5
broker_drift_detected_total{namespace="team-platform",type="database"} 2

# Health
broker_kubernetes_api_available{} 1
broker_http_request_duration_seconds{endpoint="/v1/provision",method="POST"} 0.123
```

---

## Examples

### Provision a PostgreSQL Database

```bash
curl -X POST http://broker:8082/v1/provision \
  -H "Content-Type: application/json" \
  -d '{
    "resourceType": "database",
    "resourceName": "my-postgres",
    "namespace": "team-platform",
    "team": "platform-team",
    "owner": "alice@example.com",
    "callbackUrl": "http://manager:9090/v1/callback",
    "spec": {
      "engine": "postgresql",
      "version": "15",
      "size": "medium",
      "highAvailability": true,
      "backup": {
        "enabled": true,
        "retention": "7d",
        "schedule": "0 2 * * *"
      }
    }
  }'
```

### Check Resource State

```bash
curl "http://broker:8082/v1/resources?namespace=team-platform&resourceName=my-postgres" | jq .
```

### Deprovision a Resource

```bash
curl -X POST http://broker:8082/v1/deprovision \
  -H "Content-Type: application/json" \
  -d '{
    "deploymentId": "deploy-abc123",
    "resourceType": "database",
    "resourceName": "my-postgres",
    "namespace": "team-platform",
    "callbackUrl": "http://manager:9090/v1/callback"
  }'
```

---

## Security Considerations

1. **Network Isolation** - Broker should only be accessible from manager cluster
2. **Authentication** - Implement mutual TLS or token-based auth
3. **Authorization** - Verify caller has permissions for target namespace
4. **Input Validation** - Strict validation of all request parameters
5. **Resource Limits** - Enforce quotas and limits from Team CRD
6. **Audit Logging** - Log all provisioning operations for compliance

---

## Next Steps

- [ ] Implement actual resource provisioning logic
- [ ] Add callback mechanism to notify manager
- [ ] Implement drift detection with Kubernetes API queries
- [ ] Add authentication and authorization
- [ ] Create Prometheus metrics
- [ ] Add rate limiting
- [ ] Implement request queuing for concurrent operations
- [ ] Add support for cache and message queue resources
