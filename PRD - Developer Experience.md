# PRD: Developer Experience

## Overview
The developer experience layer provides multiple interfaces for teams to interact with KIDP, prioritizing GitOps workflows while supporting CLI, portal, and natural language interfaces.

## Objectives
- Make infrastructure provisioning self-service and intuitive
- Minimize cognitive load with sensible defaults
- Provide fast feedback loops for debugging
- Enable both declarative (GitOps) and imperative (CLI) workflows
- Reduce time-to-first-deployment for new services

## Interface Hierarchy

### 1. GitOps (Primary - Production)
**Target Users**: All teams for production changes

**Workflow**:
```bash
# Edit YAML manifest
vim manifests/databases/user-db.yaml

# Commit and push
git add manifests/databases/user-db.yaml
git commit -m "Add user service database"
git push origin main

# FluxCD automatically syncs (1-2 min)
# Check status
kubectl get database user-db -n team-backend
```

**Benefits**:
- Full audit trail via Git history
- Peer review via pull requests
- Rollback via Git revert
- Declarative, version-controlled

### 2. CLI (Secondary - Development/Testing)
**Target Users**: Developers for rapid iteration

**Installation**:
```bash
# Homebrew (Mac)
brew install kidp-cli

# Or direct download
curl -sL https://cli.kidp.io/install.sh | bash
```

**Common Commands**:
```bash
# Generate manifest from template
kidp create database \
  --name user-db \
  --type postgres \
  --size small \
  --owner team-backend \
  --output manifests/databases/user-db.yaml

# Apply directly (dev only)
kidp apply -f manifests/databases/user-db.yaml

# Check status
kidp status database user-db

# Get connection info
kidp get connection user-db

# Delete resource
kidp delete database user-db

# Validate manifest before commit
kidp validate -f manifests/databases/user-db.yaml
```

**Features**:
- Manifest generation with validation
- Interactive prompts for missing fields
- Dry-run mode for testing
- Context-aware completions (bash/zsh)

### 3. Self-Service Portal (Read-Only)
**Target Users**: Non-technical stakeholders, status monitoring

**Purpose**: Observability, not mutation

**Features**:
- **Resource catalog**: Browse all deployable resources
- **Team dashboard**: View team's infrastructure
- **Application topology**: Visualize service dependencies
- **Status monitoring**: Real-time deployment progress
- **Cost tracking**: View spend by team/application
- **Documentation**: Quick-start guides and examples

**Key Principle**: Portal never writes state, only observes

### 4. Natural Language Agent (Future)
**Target Users**: Developers seeking guidance

**Capabilities**:
```
User: "I need a postgres database for my user service"

Agent: I'll help you create a PostgreSQL database. Here's what I suggest:

---
apiVersion: platform.company.com/v1
kind: Database
metadata:
  name: user-service-db
  namespace: team-backend
spec:
  owner:
    kind: Team
    name: backend-team
  engine: postgresql
  version: "15"
  size: small
  target: azure-westus2-prod
  backupRetention: 7d
---

This will create a small PostgreSQL 15 instance in Azure West US 2.
Estimated monthly cost: $150 USD

Would you like me to:
1. Save this to manifests/databases/user-service-db.yaml
2. Explain any of these settings
3. Suggest related resources (cache, secrets, etc.)
```

**Agent vs. Direct Action**:
- Agent generates manifests, user commits to Git
- No direct mutation of cluster state
- Maintains GitOps as source of truth

## Manifest Templates (Service Archetypes)

### Template Library
```bash
# List available templates
kidp templates list

# Inspect template
kidp templates show web-service

# Create from template
kidp create from-template web-service \
  --name user-api \
  --team backend-team \
  --output manifests/
```

### Common Archetypes

#### 1. Web Service
```yaml
# Includes: Service, Database, Cache, LoadBalancer
apiVersion: platform.company.com/v1
kind: Application
metadata:
  name: user-api
spec:
  owner:
    kind: Team
    name: backend-team
  components:
    - kind: Service
      name: api
      spec:
        image: user-api:latest
        replicas: 3
        resources:
          cpu: "500m"
          memory: "1Gi"
    - kind: Database
      name: userdb
      spec:
        engine: postgresql
        size: small
    - kind: Cache
      name: session-cache
      spec:
        engine: redis
        size: small
```

#### 2. Background Worker
```yaml
# Includes: Service, Topic, Database
apiVersion: platform.company.com/v1
kind: Application
metadata:
  name: email-worker
spec:
  components:
    - kind: Service
      name: worker
      spec:
        image: email-worker:latest
        replicas: 2
    - kind: Topic
      name: email-queue
      spec:
        partitions: 6
    - kind: Database
      name: email-log
      spec:
        engine: postgresql
        size: small
```

#### 3. Static Website
```yaml
# Includes: Service (nginx), LoadBalancer
apiVersion: platform.company.com/v1
kind: Application
metadata:
  name: marketing-site
spec:
  components:
    - kind: Service
      name: frontend
      spec:
        image: nginx:alpine
        replicas: 2
        staticContent:
          source: s3://marketing-assets/
```

## Dependency Discovery

### Automatic Suggestions
```bash
$ kidp create service api --name user-api

Analyzing service requirements...

Detected database queries in code - would you like to add a database?
  → postgres (recommended based on schema complexity)
  → mysql

Detected Redis client - would you like to add a cache?
  → redis (6GB recommended for session storage)

Detected Kafka producer - would you like to add topics?
  → user-events topic (6 partitions recommended)

Generate manifests with these dependencies? [Y/n]
```

