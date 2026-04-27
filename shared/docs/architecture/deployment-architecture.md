# Deployment Architecture

This document describes the deployment topology, CI/CD pipeline, environment strategy, and scaling approach for the Real Assessment Platform.

---

## 1. Deployment Topology

### 1.1 High-Level Architecture

```
                     Internet
                         |
                    [Cloudflare / CDN]
                         |
                    [Application Load Balancer]
                         |
                    +----+----+
                    |  WAF /  |
                    |  TLS    |
                    +----+----+
                         |
                    [Kubernetes Cluster]
                    |
        +-----------+-----------+-----------+
        |           |           |           |
   +----v----+ +----v----+ +----v----+ +----v----+
   |  API    | |  User   | | Resume  | |Interview|
   | Gateway | | Service | | Service | | Service |
   |  x3     | |  x3     | |  x2     | |  x3     |
   +---------+ +---------+ +---------+ +---------+
        |           |           |           |
        +-----------+-----------+-----------+
                    |
   +----------------+----------------+
   |                |                |
+--v------+   +----v----+    +------v------+
|  AI     |   | Scoring |    |   Admin     |
| Service |   | Service |    |   Service   |
|  x4     |   |  x2     |    |   x2        |
+---------+   +---------+    +-------------+

  +-----------------------------------------+
  |         Infrastructure Services         |
  |  PostgreSQL | Redis | Kafka | S3 | Jaeger|
  +-----------------------------------------+
```

### 1.2 Namespace Layout

```
kubernetes
├── namespace: assessment-production
│   ├── api-gateway (Deployment x3, HPA, PDB)
│   ├── user-service (Deployment x3, HPA)
│   ├── resume-service (Deployment x2, HPA)
│   ├── interview-service (Deployment x3, HPA)
│   ├── ai-service (Deployment x4, HPA)
│   ├── scoring-service (Deployment x2, HPA)
│   ├── admin-service (Deployment x2)
│   └── monitoring (Prometheus, Grafana, Jaeger)
│
├── namespace: assessment-staging
│   └── (same services, 2 replicas each)
│
├── namespace: assessment-databases
│   ├── postgresql (StatefulSet, 3 replicas)
│   ├── redis-cluster (StatefulSet, 3 nodes)
│   └── kafka (StatefulSet, 3 brokers)
│
└── namespace: ingress-system
    ├── nginx-ingress-controller
    └── cert-manager (Let's Encrypt)
```

---

## 2. CI/CD Pipeline

### 2.1 Pipeline Overview

```
Developer Push --> CI Build --> Tests --> Image Build --> Push to Registry
                                                                 |
                                                          CD Deploy
                                                          (staging)
                                                                 |
                                                       Manual Approval
                                                                 |
                                                       CD Deploy (prod)
```

### 2.2 CI Stages

```yaml
# Simplified GitHub Actions / GitLab CI pipeline

stages:
  - lint
  - test
  - build
  - scan
  - deploy-staging
  - deploy-production

lint:
  # Go services
  - cd services/user-service && golangci-lint run
  - cd services/resume-service && golangci-lint run
  - cd services/interview-service && golangci-lint run
  - cd services/api-gateway && golangci-lint run
  # Python services
  - cd services/ai-service && ruff check && mypy .
  - cd services/scoring-service && ruff check && mypy .
  # Shared
  - cd shared/protobuf && buf lint
  - cd shared/packages/python-common && ruff check

test:
  # Unit tests
  - go test ./... -race -cover
  - pytest --cov=python_common
  # Integration tests (with test containers)
  - make test-integration

build:
  # Build container images
  - docker build -t assessment/api-gateway:$SHA ./services/api-gateway
  - docker build -t assessment/user-service:$SHA ./services/user-service
  # ... all services
  - docker push assessment/*:$SHA

scan:
  # Security scanning
  - trivy image assessment/api-gateway:$SHA
  - trivy image assessment/user-service:$SHA
  # ... all images

deploy-staging:
  # Deploy to staging namespace
  - helm upgrade --install assessment ./helm/assessment \
      --namespace assessment-staging \
      --set image.tag=$SHA \
      --values config/environments/staging.yaml
  - run integration tests against staging
  - run smoke tests

deploy-production:
  # Deploy to production (manual approval gate)
  - helm upgrade --install assessment ./helm/assessment \
      --namespace assessment-production \
      --set image.tag=$SHA \
      --values config/environments/production.yaml \
      --wait --timeout=10m
  - run production smoke tests
  - if failed: helm rollback assessment --namespace assessment-production
```

