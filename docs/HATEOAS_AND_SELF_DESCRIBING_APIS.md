# HATEOAS and Self-Describing APIs

## Overview

This document explains the principles of HATEOAS (Hypermedia as the Engine of Application State) and self-describing APIs, demonstrating how to implement them effectively in REST services. The KIDP Deployment Broker serves as a reference implementation of these patterns.

## Table of Contents

- [What is HATEOAS?](#what-is-hateoas)
- [Benefits of Self-Describing APIs](#benefits-of-self-describing-apis)
- [Industry Standards and Patterns](#industry-standards-and-patterns)
- [Implementation Patterns](#implementation-patterns)
- [Reference Implementation](#reference-implementation)
- [Best Practices](#best-practices)
- [Examples from Other APIs](#examples-from-other-apis)

## What is HATEOAS?

HATEOAS is a constraint of the REST application architecture that keeps the API flexible and evolvable. In HATEOAS, the client interacts with the application entirely through hypermedia provided dynamically by the server.

### Core Principles

1. **Discoverability**: Clients discover available actions through links provided in responses
2. **Decoupling**: Clients don't need to hardcode API endpoints
3. **Evolvability**: API structure can change without breaking clients
4. **Self-Documentation**: The API describes itself through its responses

### The Richardson Maturity Model

HATEOAS represents Level 3 (the highest) of the Richardson Maturity Model:

```
Level 0: The Swamp of POX (Plain Old XML)
  - Single URI, single HTTP method
  - RPC-style calls

Level 1: Resources
  - Multiple URIs, but single HTTP method
  - Resources identified by URIs

Level 2: HTTP Verbs
  - Multiple URIs and HTTP methods
  - Proper use of GET, POST, PUT, DELETE, etc.

Level 3: Hypermedia Controls (HATEOAS)
  - Responses include links to related resources
  - Client driven by server responses
```

## Benefits of Self-Describing APIs

### 1. Reduced Documentation Burden

The API itself serves as live, always-up-to-date documentation:

```json
{
  "service": "KIDP Deployment Broker",
  "version": "0.1.0",
  "documentation": "https://github.com/aykay76/kidp/blob/master/docs/BROKER_API.md",
  "endpoints": {
    "provision": {
      "method": "POST",
      "path": "/v1/provision",
      "description": "Provision a new resource",
      "request": { /* example payload */ },
      "response": { /* example response */ }
    }
  }
}
```

### 2. Improved Developer Experience

Developers can:
- Start exploring from the root endpoint
- Understand capabilities without reading documentation
- See example requests/responses inline
- Discover features through API responses

### 3. Client Resilience

Clients that follow links instead of hardcoding URLs:
- Continue working when endpoints change
- Discover new features automatically
- Handle API versioning gracefully

### 4. API Evolution

Server changes don't break clients:
- Add new endpoints without client updates
- Deprecate features gradually (links disappear)
- Version resources independently

### 5. Testing and Monitoring

Self-describing APIs enable:
- Automated API testing tools
- Dynamic client generation
- Better observability and debugging

## Industry Standards and Patterns

### HAL (Hypertext Application Language)

HAL is a simple format that provides a consistent way to hyperlink between resources:

```json
{
  "_links": {
    "self": { "href": "/orders/123" },
    "customer": { "href": "/customers/456" },
    "items": { "href": "/orders/123/items" }
  },
  "orderId": "123",
  "total": 99.99,
  "status": "shipped"
}
```

**Key Features**:
- `_links` object for hypermedia controls
- `_embedded` for nested resources
- Simple and widely supported

### JSON:API

A specification for building APIs in JSON with relationships:

```json
{
  "data": {
    "type": "articles",
    "id": "1",
    "attributes": {
      "title": "HATEOAS Guide"
    },
    "relationships": {
      "author": {
        "links": {
          "related": "/articles/1/author"
        }
      }
    }
  },
  "links": {
    "self": "/articles/1"
  }
}
```

**Key Features**:
- Standardized structure for resources
- Relationship handling
- Pagination, sorting, filtering conventions
- Error format specification

### Siren

A hypermedia specification for representing entities:

```json
{
  "class": ["order"],
  "properties": {
    "orderNumber": 123,
    "status": "pending"
  },
  "entities": [
    {
      "class": ["items"],
      "rel": ["http://x.io/rels/order-items"],
      "href": "/orders/123/items"
    }
  ],
  "actions": [
    {
      "name": "cancel-order",
      "title": "Cancel Order",
      "method": "DELETE",
      "href": "/orders/123"
    }
  ],
  "links": [
    { "rel": ["self"], "href": "/orders/123" }
  ]
}
```

**Key Features**:
- Actions describe available operations
- Rich entity representation
- Strong typing with classes

### OpenAPI (Swagger)

While not strictly HATEOAS, OpenAPI provides machine-readable API descriptions:

```yaml
openapi: 3.0.0
info:
  title: Deployment Broker API
  version: 1.0.0
paths:
  /v1/provision:
    post:
      summary: Provision a resource
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ProvisionRequest'
```

**Key Features**:
- Comprehensive API specification
- Code generation support
- Interactive documentation (Swagger UI)
- Wide tooling ecosystem

## Implementation Patterns

### Pattern 1: Root Endpoint Discovery

The root endpoint (`/`) should provide a map of the entire API:

```json
{
  "service": "My API",
  "version": "1.0.0",
  "description": "Service description",
  "documentation": "https://docs.example.com",
  
  "_links": {
    "self": { "href": "/", "method": "GET" },
    "users": { "href": "/users", "method": "GET" },
    "create-user": { "href": "/users", "method": "POST" }
  },
  
  "endpoints": {
    "users": {
      "path": "/users",
      "methods": ["GET", "POST"],
      "description": "User management"
    }
  }
}
```

**Implementation in Go**:

```go
func handleRoot(w http.ResponseWriter, r *http.Request) {
    response := map[string]interface{}{
        "service": "My API",
        "version": "1.0.0",
        "_links": map[string]interface{}{
            "self": map[string]string{
                "href": "/",
                "method": "GET",
            },
            "users": map[string]string{
                "href": "/users",
                "method": "GET",
            },
        },
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

### Pattern 2: Capability Declaration

Declare what the service can do upfront:

```json
{
  "capabilities": {
    "resourceTypes": ["database", "cache", "queue"],
    "databases": ["postgresql", "mysql", "mongodb"],
    "features": [
      "drift-detection",
      "auto-scaling",
      "backup-restore"
    ]
  }
}
```

This helps clients:
- Validate compatibility before making requests
- Display appropriate UI options
- Route requests to capable services

### Pattern 3: Detailed Endpoint Documentation

Include examples directly in responses:

```json
{
  "endpoints": {
    "provision": {
      "method": "POST",
      "path": "/v1/provision",
      "description": "Provision a new resource",
      "contentType": "application/json",
      "request": {
        "resourceType": "database",
        "spec": {
          "engine": "postgresql",
          "version": "15",
          "size": "medium"
        }
      },
      "response": {
        "status": "accepted",
        "deploymentId": "deploy-abc123"
      }
    }
  }
}
```

### Pattern 4: Link Relations

Use standard or custom link relations to indicate relationships:

```json
{
  "_links": {
    "self": { "href": "/orders/123" },
    "payment": { "href": "/payments/789", "rel": "related" },
    "cancel": { "href": "/orders/123/cancel", "rel": "action" },
    "next": { "href": "/orders/124", "rel": "next" }
  }
}
```

**Standard Relations** (IANA registered):
- `self`: The current resource
- `next`/`prev`: Pagination
- `first`/`last`: Collection boundaries
- `up`: Parent resource
- `related`: Related resource
- `edit`: Editable version

### Pattern 5: HTTP Method Hints

Tell clients which HTTP methods are available:

```json
{
  "_links": {
    "resources": {
      "href": "/v1/resources",
      "methods": ["GET", "POST"],
      "accepts": "application/json"
    }
  }
}
```

Or use HTTP OPTIONS:

```http
OPTIONS /v1/resources HTTP/1.1

HTTP/1.1 200 OK
Allow: GET, POST, OPTIONS
```

### Pattern 6: API Versioning Information

Communicate versioning strategy:

```json
{
  "api": {
    "version": "v1",
    "minApiVersion": "v1",
    "deprecatedApis": [],
    "sunset": null
  }
}
```

For deprecated endpoints, use HTTP headers:

```http
HTTP/1.1 200 OK
Deprecation: true
Sunset: Sat, 31 Dec 2025 23:59:59 GMT
Link: </v2/new-endpoint>; rel="successor-version"
```

### Pattern 7: Runtime Status

Include operational information:

```json
{
  "runtime": {
    "status": "healthy",
    "uptime": "72h35m",
    "kubernetesConnected": true,
    "version": "0.1.0"
  }
}
```

This helps with:
- Monitoring and alerting
- Debugging connectivity issues
- Understanding service state

## Reference Implementation

The KIDP Deployment Broker implements all these patterns. Here's the complete root endpoint response:

```json
{
  "service": "KIDP Deployment Broker",
  "description": "Stateless broker for provisioning and managing resources in Kubernetes clusters",
  "version": "0.1.0",
  "status": "running",
  
  "documentation": "https://github.com/aykay76/kidp/blob/master/docs/BROKER_API.md",
  "repository": "https://github.com/aykay76/kidp",
  "support": "https://github.com/aykay76/kidp/issues",
  
  "capabilities": {
    "resourceTypes": ["database", "cache", "queue"],
    "databases": ["postgresql", "mysql", "mongodb", "redis"],
    "features": [
      "drift-detection",
      "async-provisioning",
      "health-monitoring"
    ]
  },
  
  "endpoints": {
    "health": {
      "method": "GET",
      "path": "/health",
      "description": "Returns broker service health status",
      "response": {
        "status": "healthy",
        "version": "0.1.0"
      }
    },
    "provision": {
      "method": "POST",
      "path": "/v1/provision",
      "contentType": "application/json",
      "description": "Provision a new resource in the target Kubernetes cluster",
      "request": {
        "namespace": "team-platform",
        "resourceType": "database",
        "resourceName": "my-db",
        "team": "platform-team",
        "owner": "user@example.com",
        "spec": {
          "engine": "postgresql",
          "version": "15",
          "size": "medium"
        },
        "callbackUrl": "http://manager:9090/v1/callback"
      },
      "response": {
        "status": "accepted",
        "deploymentId": "deploy-abc123",
        "message": "Provisioning request accepted"
      }
    },
    "resources": {
      "path": "/v1/resources",
      "methods": ["GET", "POST"],
      "description": "Query actual state of resources for drift detection",
      "parameters": {
        "namespace": "namespace (required)",
        "resourceType": "filter by type (optional)",
        "resourceName": "filter by name (optional)",
        "deploymentId": "filter by deployment (optional)"
      },
      "features": [
        "drift-detection",
        "health-status",
        "resource-usage",
        "cost-tracking"
      ],
      "example": "/v1/resources?namespace=team-platform&resourceType=database"
    }
  },
  
  "_links": {
    "self": {
      "href": "/",
      "method": "GET"
    },
    "health": {
      "href": "/health",
      "method": "GET"
    },
    "readiness": {
      "href": "/readiness",
      "method": "GET"
    },
    "provision": {
      "href": "/v1/provision",
      "method": "POST"
    },
    "deprovision": {
      "href": "/v1/deprovision",
      "method": "POST"
    },
    "resources": {
      "href": "/v1/resources",
      "methods": "GET, POST"
    }
  },
  
  "api": {
    "version": "v1",
    "minApiVersion": "v1",
    "deprecatedApis": []
  },
  
  "runtime": {
    "uptime": "72h35m",
    "kubernetesConnected": true
  }
}
```

### Implementation Code

See `cmd/broker/main.go` for the complete implementation:

```go
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
    response := map[string]interface{}{
        "service": "KIDP Deployment Broker",
        "description": "Stateless broker for provisioning and managing resources in Kubernetes clusters",
        "version": "0.1.0",
        "status": "running",
        
        // Documentation and support
        "documentation": "https://github.com/aykay76/kidp/blob/master/docs/BROKER_API.md",
        "repository": "https://github.com/aykay76/kidp",
        "support": "https://github.com/aykay76/kidp/issues",
        
        // Capabilities
        "capabilities": map[string]interface{}{
            "resourceTypes": []string{"database", "cache", "queue"},
            "databases": []string{"postgresql", "mysql", "mongodb", "redis"},
            "features": []string{
                "drift-detection",
                "async-provisioning", 
                "health-monitoring",
            },
        },
        
        // Detailed endpoint documentation with examples
        "endpoints": map[string]interface{}{
            // ... endpoint definitions
        },
        
        // HATEOAS links
        "_links": map[string]interface{}{
            "self": map[string]string{
                "href": "/",
                "method": "GET",
            },
            // ... other links
        },
        
        // API versioning
        "api": map[string]interface{}{
            "version": "v1",
            "minApiVersion": "v1",
            "deprecatedApis": []string{},
        },
        
        // Runtime information
        "runtime": map[string]interface{}{
            "uptime": time.Since(s.startTime).String(),
            "kubernetesConnected": s.k8sClient != nil,
        },
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

## Best Practices

### 1. Design Root Endpoint First

Make `/` your API's front door:
- Start here for new developers
- Include links to all major resources
- Provide service metadata
- Show capabilities and features

### 2. Use Consistent Link Format

Pick a format and stick to it:

```json
{
  "_links": {
    "resource": {
      "href": "/path",
      "method": "GET",
      "type": "application/json"
    }
  }
}
```

### 3. Include Examples

Real examples are worth a thousand words:

```json
{
  "endpoints": {
    "provision": {
      "request": { /* actual example */ },
      "response": { /* actual example */ }
    }
  }
}
```

### 4. Declare Capabilities Upfront

Let clients know what you can do:

```json
{
  "capabilities": {
    "features": ["feature1", "feature2"],
    "limits": {
      "maxRequestSize": "10MB",
      "rateLimit": "1000/hour"
    }
  }
}
```

### 5. Version Clearly

Make versioning explicit and predictable:

```json
{
  "api": {
    "version": "v2",
    "minApiVersion": "v1",
    "deprecatedApis": ["v0"],
    "sunset": {
      "v1": "2026-12-31"
    }
  }
}
```

### 6. Include Operational Status

Help operators and developers understand system state:

```json
{
  "runtime": {
    "status": "healthy",
    "uptime": "72h",
    "dependencies": {
      "database": "connected",
      "kubernetes": "connected"
    }
  }
}
```

### 7. Use Standard HTTP Features

Don't reinvent the wheel:
- `OPTIONS` for discovering allowed methods
- `Accept` header for content negotiation
- `Link` header for related resources
- `Deprecation` and `Sunset` headers

### 8. Make Links Actionable

Include everything clients need:

```json
{
  "_links": {
    "create-user": {
      "href": "/users",
      "method": "POST",
      "accepts": "application/json",
      "schema": "/schemas/user"
    }
  }
}
```

### 9. Document Errors Clearly

Help clients handle problems:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request",
    "details": [
      {
        "field": "email",
        "message": "Email format invalid"
      }
    ],
    "_links": {
      "documentation": {
        "href": "/docs/errors/validation-error"
      }
    }
  }
}
```

### 10. Support Multiple Formats

Consider supporting both HAL and JSON:API:

```http
GET / HTTP/1.1
Accept: application/hal+json

GET / HTTP/1.1
Accept: application/vnd.api+json
```

## Examples from Other APIs

### GitHub API

GitHub provides excellent HATEOAS implementation:

```json
{
  "current_user_url": "https://api.github.com/user",
  "authorizations_url": "https://api.github.com/authorizations",
  "repository_url": "https://api.github.com/repos/{owner}/{repo}",
  "user_url": "https://api.github.com/users/{user}",
  "_links": {
    "self": { "href": "https://api.github.com" }
  }
}
```

Every resource response includes links:

```json
{
  "id": 1,
  "name": "octocat",
  "url": "https://api.github.com/users/octocat",
  "repos_url": "https://api.github.com/users/octocat/repos",
  "events_url": "https://api.github.com/users/octocat/events{/privacy}"
}
```

### AWS API Gateway

AWS APIs include service metadata:

```json
{
  "version": "2015-07-09",
  "metadata": {
    "apiVersion": "2015-07-09",
    "endpointPrefix": "apigateway",
    "protocol": "rest-json",
    "serviceFullName": "Amazon API Gateway",
    "serviceId": "API Gateway",
    "signatureVersion": "v4"
  }
}
```

### Stripe API

Stripe includes related objects and expansion:

```json
{
  "id": "ch_123",
  "object": "charge",
  "amount": 1000,
  "customer": "cus_456",
  "url": "/v1/charges/ch_123",
  "_links": {
    "customer": "/v1/customers/cus_456",
    "refund": "/v1/charges/ch_123/refund"
  }
}
```

### Kubernetes API

Kubernetes uses self-links extensively:

```json
{
  "kind": "Pod",
  "apiVersion": "v1",
  "metadata": {
    "name": "my-pod",
    "selfLink": "/api/v1/namespaces/default/pods/my-pod",
    "uid": "abc-123"
  },
  "spec": { /* ... */ }
}
```

## Testing Self-Describing APIs

### Manual Testing

```bash
# Start at the root
curl http://localhost:8082/ | jq .

# Follow links from the response
curl http://localhost:8082/health | jq .

# Explore capabilities
curl -s http://localhost:8082/ | jq '.capabilities'

# View endpoint documentation
curl -s http://localhost:8082/ | jq '.endpoints.provision'
```

### Automated Testing

Test that the API describes itself correctly:

```go
func TestRootEndpoint(t *testing.T) {
    resp, _ := http.Get("http://localhost:8082/")
    var body map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&body)
    
    // Assert required fields exist
    assert.Contains(t, body, "service")
    assert.Contains(t, body, "version")
    assert.Contains(t, body, "_links")
    assert.Contains(t, body, "endpoints")
    
    // Validate links are actionable
    links := body["_links"].(map[string]interface{})
    for name, link := range links {
        l := link.(map[string]interface{})
        assert.Contains(t, l, "href")
        assert.Contains(t, l, "method")
    }
}
```

### Client Generator Testing

Verify that clients can be generated:

```bash
# Generate OpenAPI spec from root endpoint
curl http://localhost:8082/ | ./generate-openapi > api.yaml

# Generate client code
openapi-generator generate -i api.yaml -g go -o ./client
```

## Conclusion

HATEOAS and self-describing APIs represent the highest level of REST API maturity. They provide:

- **Developer Experience**: Easy onboarding, live documentation
- **Resilience**: Clients adapt to server changes
- **Discoverability**: Features are found through exploration
- **Professionalism**: Industry best practices

The KIDP Deployment Broker demonstrates these principles in practice. Use it as a reference when building your own services.

## Additional Resources

### Specifications
- [HAL Specification](https://datatracker.ietf.org/doc/html/draft-kelly-json-hal)
- [JSON:API Specification](https://jsonapi.org/)
- [Siren Specification](https://github.com/kevinswiber/siren)
- [OpenAPI Specification](https://spec.openapis.org/oas/latest.html)
- [IANA Link Relations](https://www.iana.org/assignments/link-relations/link-relations.xhtml)

### Books
- "RESTful Web APIs" by Leonard Richardson & Mike Amundsen
- "REST in Practice" by Jim Webber, Savas Parastatidis & Ian Robinson
- "Building Hypermedia APIs with HTML5 and Node" by Mike Amundsen

### Tools
- [HAL Browser](https://github.com/mikekelly/hal-browser) - Interactive HAL API explorer
- [Swagger UI](https://swagger.io/tools/swagger-ui/) - Interactive OpenAPI documentation
- [Postman](https://www.postman.com/) - API testing with link following

### Examples
- [GitHub API](https://docs.github.com/en/rest) - Excellent HATEOAS implementation
- [PayPal API](https://developer.paypal.com/api/rest/) - HAL-based hypermedia API
- [Stripe API](https://stripe.com/docs/api) - Clean, well-documented REST API

---

**Next Steps**: Review the [BROKER_API.md](./BROKER_API.md) for complete endpoint documentation, and see [DRIFT_DETECTION.md](./DRIFT_DETECTION.md) for how drift detection leverages the resource state endpoint.
