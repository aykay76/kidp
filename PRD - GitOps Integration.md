# PRD: GitOps Integration

## Overview
GitOps integration enables version-controlled, declarative infrastructure management by syncing Git repository state to the management cluster using FluxCD.

## Objectives
- Provide Git as the single source of truth for all infrastructure
- Enable automated reconciliation of desired state
- Support multiple environments and teams with isolated repos
- Maintain full audit trail of all infrastructure changes

## Architecture

### FluxCD Components

#### 1. Source Controller
**Purpose**: Monitor Git repositories for changes

**Configuration**:
```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: kidp-infrastructure
  namespace: flux-system
spec:
  interval: 1m
  url: https://github.com/company/kidp-infrastructure
  ref:
    branch: main
  secretRef:
    name: github-deploy-key
```

**Requirements**:
- Support GitHub, GitLab, Bitbucket
- SSH and HTTPS authentication
- Branch and tag tracking
- Webhook integration for instant sync

#### 2. Kustomize Controller
**Purpose**: Apply manifests with overlays and patches

**Configuration**:
```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: infrastructure
  namespace: flux-system
spec:
  interval: 5m
  path: ./infrastructure
  prune: true
  sourceRef:
    kind: GitRepository
    name: kidp-infrastructure
  validation: client
  healthChecks:
    - apiVersion: apps/v1
      kind: Deployment
      name: kidp-operator
      namespace: kidp-system
```

**Features**:
- Environment-specific overlays
- Secret management with SOPS
- Resource pruning for deleted manifests
- Health checks for applied resources

#### 3. Helm Controller
**Purpose**: Manage Helm releases from Git

**Configuration**:
```yaml
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: kidp-operators
  namespace: kidp-system
spec:
  interval: 10m
  chart:
    spec:
      chart: ./charts/operators
      sourceRef:
        kind: GitRepository
        name: kidp-infrastructure
  values:
    replicaCount: 3
    image:
      tag: v1.0.0
```

## Repository Structure

### Recommended Layout
```
kidp-infrastructure/
├── infrastructure/          # Platform infrastructure
│   ├── base/               # Base configurations
│   │   ├── crds/
│   │   ├── operators/
│   │   └── brokers/
│   ├── overlays/
│   │   ├── production/
│   │   ├── staging/
│   │   └── development/
│   └── kustomization.yaml
├── teams/                  # Team-specific resources
│   ├── team-backend/
│   │   ├── applications/
│   │   ├── databases/
│   │   └── kustomization.yaml
│   └── team-frontend/
│       ├── applications/
│       └── kustomization.yaml
├── policies/               # Platform policies
│   ├── resource-quotas.yaml
│   ├── network-policies.yaml
│   └── pod-security.yaml
└── charts/                 # Helm charts
    └── operators/
        ├── Chart.yaml
        ├── values.yaml
        └── templates/
```

### Multi-Tenant Model

#### Option 1: Monorepo
**Single repository with team directories**

Pros:
- Centralized visibility
- Easier cross-team collaboration
- Simplified access control

Cons:
- All teams see all configs
- Larger blast radius for errors
- Git operations slower with growth

#### Option 2: Multi-Repo (Recommended)
**One repo per team + shared infrastructure repo**

```yaml
# Infrastructure repo
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: kidp-infrastructure
---
# Team repos
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: team-backend-infra
spec:
  url: https://github.com/company/team-backend-infra
```

Pros:
- Team isolation
- Independent deployment cadence
- Scoped access control

Cons:
- More repos to manage
- Requires coordination for shared resources

## Resource Ownership & RBAC

### GitRepository Access Control
```yaml
# Only platform admins can modify infrastructure repo
apiVersion: v1
kind: Secret
metadata:
  name: infra-repo-key
  namespace: flux-system
type: Opaque
data:
  identity: <ssh-private-key>
  known_hosts: <github-known-hosts>
---
# Teams get read access to their own repos
apiVersion: v1
kind: Secret
metadata:
  name: team-backend-repo-key
  namespace: team-backend
```

### Kustomization RBAC
```yaml
# Platform admins deploy to flux-system
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: flux-admin
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: kustomize-controller
  namespace: flux-system
---
# Team controllers limited to team namespace
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: flux-team-backend
  namespace: team-backend
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: team-resource-manager
subjects:
- kind: ServiceAccount
  name: kustomize-controller
  namespace: team-backend
```

## Drift Detection & Remediation

### Automatic Reconciliation
```yaml
spec:
  interval: 5m  # Check for drift every 5 minutes
  prune: true   # Delete resources removed from Git
  force: false  # Don't override manual changes (alert instead)
```

### Manual Override Protection
- Critical resources marked with annotation: `flux.weave.works/ignore: "true"`
- Manual changes trigger alerts
- Reconciliation disabled until investigation complete

