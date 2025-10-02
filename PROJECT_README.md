# KIDP - Kubernetes Internal Developer Platform

A Kubernetes-native Internal Developer Platform (IDP) built with Go and the controller-runtime framework. KIDP provides self-service infrastructure provisioning for development teams through declarative Kubernetes CRDs.

## 🎯 Project Overview

KIDP implements a **management cluster + stateless broker** architecture where:
- **Management cluster**: Runs this operator and stores the desired state in CRDs
- **Stateless brokers**: Deployed per environment/cloud, receive webhooks to provision actual infrastructure
- **GitOps integration**: All state is stored in Git and synced via ArgoCD/Flux

This repository contains the management cluster operator that orchestrates resource provisioning through brokers.

## 🏗️ Architecture

```
┌─────────────────────────────────────┐
│     Management Cluster (KIDP)       │
│  ┌─────────────────────────────┐   │
│  │   Custom Resources (CRDs)    │   │
│  │  - Teams                     │   │
│  │  - Databases                 │   │
│  │  - Applications (planned)    │   │
│  │  - Services (planned)        │   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │     KIDP Operator            │   │
│  │  - Reconciliation loops      │   │
│  │  - Webhook calls to brokers  │   │
│  │  - Status updates            │   │
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
           │
           │ HTTPS Webhooks
           │
    ┌──────┴──────┐
    │             │
┌───▼────┐   ┌───▼────┐
│ Broker │   │ Broker │
│ (Azure)│   │ (AWS)  │
└────────┘   └────────┘
```

## 📦 Current Features

### Implemented CRDs

#### Team (Cluster-scoped)
Represents a development team with:
- Display name and contacts
- Cost center for budget tracking
- Budget limits and alert thresholds
- Resource quotas (max databases, services, etc.)
- Auto-calculated current spend and resource counts

#### Database (Namespaced)
Represents a managed database with:
- Owner reference (Team or Application)
- Engine type (PostgreSQL, MySQL, MongoDB, Redis, SQL Server)
- Version and size (small/medium/large/xlarge)
- High availability configuration
- Backup settings (schedule, retention, PITR)
- Encryption (at rest and in transit)
- Status tracking (Pending → Provisioning → Ready → Failed)

## 🚀 Quick Start

### Prerequisites

- Go 1.23+
- kubectl
- kind (for local development)

### Local Development

1. **Clone and initialize**:
```bash
git clone https://github.com/aykay76/kidp.git
cd kidp
go mod download
```

2. **Generate code and manifests**:
```bash
make generate  # Generate DeepCopy methods
make manifests # Generate CRD YAML files
```

3. **Build the operator**:
```bash
make build
```

4. **Create a local kind cluster**:
```bash
make kind-create
```

5. **Install the CRDs**:
```bash
kubectl apply -f config/crd/bases
```

6. **Run the operator locally**:
```bash
./bin/manager --leader-elect=false --metrics-bind-address=:9090
```

7. **Apply sample resources** (in another terminal):
```bash
kubectl apply -f config/samples/platform_v1_team.yaml
kubectl apply -f config/samples/platform_v1_database.yaml
```

8. **Check the resources**:
```bash
kubectl get teams,databases
kubectl get team platform-team -o yaml
kubectl get database postgres-app-db -o yaml
```

### Cleanup

```bash
make kind-delete
```

## 📁 Project Structure

```
.
├── api/v1/                      # CRD type definitions
│   ├── groupversion_info.go    # API group registration
│   ├── team_types.go            # Team CRD
│   ├── database_types.go        # Database CRD
│   └── zz_generated.deepcopy.go # Generated DeepCopy methods
├── cmd/manager/                 # Main operator entry point
│   └── main.go
├── internal/controller/         # Reconciliation controllers
│   ├── team_controller.go       # Team reconciler
│   └── database_controller.go   # Database reconciler
├── config/
│   ├── crd/bases/               # Generated CRD manifests
│   └── samples/                 # Example resource YAMLs
├── hack/                        # Build scripts and configs
│   ├── boilerplate.go.txt       # License header
│   └── kind-config.yaml         # Local cluster config
├── Makefile                     # Build automation
├── go.mod                       # Go dependencies
└── README.md                    # This file
```

