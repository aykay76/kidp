# PRD: Service Catalog & Marketplace

## Overview
The Service Catalog & Marketplace provides a curated, searchable library of pre-configured infrastructure patterns, templates, and integrations that accelerate development and enforce best practices. Think of it as the "npm registry" or "Helm Hub" for your internal platform.

## Objectives
- Reduce time-to-first-deployment from hours to minutes
- Standardize common patterns across teams
- Enable knowledge sharing and reusability
- Provide discovery mechanism for available resources
- Enforce governance through approved patterns
- Measure and optimize template adoption

## Architecture

### Catalog Structure
```
Service Catalog
├── Resource Templates        # Individual resource definitions
│   ├── Databases
│   ├── Caches
│   ├── Topics/Queues
│   ├── Storage
│   └── Networking
├── Application Archetypes    # Multi-resource patterns
│   ├── Web Services
│   ├── Background Workers
│   ├── Data Pipelines
│   ├── Static Sites
│   └── ML Services
├── Reference Implementations # Full working examples
│   ├── Microservices Demo
│   ├── Event-Driven Architecture
│   └── Data Lake Template
├── Integration Connectors    # Third-party services
│   ├── Observability (Datadog, New Relic)
│   ├── Data Stores (Snowflake, MongoDB Atlas)
│   ├── Communication (Twilio, SendGrid)
│   └── Security (Vault, 1Password)
└── Policy Packs              # Governance bundles
    ├── PCI Compliance
    ├── HIPAA Compliance
    └── Cost Optimization
```

## Catalog Metadata

### Template Descriptor
```yaml
apiVersion: catalog.platform.company.com/v1
kind: CatalogTemplate
metadata:
  name: web-service-standard
  namespace: platform-catalog
  labels:
    catalog.platform.company.com/category: application-archetype
    catalog.platform.company.com/maturity: stable
    catalog.platform.company.com/compliance: pci-dss,sox
spec:
  displayName: "Standard Web Service"
  description: "Production-ready web service with database, cache, and observability"
  icon: "https://icons.company.com/web-service.svg"
  version: "2.3.0"
  
  maintainers:
    - name: "Platform Team"
      email: "platform@company.com"
      slack: "#platform-support"
  
  tags:
    - web
    - api
    - rest
    - production-ready
  
  maturity: stable  # experimental | alpha | beta | stable | deprecated
  
  resources:
    - kind: Database
      required: true
      description: "Primary application database"
    - kind: Cache
      required: false
      description: "Optional Redis cache for sessions"
    - kind: Service
      required: true
      description: "Container-based web service"
    - kind: LoadBalancer
      required: true
      description: "HTTPS load balancer with WAF"
  
  parameters:
    - name: serviceName
      type: string
      required: true
      description: "Name of your service"
      validation: "^[a-z][a-z0-9-]*$"
    
    - name: team
      type: string
      required: true
      description: "Owning team"
      enum: ["backend-team", "frontend-team", "data-team"]
    
    - name: databaseEngine
      type: string
      required: false
      default: "postgresql"
      description: "Database engine"
      enum: ["postgresql", "mysql", "mongodb"]
    
    - name: databaseSize
      type: string
      required: false
      default: "medium"
      description: "Database instance size"
      enum: ["small", "medium", "large"]
      costImpact:
        small: 150
        medium: 400
        large: 1200
    
    - name: enableCache
      type: boolean
      required: false
      default: false
      description: "Enable Redis cache"
      costImpact: 80
    
    - name: replicas
      type: integer
      required: false
      default: 3
      minimum: 2
      maximum: 10
      description: "Number of service replicas"
  
  estimatedCost:
    monthly:
      minimum: 500
      typical: 750
      maximum: 2000
    currency: "USD"
  
  deploymentTime:
    estimated: "15-20 minutes"
  
  compliance:
    - pci-dss: "Encrypts data at rest and in transit"
    - sox: "Provides audit logging and access controls"
  
  documentation:
    readme: "https://docs.company.com/catalog/web-service"
    architecture: "https://docs.company.com/catalog/web-service/architecture"
    examples: "https://github.com/company/examples/web-service"
  
  support:
    tier: "platform"  # platform | community | experimental
    sla: "24x7 on-call support"
    
  metrics:
    deployments: 847
    successRate: 98.5
    averageRating: 4.7
    lastUpdated: "2024-09-15T10:00:00Z"
```