### Drift Alerts
```yaml
# Prometheus alert
- alert: FluxDriftDetected
  expr: |
    gotk_reconcile_condition{type="Ready",status="False"} == 1
  for: 10m
  annotations:
    summary: "Flux detected drift in {{ $labels.name }}"
```

## Secret Management

### SOPS Integration
```yaml
# .sops.yaml - Encryption rules
creation_rules:
  - path_regex: .*.yaml
    encrypted_regex: ^(data|stringData)$
    age: age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

**Workflow**:
1. Encrypt secrets locally: `sops -e secret.yaml > secret.enc.yaml`
2. Commit encrypted file to Git
3. FluxCD decrypts on apply using SOPS key in cluster
4. Never commit plaintext secrets

### External Secrets Operator
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: database-credentials
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: azure-keyvault
    kind: SecretStore
  target:
    name: db-credentials
  data:
  - secretKey: password
    remoteRef:
      key: db-user-service-password
```

## Validation & Testing

### Pre-Commit Validation
```bash
#!/bin/bash
# .git/hooks/pre-commit

# Validate YAML syntax
yamllint -c .yamllint.yaml .

# Validate Kubernetes manifests
kubectl apply --dry-run=client -f .

# Check for secrets in plaintext
detect-secrets scan

# Validate SOPS encryption
sops verify encrypted-secrets/
```

### Pull Request Checks
```yaml
# GitHub Actions
name: Validate Infrastructure
on: [pull_request]
jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Setup Flux CLI
        uses: fluxcd/flux2/action@main
      - name: Validate Flux manifests
        run: flux check --pre
      - name: Run kustomize build
        run: kustomize build infrastructure/overlays/production
      - name: Policy validation
        uses: open-policy-agent/conftest-action@v0
        with:
          files: infrastructure/
```

### Staging Environment
- All changes tested in staging before production
- Automated smoke tests after deployment
- Rollback mechanism if tests fail

## Promotion Strategy

### Environment Progression
```
development → staging → production
```

### Automated Promotion
```yaml
# Auto-promote if staging healthy for 24h
apiVersion: image.toolkit.fluxcd.io/v1beta1
kind: ImageUpdateAutomation
metadata:
  name: promote-to-production
spec:
  interval: 1h
  sourceRef:
    kind: GitRepository
    name: kidp-infrastructure
  git:
    checkout:
      ref:
        branch: main
    commit:
      author:
        email: fluxcd@company.com
        name: FluxCD Automation
    push:
      branch: production
  update:
    path: ./infrastructure/overlays/production
    strategy: Setters
```

### Manual Promotion
```bash
# Create promotion PR
git checkout -b promote-v1.2.3
sed -i 's/tag: v1.2.2/tag: v1.2.3/' infrastructure/overlays/production/kustomization.yaml
git commit -m "Promote v1.2.3 to production"
gh pr create --title "Promote v1.2.3 to production"
```

## Rollback Procedures

### Automatic Rollback
```yaml
spec:
  rollback:
    enable: true
    force: true
    timeout: 5m
    disableWait: false
```

### Manual Rollback
```bash
# Revert to previous Git commit
git revert HEAD
git push

# Or use Flux CLI
flux reconcile kustomization infrastructure --with-source
```

### Emergency Rollback
```bash
# Suspend GitOps, manual intervention
flux suspend kustomization infrastructure
kubectl apply -f backup/previous-state.yaml
```

## Observability

### Flux Dashboard
```yaml
# Deploy Weave GitOps UI
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: weave-gitops
spec:
  chart:
    spec:
      chart: weave-gitops
      sourceRef:
        kind: HelmRepository
        name: weaveworks
```

### Prometheus Metrics
```
gotk_reconcile_duration_seconds
gotk_reconcile_condition{type="Ready"}
gotk_suspend_status
```

### Notifications
```yaml
apiVersion: notification.toolkit.fluxcd.io/v1beta1
kind: Alert
metadata:
  name: slack-notifications
spec:
  eventSeverity: info
  eventSources:
    - kind: GitRepository
      name: '*'
    - kind: Kustomization
      name: '*'
  providerRef:
    name: slack
```

## Disaster Recovery

### Backup Strategy
- Git is the backup (full history)
- etcd snapshots for cluster state
- Automated daily snapshots to S3/Azure Blob

### Recovery Procedure
1. Restore etcd from snapshot
2. Re-bootstrap FluxCD
3. Reconcile from Git (automatic)
4. Validate all resources healthy

### RTO/RPO Targets
- **RTO**: < 1 hour (cluster rebuild + GitOps sync)
- **RPO**: 0 (Git is source of truth)

## Success Metrics
- **Sync latency**: < 2 minutes from Git push to cluster apply
- **Drift detection time**: < 5 minutes
- **Failed reconciliation rate**: < 1%
- **Mean time to rollback**: < 5 minutes

## Future Enhancements
- Progressive delivery with Flagger
- Multi-cluster GitOps (fleet management)
- Automated dependency updates (Renovate/Dependabot)
- Cost estimation in PR comments