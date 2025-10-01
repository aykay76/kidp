# PRD: Observability & Monitoring

## Overview
Comprehensive observability stack for monitoring platform health, resource status, deployment progress, cost tracking, and security compliance.

## Objectives
- Provide real-time visibility into platform operations
- Enable proactive issue detection and resolution
- Track resource costs and optimize spending
- Maintain security and compliance posture
- Support capacity planning and forecasting

## Monitoring Stack

### Core Components

#### 1. Metrics (Prometheus)
**Purpose**: Time-series metrics for all platform components

**Data Sources**:
- **Operators**: Reconciliation metrics, error rates
- **Brokers**: Deployment success rates, API latency
- **Resources**: Status conditions, phase transitions
- **Infrastructure**: K8s cluster metrics, etcd health

**Retention**:
- Raw metrics: 15 days
- Aggregated (5m): 90 days
- Long-term (1h): 2 years (Thanos/Cortex)

#### 2. Logging (Loki)
**Purpose**: Centralized log aggregation

**Log Sources**:
- Operator reconciliation logs
- Broker deployment execution
- Cloud API interactions
- Webhook requests/responses
- GitOps sync operations

**Retention**:
- Debug logs: 7 days
- Info/Warn logs: 30 days
- Error logs: 90 days
- Audit logs: 1 year

#### 3. Tracing (Jaeger/Tempo)
**Purpose**: Distributed tracing for deployment flows

**Trace Spans**:
```
Deployment Request
├── Git Commit Detected
├── FluxCD Reconciliation
├── Operator Reconciliation
│   ├── Validation
│   ├── Broker API Call
│   │   ├── Authentication
│   │   ├── Cloud API Calls
│   │   └── Status Callback
│   └── Status Update
└── Resource Ready
```

**Retention**: 7 days (sampled at 10%)

#### 4. Dashboards (Grafana)
**Purpose**: Visualization and alerting

**Dashboard Categories**:
- Platform Overview
- Resource Status
- Team/Application Views
- Cost Tracking
- Security & Compliance
- Capacity Planning

## Key Metrics

### Platform Health Metrics

#### Operator Metrics
```promql
# Reconciliation duration
kidp_operator_reconcile_duration_seconds{controller, result}

# Reconciliation rate
rate(kidp_operator_reconcile_total[5m])

# Error rate
rate(kidp_operator_reconcile_errors_total[5m])

# Queue depth
kidp_operator_queue_depth{controller}

# CRD count by status
kidp_resource_count{kind, status}
```

#### Broker Metrics
```promql
# Deployment success rate
rate(kidp_broker_deployments_total{status="success"}[5m]) /
rate(kidp_broker_deployments_total[5m])

# API request latency
histogram_quantile(0.95, kidp_broker_api_duration_seconds)

# Active deployments
kidp_broker_active_deployments{cloud_provider, region}

# Cloud API errors
rate(kidp_broker_cloud_api_errors_total[5m])

# Callback success rate
rate(kidp_broker_callbacks_total{status="success"}[5m]) /
rate(kidp_broker_callbacks_total[5m])
```

#### GitOps Metrics
```promql
# Sync latency (Git commit → cluster apply)
flux_sync_duration_seconds

# Drift detection count
flux_drift_detected_total

# Failed reconciliations
flux_reconcile_condition{type="Ready", status="False"}
```

### Resource Metrics

#### Database Metrics
```promql
# Provisioning time
histogram_quantile(0.95, kidp_database_provisioning_duration_seconds)

# Connection pool utilization
kidp_database_connections_active / kidp_database_connections_max

# Query latency (from cloud provider)
kidp_database_query_latency_p95_milliseconds

# Storage utilization
kidp_database_storage_used_gb / kidp_database_storage_allocated_gb
```

#### Service Metrics
```promql
# Pod availability
sum(kube_pod_status_ready{namespace=~"team-.*"}) /
sum(kube_pod_status_phase{namespace=~"team-.*"})

# Request rate
rate(http_requests_total[5m])

# Error rate
rate(http_requests_total{status=~"5.."}[5m]) /
rate(http_requests_total[5m])

# Response time
histogram_quantile(0.95, http_request_duration_seconds)
```

### Cost Metrics
```promql
# Total platform spend
sum(kidp_resource_cost_monthly_usd)

# Cost by team
sum by (team) (kidp_resource_cost_monthly_usd)

# Cost by resource type
sum by (kind) (kidp_resource_cost_monthly_usd)

# Cost trend (week over week)
(sum(kidp_resource_cost_monthly_usd) -
 sum(kidp_resource_cost_monthly_usd offset 1w)) /
 sum(kidp_resource_cost_monthly_usd offset 1w) * 100
```