## Discovery & Search

### Catalog Browser (CLI)
```bash
# List all templates
kidp catalog list

# Filter by category
kidp catalog list --category application-archetype

# Search by keyword
kidp catalog search "web api"

# Filter by maturity
kidp catalog list --maturity stable,beta

# Show template details
kidp catalog show web-service-standard

# Show cost estimate for configuration
kidp catalog estimate web-service-standard \
  --param databaseSize=large \
  --param enableCache=true
```

### Catalog Browser (Web UI)

#### Homepage View
```
┌─────────────────────────────────────────────────┐
│  🔍 Search Catalog                              │
│  [Search templates, resources, examples...    ] │
└─────────────────────────────────────────────────┘

Popular Templates
┌──────────────┬──────────────┬──────────────┐
│ Web Service  │ Worker       │ Data Pipeline│
│ ⭐ 4.7 (847) │ ⭐ 4.5 (234) │ ⭐ 4.8 (156) │
│ $500-2k/mo   │ $200-800/mo  │ $1k-5k/mo    │
└──────────────┴──────────────┴──────────────┘

Browse by Category
├── 📦 Application Archetypes (12)
├── 🗄️  Databases (8)
├── 💾 Caches (4)
├── 📨 Messaging (6)
├── 🔌 Integrations (23)
└── 📋 Policy Packs (7)

Recently Added
• MongoDB Atlas Connector (3 days ago)
• ML Training Pipeline (1 week ago)
```

#### Template Detail View
```
Web Service (Standard)                    ⭐ 4.7 ★
v2.3.0 | Stable | Updated 2 weeks ago

A production-ready web service pattern with database,
optional cache, load balancing, and full observability.

📊 Usage: 847 deployments | 98.5% success rate

💰 Cost Estimate
├── Minimum: $500/month
├── Typical:  $750/month
└── Maximum:  $2,000/month

📦 Includes
├── ✓ PostgreSQL Database (configurable)
├── ✓ Redis Cache (optional)
├── ✓ Container Service (2-10 replicas)
├── ✓ HTTPS Load Balancer
├── ✓ SSL Certificate (auto-renew)
└── ✓ Grafana Dashboard

🔒 Compliance
├── ✓ PCI-DSS ready
├── ✓ SOX compliant
└── ✓ Encryption at rest/transit

⚙️  Configure & Deploy
[Customize Parameters →]

📖 Documentation | 💬 Slack Support | 🐛 Report Issue
```

### Search & Filtering

#### Search Capabilities
```bash
# Full-text search
kidp catalog search "postgresql redis"

# Tag-based search
kidp catalog search --tag web --tag production-ready

# Compliance filter
kidp catalog search --compliance pci-dss

# Cost range filter
kidp catalog search --max-cost 1000

# Maturity filter
kidp catalog search --maturity stable

# Team-specific (show team's previously used templates)
kidp catalog list --team backend-team --sort usage
```

#### Intelligent Recommendations
```bash
$ kidp catalog recommend --for web-api

Based on your request for 'web-api':

Recommended: Web Service (Standard) ⭐ 4.7
├── Best match for your use case
├── Used by 12 teams at company
├── $750/month typical cost
└── Deploy: kidp create from-template web-service-standard

Similar templates:
├── Web Service (Minimal) - $400/month
├── API Gateway Pattern - $600/month
└── GraphQL API - $850/month

Teams similar to yours also used:
├── Background Worker (often paired with web services)
└── Cache Layer (boosts API performance)
```

## Template Creation & Publishing

### Template Authoring