### Relationship Visualization
```bash
$ kidp graph application user-service

user-service (Application)
├── user-api (Service)
│   ├── → user-db (Database) [owner]
│   ├── → session-cache (Cache) [dependency]
│   └── → user-events (Topic) [produces]
├── user-db (Database)
│   └── → backup-bucket (Storage) [backup]
└── session-cache (Cache)

Legend:
  [owner] - Explicit ownership
  [dependency] - Runtime dependency
  [produces/consumes] - Event relationship
```

## Validation & Feedback

### Pre-Commit Validation
```yaml
# .git/hooks/pre-commit
#!/bin/bash
kidp validate --all manifests/

# Check for:
# - YAML syntax errors
# - Missing required fields
# - Policy violations
# - Cost estimates exceeding budget
```

### Policy Enforcement
```bash
$ kidp apply -f database.yaml

✗ Policy violation: DatabaseSizePolicy
  
  Rule: Databases in production must be size 'medium' or larger
  Current: small
  Suggested: medium
  
  Override with: --force (requires admin approval)
```

### Cost Estimation
```bash
$ kidp estimate -f manifests/

Estimated monthly costs:
┌─────────────────────┬──────────┬──────────┐
│ Resource            │ Type     │ Cost     │
├─────────────────────┼──────────┼──────────┤
│ user-db             │ Database │ $150     │
│ session-cache       │ Cache    │ $80      │
│ user-api (3 pods)   │ Compute  │ $120     │
│ lb-user-api         │ Network  │ $25      │
└─────────────────────┴──────────┴──────────┘
Total: $375/month

Team 'backend-team' budget remaining: $625/month
```

## Status & Debugging

### Real-Time Status
```bash
$ kidp status database user-db

Name:         user-db
Namespace:    team-backend
Status:       Deploying
Phase:        provisioning_storage
Progress:     60%
Started:      2024-10-01 10:15:00 UTC (5m ago)
ETA:          ~3 minutes

Conditions:
  ✓ Validated    2024-10-01 10:15:05 UTC
  ✓ QuotaChecked 2024-10-01 10:15:10 UTC
  ⟳ Deploying    2024-10-01 10:15:15 UTC

Recent Events:
  10:15:15  Normal   DeploymentStarted  Initiated deployment to azure-westus2-prod
  10:17:30  Normal   StorageProvisioned Storage account created successfully
  10:19:45  Warning  NetworkDelay       Network configuration taking longer than expected
```

### Logs & Events
```bash
# Stream deployment logs
kidp logs database user-db --follow

# Get deployment events
kidp events database user-db

# Operator logs for troubleshooting
kidp logs operator database-operator
```

### Troubleshooting Guide
```bash
$ kidp troubleshoot database user-db

Running diagnostics...

✓ Resource exists in management cluster
✓ Deployment request sent to broker
✗ Broker response timeout (expected < 30s, actual 45s)

Possible causes:
  1. Broker connectivity issues
     → Check: kubectl logs -n kidp-system deployment/broker-azure
  
  2. Cloud API rate limiting
     → Check: kidp status broker azure-westus2-prod
  
  3. Insufficient quota
     → Run: kidp quota check --region azure-westus2-prod

Suggested next steps:
  - View broker logs: kidp logs broker azure-westus2-prod
  - Contact platform team if issue persists
```

## Onboarding & Documentation

### Quick Start Guide
```bash
# New team setup
kidp init team backend-team

Created:
  ✓ Team CRD: backend-team
  ✓ Namespace: team-backend
  ✓ Git repository: github.com/company/team-backend-infra
  ✓ RBAC roles and bindings
  ✓ Resource quotas
  ✓ Sample manifests

Next steps:
  1. Clone your team repo:
     git clone git@github.com:company/team-backend-infra.git
  
  2. Create your first application:
     cd team-backend-infra
     kidp create from-template web-service --name my-app
  
  3. Commit and push:
     git add .
     git commit -m "Initial application"
     git push
  
  4. Monitor deployment:
     kidp status application my-app
```

### Interactive Tutorials
```bash
# Launch interactive tutorial
kidp learn

Available tutorials:
  1. Deploy your first application
  2. Add a database to your service
  3. Configure auto-scaling
  4. Set up monitoring and alerts
  5. Implement blue/green deployments

Select tutorial [1-5]:
```

### Contextual Help
```bash
# Get help on any resource type
kidp explain database

# Show examples
kidp examples database

# Field-level documentation
kidp explain database.spec.backupRetention
```

## Collaboration Features

### Team Views
```bash
# View all team resources
kidp get all --team backend-team

# Team topology map
kidp topology team backend-team
```

### Shared Templates
```bash
# Publish template for team use
kidp template publish ./my-template.yaml \
  --name custom-worker \
  --team backend-team

# Use shared template
kidp create from-template custom-worker --name my-worker
```

### Cost Allocation
```bash
# Team cost dashboard
kidp cost team backend-team --month october

# Cost breakdown by application
kidp cost application user-service --detailed
```

## Observability Integration

### Metrics Dashboard
```bash
# Open Grafana dashboard for resource
kidp dashboard database user-db

# Quick metrics in terminal
kidp metrics database user-db
  CPU: 45%
  Memory: 2.1 GB / 4 GB
  Connections: 23 / 100
  Query latency (p95): 12ms
```

### Alert Integration
```bash
# View active alerts
kidp alerts list --team backend-team

# Acknowledge alert
kidp alerts ack alert-user-db-cpu-high
```

## Success Metrics
- **Time to first deployment**: < 15 minutes
- **Manifest generation accuracy**: > 95%
- **CLI command success rate**: > 98%
- **Developer satisfaction**: > 8/10 (quarterly survey)
- **Self-service adoption**: > 80% of deployments via GitOps

## Future Enhancements
- IDE extensions (VSCode, IntelliJ)
- Slack bot for status queries
- Mobile app for monitoring
- Advanced cost optimization recommendations
- Automated dependency upgrades