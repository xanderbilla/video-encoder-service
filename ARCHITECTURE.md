# Architecture Documentation

## Overview

Video Encoder Service is built using **Clean Architecture** principles, ensuring maintainability, testability, and scalability.

## Architecture Layers

```
┌─────────────────────────────────────────────────────────┐
│                     API Layer                            │
│  (HTTP Handlers, Middleware, Routes)                    │
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│                 Application Layer                        │
│  (Use Cases, Application Services)                      │
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│                   Core/Domain Layer                      │
│  (Business Logic, Entities, Ports/Interfaces)           │
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│                Infrastructure Layer                      │
│  (FFmpeg, Filesystem, Cache, External Services)         │
└─────────────────────────────────────────────────────────┘
```

## Project Structure

```
encoder-service/
├── cmd/
│   └── api/                    # Application entry point
│       └── main.go
│
├── internal/
│   ├── api/                    # API/Presentation Layer
│   │   ├── handlers/           # HTTP request handlers
│   │   ├── middleware/         # HTTP middleware
│   │   └── routes/             # Route definitions
│   │
│   ├── application/            # Application Layer
│   │   ├── services/           # Application services
│   │   └── usecases/           # Business use cases
│   │
│   ├── core/                   # Core/Domain Layer
│   │   ├── domain/             # Domain models & errors
│   │   └── ports/              # Interfaces/Ports
│   │
│   └── infrastructure/         # Infrastructure Layer
│       ├── ffmpeg/             # FFmpeg integration
│       ├── filesystem/         # File operations & storage
│       └── cache/              # Caching implementation
│
├── pkg/                        # Public packages
│   ├── config/                 # Configuration management
│   ├── constants/              # Application constants
│   ├── types/                  # Shared types
│   └── utils/                  # Utility functions
│
├── docs/                       # Documentation
│   ├── API.md                  # API documentation
│   └── FEATURES.md             # Features & advantages
│
├── chunks/                     # Temporary chunk storage
├── outputs/                    # Encoded video outputs
├── cache/                      # Deduplication cache
├── videos/                     # Input videos
└── logs/                       # Application logs
```

## Layer Responsibilities

### 1. API Layer (`internal/api/`)

**Purpose**: Handle HTTP communication

- **Handlers**: Process HTTP requests and responses
- **Middleware**: Request ID generation, logging, error handling
- **Routes**: HTTP route definitions and setup

**Key Files**:

- `handlers/job_handler.go` - Job management endpoints
- `middleware/request_id.go` - Request ID middleware
- `middleware/logging.go` - Request logging
- `routes/routes.go` - Route configuration

### 2. Application Layer (`internal/application/`)

**Purpose**: Orchestrate business logic

- **Services**: Application-level services (JobService, DiskService)
- **Use Cases**: Business workflows (TranscodeUseCase)

**Key Files**:

- `services/job_service.go` - Job lifecycle management
- `services/disk_service.go` - Disk space management
- `usecases/transcode_usecase.go` - Transcoding workflow

### 3. Core Layer (`internal/core/`)

**Purpose**: Define business rules and contracts

- **Domain**: Business entities and domain errors
- **Ports**: Interfaces for external dependencies

**Key Files**:

- `domain/errors.go` - Domain-specific errors
- `ports/repository.go` - Job storage interface
- `ports/encoder.go` - Video encoding interface
- `ports/validator.go` - File validation interface
- `ports/cache.go` - Caching interface

### 4. Infrastructure Layer (`internal/infrastructure/`)

**Purpose**: Implement external integrations

- **FFmpeg**: Video encoding and probing
- **Filesystem**: File validation and storage
- **Cache**: Deduplication cache

**Key Files**:

- `ffmpeg/encoder.go` - HLS encoding implementation
- `ffmpeg/probe.go` - Video metadata extraction
- `filesystem/validator.go` - File validation
- `filesystem/repository.go` - In-memory job storage
- `cache/deduplication.go` - SHA256-based caching

### 5. Package Layer (`pkg/`)

**Purpose**: Shared utilities and types

- **Config**: Environment-based configuration
- **Constants**: Application-wide constants
- **Types**: Shared data structures
- **Utils**: Common utility functions

**Key Files**:

- `config/config.go` - Configuration loader
- `constants/constants.go` - Application constants
- `types/job.go` - Job data structures
- `types/encoding.go` - Encoding types
- `types/response.go` - API response types
- `utils/response.go` - Response helpers
- `utils/file.go` - File utilities
- `utils/disk.go` - Disk utilities

## Design Principles

### 1. Dependency Inversion

- Dependencies point inward (toward domain)
- Outer layers depend on inner layers
- Inner layers define interfaces (ports)
- Outer layers implement interfaces