#### Simple Template (Single Resource)
```yaml
# templates/databases/postgres-small.yaml
apiVersion: catalog.platform.company.com/v1
kind: CatalogTemplate
metadata:
  name: postgres-small
spec:
  displayName: "PostgreSQL (Small)"
  description: "Small PostgreSQL database for development/testing"
  
  parameters:
    - name: name
      type: string
      required: true
    - name: team
      type: string
      required: true
  
  template: |
    apiVersion: platform.company.com/v1
    kind: Database
    metadata:
      name: {{ .name }}
      namespace: team-{{ .team }}
    spec:
      owner:
        kind: Team
        name: {{ .team }}
      engine: postgresql
      version: "15"
      size: small
      backup:
        enabled: true
        retention: 7d
```

#### Complex Template (Multi-Resource)
```yaml
# templates/archetypes/web-service.yaml
apiVersion: catalog.platform.company.com/v1
kind: CatalogTemplate
metadata:
  name: web-service-standard
spec:
  displayName: "Web Service (Standard)"
  
  parameters:
    - name: serviceName
      type: string
      required: true
    - name: team
      type: string
      required: true
    - name: databaseEngine
      type: string
      default: "postgresql"
      enum: ["postgresql", "mysql"]
    - name: enableCache
      type: boolean
      default: false
  
  template: |
    ---
    # Application wrapper
    apiVersion: platform.company.com/v1
    kind: Application
    metadata:
      name: {{ .serviceName }}
      namespace: team-{{ .team }}
    spec:
      owner:
        kind: Team
        name: {{ .team }}
    ---
    # Database
    apiVersion: platform.company.com/v1
    kind: Database
    metadata:
      name: {{ .serviceName }}-db
      namespace: team-{{ .team }}
      labels:
        app: {{ .serviceName }}
    spec:
      owner:
        kind: Application
        name: {{ .serviceName }}
      engine: {{ .databaseEngine }}
      version: "15"
      size: medium
      backup:
        enabled: true
        retention: 30d
    ---
    {{ if .enableCache }}
    # Redis Cache (conditional)
    apiVersion: platform.company.com/v1
    kind: Cache
    metadata:
      name: {{ .serviceName }}-cache
      namespace: team-{{ .team }}
      labels:
        app: {{ .serviceName }}
    spec:
      owner:
        kind: Application
        name: {{ .serviceName }}
      engine: redis
      version: "7"
      size: small
    {{ end }}
    ---
    # Service placeholder (user provides image later)
    apiVersion: platform.company.com/v1
    kind: Service
    metadata:
      name: {{ .serviceName }}
      namespace: team-{{ .team }}
      labels:
        app: {{ .serviceName }}
    spec:
      owner:
        kind: Application
        name: {{ .serviceName }}
      image: registry.company.com/{{ .team }}/{{ .serviceName }}:latest
      replicas: 3
      resources:
        requests:
          cpu: "500m"
          memory: "1Gi"
        limits:
          cpu: "2000m"
          memory: "4Gi"
      dependencies:
        databaseSelector:
          matchLabels:
            app: {{ .serviceName }}
        {{- if .enableCache }}
        cacheSelector:
          matchLabels:
            app: {{ .serviceName }}
        {{- end }}
```

### Template Publishing Workflow

#### 1. Development Phase
```bash
# Create new template
kidp catalog create-template \
  --name my-awesome-pattern \
  --category application-archetype \
  --output templates/my-pattern.yaml

# Test template locally
kidp catalog validate templates/my-pattern.yaml

# Generate sample output
kidp catalog render templates/my-pattern.yaml \
  --param serviceName=test-app \
  --param team=my-team

# Test deployment in sandbox
kidp create from-template my-awesome-pattern \
  --param serviceName=test-app \
  --sandbox
```

#### 2. Review & Approval
```bash
# Submit for review
kidp catalog submit templates/my-pattern.yaml \
  --maturity experimental

# Automated checks run:
# ✓ YAML syntax valid
# ✓ All required fields present
# ✓ Cost estimates provided
# ✓ Documentation links valid
# ✓ Security scan passed
# ✓ Policy compliance checked

# Platform team reviews in PR
# - Architecture review
# - Cost review
# - Security review
# - Documentation completeness
```