## Dashboards

### 1. Platform Overview Dashboard
**Purpose**: Executive summary of platform health

**Panels**:
- Total resources deployed
- Deployment success rate (24h)
- Active incidents
- Platform uptime
- Monthly cost trend
- Top resource types by count
- Active teams and applications

### 2. Operator Dashboard
**Purpose**: Monitor operator health and performance

**Panels**:
- Reconciliation rate per controller
- Error rate per controller
- Queue depth and backlog
- Reconciliation duration (p50, p95, p99)
- Recent errors (log table)
- Resource status distribution

### 3. Broker Dashboard
**Purpose**: Monitor deployment execution

**Panels**:
- Deployment requests per minute
- Success/failure rate
- API latency per cloud provider
- Active deployments by region
- Cloud API error rate
- Callback retry rate
- Deployment duration histogram

### 4. Team Dashboard
**Purpose**: Team-specific resource view

**Panels**:
- Team resource inventory
- Application health status
- Recent deployments
- Cost breakdown
- Resource utilization
- Quota consumption
- Recent events and alerts

### 5. Cost Dashboard
**Purpose**: Financial tracking and optimization

**Panels**:
- Total monthly spend
- Spend by team (pie chart)
- Spend by resource type
- Cost trend (6 months)
- Top 10 most expensive resources
- Budget vs. actual
- Cost optimization recommendations

### 6. Security Dashboard
**Purpose**: Security posture monitoring

**Panels**:
- Policy violations
- Resources without encryption
- Secrets rotation status
- Failed authentication attempts
- Network policy coverage
- Vulnerability scan results

## Alerting Rules

### Critical Alerts (PagerDuty)

#### Operator Down
```yaml
alert: OperatorDown
expr: up{job="kidp-operator"} == 0
for: 5m
severity: critical
annotations:
  summary: "Operator {{ $labels.controller }} is down"
  description: "No metrics from operator for 5 minutes"
```

#### Broker Unreachable
```yaml
alert: BrokerUnreachable
expr: up{job="kidp-broker"} == 0
for: 2m
severity: critical
annotations:
  summary: "Broker {{ $labels.region }} is unreachable"
```

#### High Error Rate
```yaml
alert: HighDeploymentErrorRate
expr: |
  rate(kidp_broker_deployments_total{status="failed"}[10m]) /
  rate(kidp_broker_deployments_total[10m]) > 0.1
for: 10m
severity: critical
annotations:
  summary: "Deployment error rate > 10% for {{ $labels.cloud_provider }}"
```

### Warning Alerts (Slack)

#### Slow Reconciliation
```yaml
alert: SlowReconciliation
expr: |
  histogram_quantile(0.95,
    kidp_operator_reconcile_duration_seconds) > 30
for: 15m
severity: warning
annotations:
  summary: "Slow reconciliation for {{ $labels.controller }}"
```

#### Queue Backlog
```yaml
alert: ReconciliationBacklog
expr: kidp_operator_queue_depth > 100
for: 10m
severity: warning
annotations:
  summary: "Reconciliation queue backlog for {{ $labels.controller }}"
```

#### Cost Budget Exceeded
```yaml
alert: TeamBudgetExceeded
expr: |
  sum by (team) (kidp_resource_cost_monthly_usd) >
  kidp_team_budget_monthly_usd
for: 1h
severity: warning
annotations:
  summary: "Team {{ $labels.team }} exceeded monthly budget"
```

### Info Alerts (Email)

#### New Resource Type Deployed
```yaml
alert: NewResourceTypeDeployed
expr: |
  count by (kind) (kidp_resource_count) unless
  count by (kind) (kidp_resource_count offset 1d)
severity: info
annotations:
  summary: "New resource type deployed: {{ $labels.kind }}"
```

## Logging Standards

### Structured Logging Format
```json
{
  "timestamp": "2024-10-01T10:30:00Z",
  "level": "info",
  "component": "database-operator",
  "controller": "database",
  "namespace": "team-backend",
  "name": "user-db",
  "reconcileID": "550e8400-e29b-41d4-a716-446655440000",
  "message": "Initiated deployment to broker",
  "deploymentID": "dep_abc123",
  "brokerURL": "https://broker-azure-westus2.company.com",
  "duration_ms": 45
}
```