### 2. Separation of Concerns

- Each layer has a specific responsibility
- Business logic isolated from infrastructure
- Easy to test and maintain

### 3. DRY (Don't Repeat Yourself)

- Common functionality in `pkg/utils/`
- Shared types in `pkg/types/`
- Reusable constants in `pkg/constants/`

### 4. Interface-Based Design

- Core layer defines interfaces (ports)
- Infrastructure implements interfaces
- Easy to mock and test
- Flexible to swap implementations

## Data Flow

### Request Flow

```
HTTP Request
    ↓
Middleware (Request ID, Logging)
    ↓
Handler (Validation, Response)
    ↓
Use Case (Business Logic)
    ↓
Service (Application Logic)
    ↓
Repository/Encoder (Infrastructure)
    ↓
External System (FFmpeg, Filesystem)
```

### Job Processing Flow

```
1. Client submits job via POST /transcode
2. Handler validates request
3. JobService creates job
4. Job added to queue
5. Worker picks up job
6. TranscodeUseCase executes:
   - Compute file hash
   - Check for duplicates
   - Analyze video
   - Encode to HLS
   - Update job status
7. Results stored
8. Client polls GET /jobs/{id}
```

## Key Components

### Job Queue

- Channel-based queue (capacity: 100)
- FIFO processing
- Non-blocking submission

### Worker Pool

- Configurable workers (default: 2)
- Concurrent job processing
- Active worker tracking

### Deduplication Cache

- SHA256 file hashing
- Persistent JSON storage
- Instant duplicate detection

### Disk Manager

- Real-time usage monitoring
- Automatic cleanup
- Configurable thresholds

### Retry Mechanism

- Exponential backoff
- Configurable attempts (default: 3)
- Job state preservation

## Configuration

All configuration via environment variables:

```env
# Server
PORT=8080
SHUTDOWN_TIMEOUT=30

# Workers
MAX_WORKERS=2
MAX_RETRIES=3
HEARTBEAT_INTERVAL=10

# Storage
OUTPUTS_DIR=outputs
CHUNKS_DIR=chunks
CACHE_DIR=cache

# Encoding
MAX_FILE_SIZE_MB=1024
DEFAULT_CHUNK_DURATION=4

# Disk
DISK_THRESHOLD_PERCENT=80
DISK_HARD_LIMIT_PERCENT=90
CLEANUP_INTERVAL=600
```

## Error Handling

### Error Propagation

```
Infrastructure Error
    ↓
Domain Error (wrapped)
    ↓
Application Error (logged)
    ↓
API Error Response (structured)
```

### Error Types

- **Domain Errors**: Business rule violations
- **Validation Errors**: Input validation failures
- **Infrastructure Errors**: External system failures

## Testing Strategy

### Unit Tests

- Test individual functions
- Mock dependencies
- Focus on business logic

### Integration Tests

- Test layer interactions
- Use test doubles
- Verify workflows

### End-to-End Tests

- Test complete flows
- Use real dependencies
- Verify system behavior

## Deployment

### Docker

```bash
docker-compose up
```

### Binary

```bash
go build -o encoder-service cmd/api/main.go
./encoder-service
```

### Environment

- Development: Local with hot reload
- Staging: Docker with volume mounts
- Production: Docker with orchestration

## Monitoring

### Health Endpoint

- Service status
- Worker statistics
- Queue metrics
- Job statistics
- Disk usage

### Logging

- Structured logging
- Request ID tracking
- Error context
- Performance metrics

## Scalability

### Horizontal Scaling

- Stateless design
- Shared storage
- Load balancer ready

### Vertical Scaling

- Increase worker count
- Adjust queue capacity
- Optimize FFmpeg settings

### Future Enhancements

- Database for job storage
- Message queue (RabbitMQ/Kafka)
- Distributed caching (Redis)
- Cloud storage (S3/GCS)

## Security

### Current

- Input validation
- File size limits
- Format restrictions
- Path sanitization

### Planned

- JWT authentication
- API key management
- Rate limiting
- Audit logging

## Performance

### Optimizations

- Concurrent processing
- Efficient caching
- Resource management
- Optimized FFmpeg parameters

### Bottlenecks

- FFmpeg encoding (CPU-bound)
- Disk I/O (storage-bound)
- Network (if using remote storage)

## Maintenance

### Code Quality

- Go best practices
- Clean Architecture
- DRY principles
- Comprehensive comments

### Documentation

- API documentation
- Architecture documentation
- Feature documentation
- Inline code comments

### Versioning

- Semantic versioning
- Changelog maintenance
- Migration guides