#### 3. Publishing
```bash
# Publish to catalog (requires approval)
kidp catalog publish templates/my-pattern.yaml \
  --maturity beta \
  --category application-archetype

# Template now discoverable
kidp catalog list | grep my-awesome-pattern

# Monitor adoption
kidp catalog stats my-awesome-pattern
```

### Template Lifecycle

#### Maturity Stages
```
experimental → alpha → beta → stable → deprecated
     ↓          ↓       ↓       ↓          ↓
  Internal   Limited  Public  Production  Sunset
   testing    pilot    GA      ready      notice
```

**Experimental**
- Available only to template authors
- No SLA, may break
- Active development

**Alpha**
- Available to volunteer pilot teams
- Breaking changes possible
- Feedback collection phase

**Beta**
- Available to all teams
- API relatively stable
- Production-ready for non-critical workloads

**Stable**
- Fully supported
- Backward compatibility guaranteed
- SLA enforced

**Deprecated**
- Still available but not recommended
- Migration guide provided
- Eventual removal date announced

#### Version Management
```yaml
# Multiple versions can coexist
web-service-standard:v1.0.0  # deprecated
web-service-standard:v2.0.0  # stable
web-service-standard:v2.3.0  # stable (latest)
web-service-standard:v3.0.0  # beta

# Default behavior
kidp create from-template web-service-standard
# Uses: v2.3.0 (latest stable)

# Pin to specific version
kidp create from-template web-service-standard:v2.0.0

# Use bleeding edge
kidp create from-template web-service-standard:v3.0.0
```

### Template Updates & Migration

#### Breaking Changes
```bash
# When v3.0.0 introduces breaking changes
$ kidp catalog diff web-service-standard:v2.3.0 v3.0.0

Breaking Changes (v2.3.0 → v3.0.0):
├── Parameter 'databaseEngine' renamed to 'dbEngine'
├── Parameter 'replicas' now requires minimum: 3 (was 2)
├── New required parameter: 'environment'
└── Removed parameter: 'legacyMode'

Migration Guide:
https://docs.company.com/catalog/web-service/migration-v3

Automated Migration:
kidp catalog migrate \
  --from web-service-standard:v2.3.0 \
  --to web-service-standard:v3.0.0 \
  --path manifests/
```

#### Automatic Updates
```yaml
# In template metadata
apiVersion: catalog.platform.company.com/v1
kind: CatalogTemplate
metadata:
  annotations:
    # Pin to major version, auto-update minor/patch
    catalog.platform.company.com/version-policy: "^2.0.0"
    # Or: latest stable
    catalog.platform.company.com/version-policy: "stable"
    # Or: exact version (no auto-update)
    catalog.platform.company.com/version-policy: "2.3.0"
```

## Integration Connectors

### Third-Party Service Integration

#### Connector Architecture
```yaml
apiVersion: catalog.platform.company.com/v1
kind: IntegrationConnector
metadata:
  name: datadog-apm
spec:
  displayName: "Datadog APM Integration"
  description: "Automatic Datadog APM instrumentation for services"
  
  vendor:
    name: "Datadog"
    website: "https://www.datadoghq.com"
    support: "https://docs.datadoghq.com/help/"
  
  category: observability
  
  authentication:
    type: api-key
    secretTemplate: |
      apiVersion: v1
      kind: Secret
      metadata:
        name: datadog-api-key
      data:
        api-key: {{ .apiKey | b64enc }}
        app-key: {{ .appKey | b64enc }}
  
  resources:
    - kind: Service
      inject:
        envVars:
          - name: DD_AGENT_HOST
            value: "datadog-agent.monitoring.svc.cluster.local"
          - name: DD_SERVICE
            valueFrom:
              fieldRef:
                fieldPath: metadata.name
        sidecars:
          - name: datadog-agent
            image: datadog/agent:latest
            env:
              - name: DD_API_KEY
                valueFrom:
                  secretKeyRef:
                    name: datadog-api-key
                    key: api-key
  
  configuration:
    parameters:
      - name: enableProfiling
        type: boolean
        default: false
      - name: sampleRate
        type: number
        default: 1.0
        minimum: 0.0
        maximum: 1.0
  
  cost:
    model: "usage-based"
    estimatedMonthly: 150
    billedBy: "vendor"
```

