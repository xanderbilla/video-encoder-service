# 🎥 Video Encoder Service

Production-ready video transcoding service with HLS multi-quality encoding, intelligent deduplication, and automated disk management.

## ✨ Key Features

- **Multi-Quality HLS**: Adaptive bitrate streaming (240p to 4K)
- **Smart Deduplication**: SHA256-based caching saves time and storage
- **Auto Disk Management**: Intelligent cleanup when space is low
- **Worker Pool**: Concurrent processing with configurable workers
- **Retry Mechanism**: Automatic retry with exponential backoff
- **Clean Architecture**: Maintainable, testable, and scalable

## 🚀 Quick Start

### Using Docker (Recommended)

```bash
docker-compose up
```

### Local Development

```bash
# Install dependencies
go mod download

# Build
go build -o encoder-service cmd/api/main.go

# Run
./encoder-service
```

Service runs on `http://localhost:8080`

## 📡 API Usage

### Create Job

```bash
curl -X POST http://localhost:8080/transcode \
  -H "Content-Type: application/json" \
  -d '{"inputPath":"videos/sample.mp4","chunkDuration":4}'
```

### Check Status

```bash
curl http://localhost:8080/jobs/{jobId}
```

### Health Check

```bash
curl http://localhost:8080/health
```

## 📁 Output Structure

```
outputs/{jobId}/hls/
├── master.m3u8
├── 240p/
│   ├── main.m3u8
│   └── chunk_*.ts
├── 360p/
├── 480p/
├── 720p/
└── 1080p/
```

## ⚙️ Configuration

Configure via environment variables (see `.env.example`):

```env
PORT=8080
MAX_WORKERS=2
MAX_RETRIES=3
OUTPUTS_DIR=outputs
CHUNKS_DIR=chunks
```

## 📚 Documentation

- [API Documentation](docs/API.md) - Complete API reference
- [Features & Advantages](docs/FEATURES.md) - Detailed feature list
- [Architecture](ARCHITECTURE.md) - System design and structure

## 🧪 Testing

Import `postman_collection.json` for comprehensive API testing.

## 📋 Requirements

- Go 1.21+
- FFmpeg & FFprobe
- Docker (optional)

## 🏗️ Architecture

```
┌─────────────────────────────────────┐
│         API Layer                   │
│  (Handlers, Middleware, Routes)     │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│      Application Layer              │
│  (Services, Use Cases)              │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│       Core/Domain Layer             │
│  (Business Logic, Interfaces)       │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│    Infrastructure Layer             │
│  (FFmpeg, Filesystem, Cache)        │
└─────────────────────────────────────┘
```

## 🎬 Quality Variants

| Quality | Resolution | Bitrate |
| ------- | ---------- | ------- |
| 240p    | 426×240    | 400k    |
| 360p    | 640×360    | 800k    |
| 480p    | 854×480    | 1200k   |
| 720p    | 1280×720   | 2500k   |
| 1080p   | 1920×1080  | 5000k   |
| 1440p   | 2560×1440  | 8000k   |
| 4K      | 3840×2160  | 15000k  |

## 🤝 Contributing

1. Follow Clean Architecture principles
2. Keep layers separated
3. Write tests for use cases
4. Document public APIs

## 📝 License

MIT License

## 🔧 Tech Stack

- **Go** - High-performance backend
- **FFmpeg** - Video encoding engine
- **Docker** - Containerization
- **Clean Architecture** - Design pattern
