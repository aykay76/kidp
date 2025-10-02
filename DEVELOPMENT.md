# KIDP Development Guide

## 🎉 What We've Built

We've successfully scaffolded a Kubernetes operator for the KIDP (Kubernetes Internal Developer Platform) project with:

✅ **Core Infrastructure**
- Go 1.23 project with controller-runtime v0.19.0
- Kubebuilder-style project structure
- Comprehensive Makefile for build automation
- kind cluster configuration for local development

✅ **Custom Resource Definitions (CRDs)**
- **Team CRD** (cluster-scoped): Team management with budgets, quotas, and contacts
- **Database CRD** (namespaced): Managed databases with backup, encryption, HA support

✅ **Controllers**
- Team reconciler with status tracking
- Database reconciler with placeholder provisioning logic

✅ **Local Testing**
- kind cluster with 3 nodes (1 control-plane, 2 workers)
- Sample manifests for Team and Database resources
- Verified end-to-end functionality

## 📊 Current State

### What Works
- ✅ CRD type definitions with kubebuilder markers
- ✅ DeepCopy code generation
- ✅ CRD manifest generation
- ✅ Operator compiles and runs
- ✅ Resources can be created/updated/deleted
- ✅ Controllers reconcile resources and update status
- ✅ Kind cluster integration

### What's Next (Immediate)
- 🔨 Implement broker webhook client
- 🔨 Add finalizers for cleanup
- 🔨 Create remaining CRDs (Application, Service, Cache, Topic)
- 🔨 Add validation webhooks
- 🔨 Implement cost tracking logic

### What's Next (Soon)
- 🔮 Deployment broker implementation (separate service)
- 🔮 ArgoCD/Flux integration
- 🔮 Developer CLI tool
- 🔮 Web UI dashboard
- 🔮 Metrics and observability

## 🧪 Testing Your Changes

### 1. After Modifying CRD Types

```bash
# Regenerate code and manifests
make generate
make manifests

# Rebuild operator
make build

# Update CRDs in cluster (if running)
kubectl apply -f config/crd/bases

# Restart operator to pick up changes
pkill manager && ./bin/manager --leader-elect=false --metrics-bind-address=:9090 > operator.log 2>&1 &
```

### 2. After Modifying Controllers

```bash
# Just rebuild and restart
make build
pkill manager && ./bin/manager --leader-elect=false --metrics-bind-address=:9090 > operator.log 2>&1 &
```

### 3. Testing with Sample Resources

```bash
# Apply samples
kubectl apply -f config/samples/

# Watch reconciliation
tail -f operator.log

# Check resource status
kubectl get teams,databases
kubectl describe team platform-team
kubectl describe database postgres-app-db

# Delete resources
kubectl delete -f config/samples/
```

## 🎯 Common Development Tasks

### Add a New Field to a CRD

1. Edit `api/v1/<resource>_types.go`
2. Add the field to the Spec or Status struct
3. Add kubebuilder markers if needed (validation, default, etc.)
4. Run `make generate && make manifests`
5. Update sample YAML in `config/samples/`
6. Test with `kubectl apply`

Example:
```go
// Add to DatabaseSpec
// RetentionDays for automated cleanup
// +kubebuilder:validation:Minimum=1
// +kubebuilder:validation:Maximum=365
// +optional
RetentionDays *int32 `json:"retentionDays,omitempty"`
```

### Add Validation Logic

Add to your controller's Reconcile method:

```go
// Validate the spec
if database.Spec.Engine == "postgresql" && database.Spec.Version < "13" {
    log.Error(nil, "PostgreSQL version too old", "version", database.Spec.Version)
    database.Status.Phase = "Failed"
    database.Status.Message = "PostgreSQL < 13 is not supported"
    r.Status().Update(ctx, database)
    return ctrl.Result{}, nil
}
```

### Call a Broker Webhook

Placeholder for future implementation:

```go
// TODO: Implement broker client
brokerClient := broker.NewClient(database.Spec.Target)
provisionReq := &broker.ProvisionDatabaseRequest{
    DatabaseID: string(database.UID),
    Engine: database.Spec.Engine,
    Version: database.Spec.Version,
    Size: database.Spec.Size,
    // ... more fields
}

resp, err := brokerClient.ProvisionDatabase(ctx, provisionReq)
if err != nil {
    log.Error(err, "Failed to call broker")
    return ctrl.Result{RequeueAfter: 30 * time.Second}, err
}

database.Status.Phase = "Provisioning"
database.Status.DeploymentID = resp.DeploymentID
r.Status().Update(ctx, database)
```

### Add a Finalizer

```go
import (
    "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const finalizerName = "platform.company.com/database-cleanup"

func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    database := &platformv1.Database{}
    if err := r.Get(ctx, req.NamespacedName, database); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Handle deletion
    if !database.DeletionTimestamp.IsZero() {
        if controllerutil.ContainsFinalizer(database, finalizerName) {
            // Call broker to deprovision
            // brokerClient.Deprovision(database.Status.DeploymentID)
            
            // Remove finalizer
            controllerutil.RemoveFinalizer(database, finalizerName)
            if err := r.Update(ctx, database); err != nil {
                return ctrl.Result{}, err
            }
        }
        return ctrl.Result{}, nil
    }

    // Add finalizer if not present
    if !controllerutil.ContainsFinalizer(database, finalizerName) {
        controllerutil.AddFinalizer(database, finalizerName)
        if err := r.Update(ctx, database); err != nil {
            return ctrl.Result{}, err
        }
    }

    // ... rest of reconciliation
}
```

## 🐛 Debugging Tips

### Check Operator Logs
```bash
tail -f operator.log
# or
kubectl logs -f deployment/kidp-controller-manager -n kidp-system
```

### Check CRD Schema
```bash
kubectl get crd databases.platform.company.com -o yaml
kubectl explain database.spec
```

### Check Resource Events
```bash
kubectl describe database postgres-app-db
```

### Enable Debug Logging
```bash
./bin/manager --leader-elect=false --metrics-bind-address=:9090 --zap-log-level=debug
```

### Check RBAC Permissions
```bash
kubectl get clusterrole manager-role -o yaml
kubectl auth can-i --as=system:serviceaccount:kidp-system:controller-manager create databases
```

## 📂 Project Layout Conventions

```
api/v1/              - CRD type definitions (API contracts)
cmd/manager/         - Main entry point for operator
internal/controller/ - Reconciliation logic (business logic)
config/crd/bases/    - Generated CRD YAML (don't edit manually)
config/samples/      - Example resources for testing
config/rbac/         - Generated RBAC rules (from kubebuilder markers)
hack/                - Build scripts and dev configs
```

## 🎨 Coding Style

- Use kubebuilder markers for CRD validation
- Keep controllers focused on reconciliation logic
- Extract complex logic into helper functions
- Use structured logging: `log.Info("message", "key", value)`
- Return `ctrl.Result{RequeueAfter: duration}` for async operations
- Return `ctrl.Result{}` for successful one-time reconciliation

## 🚀 Deployment Options

### Local Development (Current)
```bash
kind cluster + operator running on host
```

### In-Cluster Development
```bash
make docker-build
make docker-push
make deploy
```

### Production (Future)
```bash
Helm chart + ArgoCD GitOps
```

## 📊 Key Metrics to Watch

Once we add metrics:
- `kidp_reconciliations_total` - Total reconciliation count
- `kidp_reconciliation_duration_seconds` - Reconciliation latency
- `kidp_resources_total` - Count of managed resources by type
- `kidp_broker_calls_total` - Broker webhook calls
- `kidp_broker_call_duration_seconds` - Broker latency

## 🤝 Next Steps

1. **Broker Client** - Implement HTTP client for webhook calls
2. **Application CRD** - Container grouping with multiple resources
3. **Finalizers** - Proper cleanup on deletion
4. **Validation Webhooks** - Admission control
5. **Cost Tracking** - Budget enforcement logic
6. **Broker Service** - Separate deployment broker implementation

## 📚 Useful Resources

- [Kubebuilder Book](https://book.kubebuilder.io/)
- [controller-runtime Godoc](https://pkg.go.dev/sigs.k8s.io/controller-runtime)
- [Kubernetes API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
- [Operator Best Practices](https://sdk.operatorframework.io/docs/best-practices/)

---

**Happy coding! 🎉**