#### Using Connectors
```bash
# Browse available connectors
kidp catalog list --category integration

# Add integration to service
kidp integration add datadog-apm \
  --to service/user-api \
  --param apiKey=$DD_API_KEY \
  --param enableProfiling=true

# List active integrations
kidp integration list --service user-api

# Remove integration
kidp integration remove datadog-apm --from service/user-api
```

#### Popular Connectors

**Observability**
- Datadog APM & Logs
- New Relic
- Splunk
- Honeycomb

**Data Stores**
- Snowflake
- MongoDB Atlas
- Confluent Cloud (Kafka)
- Elastic Cloud

**Communication**
- Twilio (SMS/Voice)
- SendGrid (Email)
- Slack API

**Security**
- HashiCorp Vault
- 1Password Secrets
- AWS Secrets Manager

## Community & Collaboration

### Template Sharing

#### Internal Marketplace
```yaml
# Team-contributed templates
apiVersion: catalog.platform.company.com/v1
kind: CatalogTemplate
metadata:
  name: ml-training-pipeline
  namespace: team-data-science  # Team namespace
  labels:
    catalog.platform.company.com/visibility: organization
    catalog.platform.company.com/maturity: alpha
spec:
  displayName: "ML Training Pipeline"
  maintainers:
    - name: "Data Science Team"
      slack: "#team-data-science"
  
  support:
    tier: community  # Community-supported, not platform team
```

#### Visibility Levels
```yaml
# Private: Only visible to owning team
catalog.platform.company.com/visibility: private

# Team: Visible to team members only
catalog.platform.company.com/visibility: team

# Organization: Visible to all teams (default)
catalog.platform.company.com/visibility: organization

# Public: Published to external catalog (future)
catalog.platform.company.com/visibility: public
```

### Ratings & Reviews

#### User Feedback
```bash
# Rate a template
kidp catalog rate web-service-standard --stars 5

# Leave review
kidp catalog review web-service-standard \
  --comment "Great template! Deployed in 10 minutes."

# View reviews
kidp catalog reviews web-service-standard

Reviews (4.7 ⭐ from 142 ratings)
────────────────────────────────────────
⭐⭐⭐⭐⭐  Sarah Chen (backend-team)
"Excellent starter template. Documentation could use
more examples for custom domains."

⭐⭐⭐⭐⭐  Mike Johnson (frontend-team)
"This is the way. Deployed 3 services with it."

⭐⭐⭐⭐☆  Alex Kumar (data-team)
"Good but database backup config was confusing."
```

### Template Analytics

#### Usage Metrics
```bash
kidp catalog stats web-service-standard

Template: Web Service (Standard) v2.3.0
────────────────────────────────────────
📊 Deployments
├── Total: 847
├── Last 30 days: 89
├── Success rate: 98.5%
└── Avg. deployment time: 17 minutes

👥 Adoption
├── Teams using: 23 / 45 (51%)
├── Top users: backend-team (187), api-team (134)
└── New users (30d): 4 teams

💰 Cost Impact
├── Total infrastructure: $637,250/month
├── Average per deployment: $752/month
└── Cost efficiency: +12% vs manual setup

⚡ Performance
├── Time saved per deployment: ~4 hours
├── Total time saved: 3,388 hours
└── Estimated value: $508,200

🐛 Issues
├── Open: 2
├── Resolved: 47
└── Avg. resolution time: 1.2 days
```

## Policy Packs

### Governance Bundles