### 2.3 Deployment Strategy

- **Rolling updates:** Default strategy for all services
- **Blue/green:** For major version changes on API Gateway
- **Canary:** 10% traffic for first 5 minutes, then full rollout (AI Service)

---

## 3. Environment Strategy

| Environment | Purpose | Data | Access | Scaling |
|-------------|---------|------|--------|---------|
| **Development** | Local dev with docker-compose | Seeded test data | Developer machines | Single replica |
| **Staging** | Pre-production validation | Anonymized production snapshot | Internal team only | 2 replicas, matches prod topology |
| **Production** | Live user traffic | Real user data | Public (authenticated) | Auto-scaled, multi-AZ |

### Environment Promotion

```
Code --> Dev (local) --> Staging (automated) --> Production (manual approval)
```

---

## 4. Scaling Strategy

### 4.1 Horizontal Pod Autoscaling (HPA)

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-gateway-hpa
  namespace: assessment-production
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api-gateway
  minReplicas: 3
  maxReplicas: 15
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
    - type: Pods
      pods:
        metric:
          name: http_requests_per_second
        target:
          type: AverageValue
          averageValue: "500"
```

### 4.2 Scaling Guidelines per Service

| Service | Min Replicas | Max Replicas | Scale Trigger |
|---------|-------------|-------------|---------------|
| API Gateway | 3 | 15 | CPU 70%, RPS 500 |
| User Service | 2 | 8 | CPU 70% |
| Resume Service | 2 | 6 | CPU 75% |
| Interview Service | 2 | 8 | CPU 70% |
| AI Service | 3 | 12 | CPU 60% (GPU if available) |
| Scoring Service | 2 | 6 | Kafka consumer lag |
| Admin Service | 1 | 3 | CPU 70% |

### 4.3 Database Scaling

- **Vertical:** Increase instance class for PostgreSQL when connection count or IOPS hits limits
- **Read replicas:** Add read replicas for read-heavy services (Resume Service queries)
- **Connection pooling:** PgBouncer in transaction mode, pool size = 25 per instance

### 4.4 Kafka Scaling

- Add brokers (3 --> 5) for high availability
- Increase partitions per topic for throughput (default: 12, max: 48)
- Consumer groups scale automatically with partition count

---

## 5. Networking

### 5.1 Ingress Configuration

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: assessment-ingress
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: "50m"
    nginx.ingress.kubernetes.io/rate-limit: "100"
spec:
  tls:
    - hosts:
        - app.assessment.example.com
        - admin.assessment.example.com
      secretName: assessment-tls
  rules:
    - host: app.assessment.example.com
      http:
        paths:
          - path: /api
            pathType: Prefix
            backend:
              service:
                name: api-gateway
                port:
                  number: 8080
    - host: admin.assessment.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: admin-service
                port:
                  number: 8086
```

### 5.2 Internal Communication

- All inter-service traffic uses Kubernetes Service DNS
- mTLS via Istio (optional, production only)
- gRPC traffic stays within cluster network

---

## 6. Storage

| Resource | Type | Size | Backup |
|----------|------|------|--------|
| PostgreSQL | PersistentVolume (SSD) | 100 GB (auto-expand) | Daily snapshots, 30-day retention |
| Redis | In-memory + AOF persistence | N/A | AOF append-only file |
| Kafka | PersistentVolume | 500 GB per broker | Topic replication factor 3 |
| S3 | Object storage | Unlimited | Cross-region replication |

---

*See also: [System Design](./system-design.md) | [Local Setup](../deployment/local-setup.md) | [Production Deployment](../deployment/production-deployment.md)*