### Log Levels
- **DEBUG**: Detailed execution flow (disabled in production)
- **INFO**: Normal operations (reconciliation, deployments)
- **WARN**: Recoverable errors (retries, quota warnings)
- **ERROR**: Failed operations requiring attention
- **FATAL**: Unrecoverable errors (operator crash)

### Log Queries (Loki)

#### Find Failed Deployments
```logql
{component="broker"} |= "deployment failed" | json
```

#### Trace Deployment by ID
```logql
{deploymentID="dep_abc123"} | json
```

#### Error Rate by Component
```logql
rate({level="error"}[5m]) by (component)
```

## Tracing

### Span Attributes
```json
{
  "trace_id": "550e8400-e29b-41d4-a716-446655440000",
  "span_id": "abc123def456",
  "parent_span_id": "789xyz012abc",
  "operation": "database.deploy",
  "resource.name": "user-db",
  "resource.namespace": "team-backend",
  "resource.kind": "Database",
  "broker.url": "https://broker-azure-westus2.company.com",
  "cloud.provider": "azure",
  "cloud.region": "westus2",
  "duration_ms": 45000,
  "status": "success"
}
```

### Trace Visualization
```
└── Deployment Request [45s]
    ├── Git Commit Detected [100ms]
    ├── FluxCD Reconciliation [2s]
    ├── Operator Reconciliation [43s]
    │   ├── Validation [200ms]
    │   ├── Broker API Call [42s]
    │   │   ├── Authentication [50ms]
    │   │   ├── Cloud API: Create Resource Group [5s]
    │   │   ├── Cloud API: Create Database [35s]
    │   │   └── Status Callback [500ms]
    │   └── Status Update [300ms]
    └── Resource Ready Event [100ms]
```

## Audit Logging

### Audit Events
- Resource creation/modification/deletion
- Policy changes
- RBAC changes
- Failed authentication attempts
- Cost threshold breaches
- Manual overrides (--force flag usage)

### Audit Log Format
```json
{
  "timestamp": "2024-10-01T10:30:00Z",
  "event": "resource.created",
  "actor": {
    "user": "jane.doe@company.com",
    "team": "backend-team",
    "method": "gitops"
  },
  "resource": {
    "kind": "Database",
    "namespace": "team-backend",
    "name": "user-db"
  },
  "changes": {
    "spec.engine": "postgresql",
    "spec.size": "small"
  },
  "ip": "10.0.5.100",
  "user_agent": "flux/v2.0"
}
```

### Compliance Reports
- Monthly resource change audit
- Policy violation summary
- Security incident report
- Cost variance report

## Performance Monitoring

### SLIs (Service Level Indicators)
```yaml
# Deployment success rate
SLI: successful_deployments / total_deployments
Target: > 98%

# API availability
SLI: successful_api_requests / total_api_requests
Target: > 99.9%

# Reconciliation latency
SLI: p95(reconciliation_duration)
Target: < 30 seconds

# Status update latency
SLI: p95(status_callback_duration)
Target: < 5 seconds
```

### Capacity Planning
```promql
# etcd size growth rate
rate(etcd_mvcc_db_total_size_in_bytes[7d])

# Operator memory usage trend
rate(container_memory_usage_bytes{pod=~".*-operator-.*"}[7d])

# Broker CPU usage trend
rate(container_cpu_usage_seconds_total{pod=~".*-broker-.*"}[7d])
```

## Integration

### Notification Channels
- **Slack**: Warnings and info alerts
- **PagerDuty**: Critical alerts (24/7 on-call)
- **Email**: Daily digest, compliance reports
- **Webhook**: Custom integrations (Jira, ServiceNow)

### External Monitoring
- **Status Page**: Public uptime status (uptimerobot, statuspage.io)
- **Cloud Provider Monitoring**: Integrate cloud-native metrics
- **APM Integration**: Connect application performance monitoring

## Success Metrics
- **MTTD** (Mean Time to Detect): < 2 minutes
- **MTTR** (Mean Time to Resolve): < 15 minutes
- **Alert fatigue**: < 5 false positives per week
- **Dashboard load time**: < 3 seconds
- **Log query performance**: < 5 seconds for 24h queries

## Future Enhancements
- AI-powered anomaly detection
- Predictive alerting (forecast issues before they occur)
- Auto-remediation for common issues
- Cost optimization recommendations
- Multi-cluster observability aggregation