# Drift Detection and Resource State Management

## Overview

The broker implements a comprehensive resource state API that enables the manager to detect and reconcile configuration drift between desired state (CRD specs) and actual state (deployed Kubernetes resources).

## Architecture

```
┌─────────────┐           ┌─────────────┐           ┌─────────────┐
│   Manager   │           │   Broker    │           │ Kubernetes  │
│  (CRDs)     │           │             │           │   Cluster   │
└──────┬──────┘           └──────┬──────┘           └──────┬──────┘
       │                         │                         │
       │  POST /v1/provision     │                         │
       │────────────────────────>│  Create Resources       │
       │  202 Accepted           │────────────────────────>│
       │<────────────────────────│                         │
       │                         │                         │
       │                         │  POST /v1/callback      │
       │<────────────────────────│  (status: Ready)        │
       │                         │                         │
       │  GET /v1/resources      │                         │
       │────────────────────────>│  Query Actual State     │
       │  200 OK (ResourceState) │────────────────────────>│
       │<────────────────────────│<────────────────────────│
       │                         │                         │
       │  Compare & Detect Drift │                         │
       │─────────────────>       │                         │
       │                         │                         │
       │  POST /v1/provision     │                         │
       │  (if drift detected)    │  Update Resources       │
       │────────────────────────>│────────────────────────>│
```

## API Endpoints

### GET /v1/resources

Query the actual state of resources managed by the broker.

**Query Parameters:**
- `namespace` (required) - Kubernetes namespace to query
- `resourceType` (optional) - Filter by resource type (database, cache, topic)
- `resourceName` (optional) - Filter by resource name
- `deploymentId` (optional) - Filter by deployment ID

**Example:**
```bash
curl "http://broker:8082/v1/resources?namespace=team-platform&resourceType=database"
```

### POST /v1/resources

Alternative method using JSON body for complex queries.

**Request Body:**
```json
{
  "namespace": "team-platform",
  "resourceType": "database",
  "resourceName": "postgres-app-db"
}
```

**Response:**
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
        "size": "medium",
        "replicas": 1
      },
      "desiredSpec": {
        "engine": "postgresql",
        "version": "15",
        "size": "medium",
        "replicas": 1
      },
      "driftDetected": true,
      "driftDetails": [
        "Version mismatch: desired=15, actual=15.2",
        "This is expected patch version drift"
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

## Drift Detection Strategy

### 1. Periodic Reconciliation

The manager controller should periodically query resource state:

```go
// In the Database controller reconcile loop
func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // ... get Database CRD ...
    
    // Query actual state from broker
    actualState, err := r.BrokerClient.GetResourceState(ctx, broker.ResourceStateRequest{
        Namespace:    db.Namespace,
        ResourceType: "database",
        ResourceName: db.Name,
        DeploymentID: db.Status.DeploymentID,
    })
    
    if err != nil {
        return ctrl.Result{}, err
    }
    
    // Detect drift
    if actualState.DriftDetected {
        log.Info("Drift detected", "resource", db.Name, "details", actualState.DriftDetails)
        
        // Update status with drift information
        db.Status.Phase = "DriftDetected"
        db.Status.Message = strings.Join(actualState.DriftDetails, "; ")
        
        // Decide whether to auto-remediate or alert
        if r.shouldAutoRemediate(actualState) {
            // Re-provision with correct spec
            return r.reprovision(ctx, db)
        }
    }
    
    // Requeue for next check (e.g., every 5 minutes)
    return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}
```

### 2. Types of Drift

**Expected Drift:**
- Patch version updates (15 → 15.2)
- Minor resource usage variations
- Kubernetes-managed metadata changes

**Unexpected Drift:**
- Manual configuration changes
- Resource deletions
- Major version changes
- Size/capacity modifications

### 3. Remediation Strategies

**Auto-Remediate:**
- Configuration changes
- Missing resources
- Incorrect scaling

**Alert Only:**
- Major version differences
- Cost-impacting changes
- Data-affecting modifications

### 4. Health Status

The broker reports three health levels:

- **Healthy** - Resource is running as expected
- **Degraded** - Resource is running but with issues (high CPU, connection errors)
- **Unhealthy** - Resource is not functioning (pod crashes, no connectivity)

## Implementation Notes

### Broker Side

The broker needs to:

1. **Query Kubernetes API** - Get actual resource state (StatefulSets, Services, Secrets)
2. **Store Desired State** - Keep track of what was requested via provision calls
3. **Compare States** - Detect differences and categorize severity
4. **Report Metrics** - Include resource usage if available
5. **Calculate Health** - Check pod status, connectivity, readiness probes

### Manager Side

The manager should:

1. **Poll Periodically** - Query broker for resource state (e.g., every 5 minutes)
2. **Update CRD Status** - Reflect actual state in the Database CRD status
3. **Emit Events** - Create Kubernetes events for drift detection
4. **Alert/Remediate** - Based on policy, fix drift or notify operators

## Example Workflow

1. **User creates Database CRD:**
   ```yaml
   apiVersion: platform.company.com/v1
   kind: Database
   spec:
     engine: postgresql
     version: "15"
     size: medium
   ```

2. **Manager provisions via broker:**
   - POST /v1/provision with spec
   - Receives deploymentId
   - Updates Database.status.deploymentId

3. **Broker creates resources:**
   - StatefulSet with PostgreSQL 15
   - Service for connectivity
   - Secret with credentials
   - Calls back to manager with Ready status

4. **Periodic drift check (5 min later):**
   - Manager: GET /v1/resources?namespace=team-platform
   - Broker returns actual state including resource usage

5. **Drift detected:**
   - User manually scaled StatefulSet replicas from 1 to 3
   - Broker reports: `driftDetected: true, driftDetails: ["Replica count mismatch: desired=1, actual=3"]`

6. **Manager responds:**
   - Updates Database.status.phase = "DriftDetected"
   - Emits Kubernetes event
   - Based on policy, either:
     - **Auto-remediate:** Scale back to 1 replica
     - **Alert only:** Notify operators via Slack/PagerDuty

## Future Enhancements

1. **Drift History** - Track drift events over time
2. **Drift Metrics** - Prometheus metrics for monitoring
3. **Remediation Policies** - Configurable auto-remediation rules
4. **Change Management** - Require approval for certain remediations
5. **Drift Prevention** - OPA policies to block manual changes
6. **Cost Tracking** - Alert on cost-increasing drift

## Benefits

- **Compliance** - Ensure deployed resources match approved specs
- **Security** - Detect unauthorized changes
- **Cost Control** - Prevent surprise infrastructure costs
- **Reliability** - Maintain consistent configurations
- **Observability** - Complete visibility into actual vs desired state