## 🛠️ Development Workflow

### Make Targets

- `make help` - Display all available targets
- `make generate` - Generate DeepCopy methods for CRDs
- `make manifests` - Generate CRD YAML manifests
- `make build` - Compile the operator binary
- `make test` - Run unit tests
- `make fmt` - Format Go code
- `make vet` - Run Go vet
- `make lint` - Run golangci-lint
- `make kind-create` - Create local kind cluster
- `make kind-delete` - Delete local kind cluster
- `make deploy` - Deploy to kind cluster with kubectl

### Adding a New CRD

1. Create `api/v1/<resource>_types.go` with your CRD definition
2. Add kubebuilder markers for validation and printing
3. Run `make generate && make manifests`
4. Create controller in `internal/controller/<resource>_controller.go`
5. Register controller in `cmd/manager/main.go`
6. Create sample in `config/samples/`

## 📊 CRD Examples

### Team Resource

```yaml
apiVersion: platform.company.com/v1
kind: Team
metadata:
  name: platform-team
spec:
  displayName: "Platform Engineering"
  contacts:
    - name: "Alice Johnson"
      email: "alice@company.com"
      role: "Team Lead"
  costCenter: "ENG-001"
  budget:
    monthlyLimit: 50000.00
    alertThresholds: [0.7, 0.9]
  quotas:
    maxApplications: 10
    maxDatabases: 20
    maxServices: 50
    maxCaches: 10
```

### Database Resource

```yaml
apiVersion: platform.company.com/v1
kind: Database
metadata:
  name: postgres-app-db
  namespace: default
spec:
  owner:
    kind: Team
    name: platform-team
  engine: postgresql
  version: "15"
  size: medium
  highAvailability: true
  backup:
    enabled: true
    schedule: "0 2 * * *"
    retention: "30d"
  encryption:
    atRest:
      enabled: true
    inTransit:
      enabled: true
```

## 🔮 Roadmap

### Planned CRDs
- **Application**: Multi-resource container (databases, caches, services)
- **Service**: Container deployments with ingress
- **Cache**: Redis/Memcached instances
- **Topic**: Kafka topics and event streams

### Planned Features
- Broker webhook communication (HTTP client)
- Async status updates from brokers
- Cost tracking and budget alerts
- Resource cleanup with finalizers
- Validation webhooks
- Conversion webhooks for versioning
- Metrics and observability
- Multi-tenancy support

### Integration Plans
- GitOps (ArgoCD/Flux) for declarative sync
- Service catalog with templates
- Developer portal UI
- CLI tool for developers
- Terraform provider for hybrid workflows

## 🏛️ Design Principles

1. **Kubernetes-native**: Everything is a CRD, managed by kubectl
2. **Stateless brokers**: All state in management cluster, brokers are ephemeral
3. **Async operations**: Non-blocking provisioning with status callbacks
4. **GitOps first**: All resources stored in Git and synced
5. **Multi-cloud**: Cloud-agnostic operator, cloud-specific brokers
6. **Developer-friendly**: Self-service with guard rails and automation

## 📚 Related PRDs

This implementation is based on several Product Requirement Documents:

- **Management Cluster PRD**: Core architecture and operator design
- **Deployment Broker PRD**: Stateless broker pattern and webhook API
- **Developer Experience PRD**: Self-service workflows and abstractions
- **GitOps Integration PRD**: Git as source of truth
- **Service Catalog & Marketplace PRD**: Template system and reusable components
- **Security & Compliance PRD**: IAM, secrets, encryption, audit

## 📄 License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

## 👥 Contributing

This is currently a personal project by [@aykay76](https://github.com/aykay76). Contributions welcome!

## 🤝 Support

For questions or issues:
- Open an issue on GitHub
- Contact: [@aykay76](https://github.com/aykay76)

---

**Status**: 🚧 Early development - Core CRDs and operator implemented, broker communication next!