#### Policy Pack Structure
```yaml
apiVersion: catalog.platform.company.com/v1
kind: PolicyPack
metadata:
  name: pci-compliance
spec:
  displayName: "PCI-DSS Compliance Pack"
  description: "Required policies for PCI-DSS compliant workloads"
  
  policies:
    - name: require-encryption-at-rest
      description: "All databases must enable encryption at rest"
      rule: |
        spec.encryption.atRest.enabled == true
      targets:
        - apiVersion: platform.company.com/v1
          kind: Database
    
    - name: require-ssl-tls
      description: "All services must enforce TLS"
      rule: |
        spec.tls.enabled == true && spec.tls.minVersion >= "1.2"
      targets:
        - apiVersion: platform.company.com/v1
          kind: Service
    
    - name: require-audit-logging
      description: "Audit logging must be enabled"
      rule: |
        spec.logging.audit.enabled == true
      targets:
        - apiVersion: platform.company.com/v1
          kind: Database
        - apiVersion: platform.company.com/v1
          kind: Service
    
    - name: restrict-public-access
      description: "No resources publicly accessible"
      rule: |
        spec.networking.publicAccess == false
      targets:
        - apiVersion: platform.company.com/v1
          kind: Database
  
  templateAnnotations:
    # Templates marked with this pack auto-apply policies
    catalog.platform.company.com/policy-pack: pci-compliance
  
  enforcement:
    mode: enforce  # enforce | warn | audit
    exceptions:
      - namespace: team-security  # Can override
```

#### Applying Policy Packs
```bash
# Apply to namespace
kidp policy apply pci-compliance --namespace team-payments

# Apply to template
kidp catalog annotate web-service-standard \
  --policy-pack pci-compliance

# Check compliance
kidp policy check --namespace team-payments

Compliance Report: team-payments
────────────────────────────────
Policy Pack: PCI-DSS Compliance

✓ require-encryption-at-rest (3/3 resources)
✓ require-ssl-tls (2/2 resources)
✗ require-audit-logging (1/3 resources)
  ├── ✗ payment-db: audit logging disabled
  └── ✗ payment-api: audit logging disabled
✓ restrict-public-access (3/3 resources)

Compliance Score: 75% (3/4 policies passing)
```

## Success Metrics

### Adoption KPIs
```yaml
# Platform-level metrics
catalog_templates_total: 87
catalog_templates_stable: 34
catalog_deployments_total: 2847
catalog_adoption_rate: 0.76  # 76% of deployments use templates

# Template-level metrics
template_usage_count{template="web-service-standard"}: 847
template_success_rate{template="web-service-standard"}: 0.985
template_avg_rating{template="web-service-standard"}: 4.7
template_time_saved_hours{template="web-service-standard"}: 3388

# Cost metrics
catalog_cost_savings_usd: 1250000  # vs. manual setup
catalog_total_spend_usd: 8500000
```

### Business Value
- **Time to First Deployment**: Reduced from 2-3 days to 15-30 minutes
- **Configuration Errors**: Reduced by 85% (standardization)
- **Developer Productivity**: +40% (focus on code, not infrastructure)
- **Cost Efficiency**: +12% (optimized resource sizing)
- **Compliance Coverage**: 100% for PCI/SOX workloads
- **Knowledge Sharing**: 76% template adoption rate

## Future Enhancements

### Phase 1 (MVP)
- Core catalog with 10-15 essential templates
- CLI-based browsing and deployment
- Basic search and filtering
- Template versioning

### Phase 2 (Growth)
- Web UI catalog browser
- Template ratings and reviews
- Integration connectors (5-10 popular services)
- Policy pack system

### Phase 3 (Scale)
- AI-powered recommendations
- Automated template optimization
- Cost optimization suggestions
- Multi-cloud template support

### Phase 4 (Advanced)
- Community marketplace (external sharing)
- Template marketplace analytics
- A/B testing for templates
- Automated template generation from existing infrastructure
- Template composition (combine multiple templates)

## Implementation Considerations

### Storage
- Templates stored as CRDs in management cluster
- Git repository for template source code
- Container registry for template icons/assets

### Security
- Template validation before publishing
- Security scanning for embedded configurations
- RBAC for template publishing
- Audit logging for template usage

### Performance
- Template caching for fast rendering
- Pre-computed cost estimates
- Lazy loading for catalog browsing
- CDN for static assets

### Integration Points
- GitOps repository (template output)
- CI/CD pipeline (automated testing)
- Cost management system (cost data)
- Observability platform (usage metrics)
- Compliance system (policy enforcement)
