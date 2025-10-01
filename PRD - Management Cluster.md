# PRD: Management Cluster

## Overview
The Management Cluster is the central control plane for KIDP, running all operators and storing all resource state as the single source of truth.

## Objectives
- Provide centralized state management for all platform resources
- Enable GitOps-driven infrastructure management
- Support multi-cloud resource orchestration
- Maintain resource relationship graphs

## Core Components

### 1. etcd State Store
**Purpose**: Store desired state for all platform resources

**Requirements**:
- High availability (3+ node cluster)
- Automated backup and restore
- Encryption at rest
- Regular snapshots

### 2. Operator Framework
**Purpose**: Reconcile desired state with actual state

**Requirements**:
- Support for multiple resource types (CRDs)
- Reconciliation loop pattern
- Status reporting and conditions
- Finalizer support for cleanup
- Leader election for HA

### 3. GitOps Integration (FluxCD)
**Purpose**: Sync Git repository state to cluster

**Requirements**:
- Monitor Git repositories for changes
- Apply manifests to cluster automatically
- Support for multiple Git sources
- Kustomization and Helm support
- Drift detection and remediation

### 4. Status API
**Purpose**: Receive status updates from deployment brokers

**Requirements**:
- REST API endpoint for broker callbacks
- JWT authentication validation
- Update CRD status fields
- Emit K8s events
- Retry logic for failed updates

## Resource Types (CRDs)

### Metadata Resources (K8s-only)
```yaml
Team:
  - Identity and ownership boundary
  - RBAC integration
  - Cost allocation tags

Application:
  - Deployment unit
  - Contains multiple services
  - Environment management

Policy:
  - Governance rules
  - CEL expressions
  - Validation webhooks
```

### Deployable Resources (Broker-managed)
```yaml
Database:
  - Multi-cloud database abstraction
  - Connection secret management
  - Backup configuration

Service:
  - Container deployment
  - Load balancer integration
  - Auto-scaling configuration

Cache:
  - Redis/Memcached instances
  - Size and eviction policies

Topic:
  - Kafka topic management
  - Partition and replication config
```

## CRD Common Patterns

### Base Structure
```yaml
apiVersion: platform.company.com/v1
kind: <ResourceType>
metadata:
  name: resource-name
  namespace: team-namespace
  labels:
    platform.company.com/managed-by: kidp
    platform.company.com/team: team-backend
spec:
  owner:
    kind: Team
    name: backend-team
  # Resource-specific configuration
status:
  phase: pending|ready|failed|unknown
  conditions:
    - type: Ready
      status: "True"
      lastTransitionTime: "2024-10-01T10:30:00Z"
      reason: ResourceReady
      message: "Resource is operational"
  # Resource-specific status
```

## Operator Behavior

### Reconciliation Logic
1. **Read desired state** from CRD spec
2. **Check current status** from CRD status
3. **For metadata resources**: Update K8s state directly
4. **For deployable resources**: 
   - Call broker webhook API
   - Update status with deploymentId
   - Wait for broker callback
5. **Update status conditions** based on results
6. **Requeue if needed** for retry logic

### Error Handling
- **Transient errors**: Exponential backoff retry
- **Permanent errors**: Update status with error condition
- **Timeout errors**: Mark as unknown, investigate manually
- **Quota errors**: Surface to user, require intervention

### Finalizer Pattern
```yaml
metadata:
  finalizers:
    - platform.company.com/cleanup-resources
```
- Delete cloud resources before removing CRD
- Wait for broker confirmation
- Remove finalizer only after successful cleanup

## Security

### RBAC Model
```yaml
# Team members can manage their own resources
Role: team-developer
  - create/read/update/delete: Applications, Services
  - read: Databases, Caches, Topics (in their namespace)

# Platform admins manage infrastructure
ClusterRole: platform-admin
  - all operations on all CRDs
  - operator deployment and configuration
```

### Authentication
- **Internal**: ServiceAccount tokens for operators
- **External**: JWT tokens for broker callbacks
- **User**: SSO/LDAP integration via K8s auth

### Network Policies
```yaml
# Only allow broker traffic to status API
NetworkPolicy:
  - Allow ingress from broker IPs to status API
  - Deny all other external ingress
  - Allow operator to broker egress
```

## Observability

### Metrics (Prometheus)
```
kidp_reconciliation_duration_seconds
kidp_resource_count{type, status}
kidp_broker_call_duration_seconds
kidp_broker_call_errors_total
```

### Logging
- Structured JSON logs
- Request tracing with correlation IDs
- Operator reconciliation events
- Status update audit trail

### Alerting
- Operator crash loops
- Reconciliation failures exceeding threshold
- Broker communication failures
- Resource stuck in pending state > 10 minutes

## Deployment

### Infrastructure Requirements
- **Kubernetes**: 1.28+
- **Nodes**: 3+ for HA
- **Resources**: 8 CPU, 16GB RAM minimum
- **Storage**: 100GB for etcd
- **Network**: Static IPs for broker callbacks

### Installation
```bash
# Install CRDs
kubectl apply -f crds/

# Install operators
helm install kidp-operators ./charts/operators

# Install FluxCD
flux install

# Configure Git source
flux create source git kidp-config \
  --url=https://github.com/company/kidp-config \
  --branch=main
```

## Testing Strategy

### Unit Tests
- CRD validation logic
- Operator reconciliation functions
- Status update handling

### Integration Tests
- Full operator lifecycle with test brokers
- GitOps sync workflows
- RBAC enforcement

### E2E Tests
- Create resource → Deploy → Delete lifecycle
- Resource relationship validation
- Failure recovery scenarios

## Success Metrics
- **Reconciliation time**: < 30 seconds for 95% of resources
- **Broker call success rate**: > 99%
- **Status update latency**: < 5 seconds
- **Resource drift detection**: < 1 minute

## Future Enhancements
- Multi-cluster federation
- Advanced scheduling (affinity rules)
- Resource cost optimization recommendations
- Automated capacity planning