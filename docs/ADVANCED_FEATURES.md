# Advanced Features Documentation

## Overview

The Video Encoder Service now includes comprehensive advanced features for production-ready video processing, including real-time progress updates, webhook notifications, job scheduling, batch processing, preview generation, content analysis, storage integration, CDN support, and API rate limiting.

---

## 1. WebSocket Progress Updates (Real-Time)

### Overview

Real-time progress updates eliminate the need for polling, providing instant feedback on job status.

### Endpoint

```
ws://localhost:8080/jobs/{jobId}/progress
```

### Connection Example

```javascript
const ws = new WebSocket("ws://localhost:8080/jobs/job_123/progress");

ws.onmessage = (event) => {
  const update = JSON.parse(event.data);
  console.log("Progress:", update);
};
```

### Progress Update Format

```json
{
  "jobId": "job_123",
  "status": "RUNNING",
  "progress": 45,
  "currentStep": "Encoding 720p",
  "eta": "2 minutes remaining",
  "timestamp": "2025-11-25T10:30:00Z",
  "message": "Processing quality 720p"
}
```

### Status Values

- `CONNECTED` - WebSocket connection established
- `QUEUED` - Job is in queue
- `RUNNING` - Job is processing
- `COMPLETED` - Job finished successfully
- `FAILED` - Job failed

### Features

- **Auto-reconnect**: Client should implement reconnection logic
- **Multiple Connections**: Multiple clients can connect to same job
- **Heartbeat**: Connection stays alive automatically
- **Broadcast**: Updates sent to all connected clients

---

## 2. Webhook Notifications (HTTP Callbacks)

### Overview

Webhook notifications allow external systems to receive job completion events via HTTP POST callbacks.

### Configuration in Job Creation

```json
{
  "inputPath": "videos/sample.mp4",
  "webhookUrl": "https://myapp.com/webhook/video-complete",
  "webhookConfig": {
    "url": "https://myapp.com/webhook/video-complete",
    "headers": {
      "Authorization": "Bearer YOUR_TOKEN",
      "X-Custom-Header": "value"
    },
    "secret": "your-webhook-secret",
    "retries": 3
  }
}
```

### Webhook Payload (Job Completed)

```json
{
  "jobId": "job_123",
  "status": "COMPLETED",
  "inputPath": "videos/sample.mp4",
  "outputUrl": "https://cdn.example.com/job_123/master.m3u8",
  "completedAt": "2025-11-25T10:35:00Z",
  "metadata": {
    "processingTimeSec": 125.5,
    "retries": 0,
    "chunkDuration": 10
  },
  "timestamp": "2025-11-25T10:35:00Z"
}
```

### Webhook Payload (Job Failed)

```json
{
  "jobId": "job_123",
  "status": "FAILED",
  "inputPath": "videos/sample.mp4",
  "error": {
    "code": "ERR_FFMPEG_FAILED",
    "stage": "encoding",
    "message": "FFmpeg process failed"
  },
  "metadata": {
    "retries": 3,
    "maxRetries": 3
  },
  "timestamp": "2025-11-25T10:35:00Z"
}
```

### Security

- **HMAC Signature**: Webhook includes `X-Webhook-Signature` header
- **Signature Format**: `sha256=<hex_encoded_hmac>`
- **Verification**: Use the secret to verify the signature

### Retry Logic

- **Max Retries**: 3 attempts (configurable)
- **Backoff**: Exponential (1s, 4s, 9s)
- **Timeout**: 30 seconds per request

---

## 3. Job Scheduling (Delayed Execution)

### Overview

Schedule jobs to run at specific times, useful for off-peak processing or batch operations.

### Endpoint

```
POST /schedule
```

### Request

```json
{
  "inputPath": "videos/sample.mp4",
  "chunkDuration": 10,
  "scheduledAt": "2025-11-26T02:00:00Z"
}
```

### Response

```json
{
  "success": true,
  "message": "Job scheduled successfully",
  "data": {
    "jobId": "job_123",
    "status": "SCHEDULED",
    "scheduledAt": "2025-11-26T02:00:00Z"
  }
}
```

### Features

- **Automatic Execution**: Job starts at scheduled time
- **Immediate Execution**: If scheduled time is in the past
- **Cancellation**: Cancel scheduled jobs before execution
- **Status Tracking**: Jobs show `SCHEDULED` status

### Use Cases

- Off-peak processing (nighttime encoding)
- Batch processing at specific times
- Coordinated multi-job execution
- Resource optimization

---

## 4. Batch Processing (Multiple Files)

### Overview

Process multiple videos in a single request, with unified tracking and status.

### Endpoint

```
POST /batch
```

### Request

```json
{
  "files": ["videos/video1.mp4", "videos/video2.mp4", "videos/video3.mp4"],
  "batchId": "batch_20251125",
  "chunkDuration": 10,
  "webhookUrl": "https://myapp.com/webhook/batch-complete",
  "metadata": {
    "campaign": "summer-2025",
    "priority": "high"
  }
}
```

### Response

```json
{
  "success": true,
  "message": "Batch job created successfully",
  "data": {
    "batchId": "batch_20251125",
    "jobs": [
      {
        "jobId": "job_1",
        "inputPath": "videos/video1.mp4",
        "status": "QUEUED"
      },
      {
        "jobId": "job_2",
        "inputPath": "videos/video2.mp4",
        "status": "QUEUED"
      },
      { "jobId": "job_3", "inputPath": "videos/video3.mp4", "status": "QUEUED" }
    ],
    "status": "PENDING"
  }
}
```

### Get Batch Status

```
GET /batch/{batchId}
```

### Batch Status Response

```json
{
  "success": true,
  "data": {
    "batchId": "batch_20251125",
    "status": "PROCESSING",
    "createdAt": "2025-11-25T10:00:00Z",
    "stats": {
      "total": 3,
      "pending": 0,
      "running": 2,
      "completed": 1,
      "failed": 0
    },
    "jobs": [...]
  }
}
```

### Batch Status Values

- `PENDING` - No jobs started yet
- `PROCESSING` - Some jobs running
- `COMPLETED` - All jobs completed successfully
- `FAILED` - All jobs failed
- `PARTIAL` - Some succeeded, some failed

---

## 5. Output Format Options

### Overview

Support for multiple output formats beyond HLS.

### Supported Formats

- **HLS** (`.m3u8` + `.ts` chunks) - Default
- **MP4** (Single file per quality) - Coming soon
- **WebM** (Single file per quality) - Coming soon

### Request with Format Options

```json
{
  "inputPath": "videos/sample.mp4",
  "outputFormats": ["hls", "mp4", "webm"],
  "qualities": ["720p", "1080p"]
}
```

### Current Implementation

- ✅ HLS format fully implemented
- ⚠️ MP4 format - placeholder
- ⚠️ WebM format - placeholder

---

## 6. Preview Generation (Thumbnails + GIF)

### Overview

Generate preview images and animated GIFs for video players and hover previews.

### Request with Preview Config

```json
{
  "inputPath": "videos/sample.mp4",
  "previewConfig": {
    "generate": true,
    "thumbnailCount": 10,
    "gifDuration": 3,
    "gifStartTime": 5
  }
}
```

### Preview Output

```json
{
  "thumbnails": [
    "outputs/job_123/previews/thumb_1.jpg",
    "outputs/job_123/previews/thumb_2.jpg",
    "outputs/job_123/previews/thumb_3.jpg"
  ],
  "gif": "outputs/job_123/previews/preview.gif"
}
```

### Features

- **Multiple Thumbnails**: Extract frames at intervals
- **Animated GIF**: 3-second preview with palette optimization
- **Custom Sizing**: Configurable width (height auto-calculated)
- **Quality Control**: JPEG quality level 2 (high quality)

### Use Cases

- Video player thumbnails
- Hover previews
- Social media previews
- Video gallery displays

---

## 7. Enhanced Content Analysis

### Overview

Comprehensive video metadata extraction including codec info, bitrate, frame rate, and audio details.

### Endpoint

```
POST /analyze
```

### Request

```json
{
  "inputPath": "videos/sample.mp4"
}
```

### Response

```json
{
  "width": 1920,
  "height": 1080,
  "duration": 120.5,
  "videoCodec": "h264",
  "audioCodec": "aac",
  "bitrate": "5000kbps",
  "videoBitrate": "4500kbps",
  "frameRate": "30/1",
  "hasAudio": true,
  "audioChannels": 2,
  "audioSampleRate": "48000",
  "fileSize": "75000000"
}
```

### Metadata Fields

- **Video**: Codec, bitrate, frame rate, resolution
- **Audio**: Codec, channels, sample rate
- **Format**: Duration, total bitrate, file size

---

## 8. Storage Integration (Cloud Storage)

### Overview

Upload encoded outputs directly to cloud storage (S3, GCS, Azure).

### Configuration

```json
{
  "inputPath": "videos/sample.mp4",
  "storageConfig": {
    "type": "s3",
    "bucket": "my-videos",
    "region": "us-east-1",
    "path": "encoded/"
  }
}
```

### Supported Storage Types

- **local** - Local filesystem (default)
- **s3** - AWS S3 (placeholder)
- **gcs** - Google Cloud Storage (placeholder)
- **azure** - Azure Blob Storage (placeholder)

### Current Implementation

- ✅ Local storage fully implemented
- ⚠️ S3 - adapter structure ready, SDK integration pending
- ⚠️ GCS - adapter structure ready, SDK integration pending
- ⚠️ Azure - adapter structure ready, SDK integration pending

### Storage Adapter Interface

```go
type StorageAdapter interface {
    Upload(localPath string, remotePath string) (string, error)
    Download(remotePath string, localPath string) error
    Delete(remotePath string) error
    GetURL(remotePath string) (string, error)
    List(prefix string) ([]string, error)
}
```

---

## 9. CDN Integration

### Overview

Push encoded videos to CDN for optimized global delivery.

### Configuration

```json
{
  "inputPath": "videos/sample.mp4",
  "cdnConfig": {
    "enabled": true,
    "provider": "cloudflare",
    "baseUrl": "https://cdn.example.com"
  }
}
```

### CDN Response

```json
{
  "cdnUrls": {
    "master": "https://cdn.example.com/videos/job_123/master.m3u8",
    "720p": "https://cdn.example.com/videos/job_123/720p/index.m3u8",
    "1080p": "https://cdn.example.com/videos/job_123/1080p/index.m3u8"
  }
}
```

### Supported CDN Providers

- **cloudflare** - Cloudflare R2/Workers
- **cloudfront** - AWS CloudFront
- **generic** - Generic CDN (default)

### Features

- **Automatic Upload**: Push files after encoding
- **Cache Invalidation**: Clear CDN cache on updates
- **URL Generation**: Generate CDN URLs for all outputs

### Current Implementation

- ✅ CDN service structure implemented
- ⚠️ Provider integrations - placeholder implementations

---

## 10. API Rate Limiting

### Overview

Protect API from abuse with token bucket rate limiting.

### Configuration

- **Rate**: 100 requests per minute (default)
- **Burst**: 20 requests (default)
- **Scope**: Per IP address

### Rate Limit Response

```http
HTTP/1.1 429 Too Many Requests
Content-Type: text/plain

Rate limit exceeded. Please try again later.
```

### Features

- **Token Bucket Algorithm**: Smooth rate limiting
- **Per-IP Tracking**: Individual limits per client
- **Automatic Cleanup**: Remove stale visitors
- **Configurable**: Adjust rate and burst per endpoint

### Endpoint-Specific Limits

```go
rateLimiter := middleware.NewEndpointRateLimiter()
rateLimiter.AddEndpoint("/transcode", 10, 5)  // 10/min, burst 5
rateLimiter.AddEndpoint("/batch", 5, 2)       // 5/min, burst 2
```

### Headers

- `X-Forwarded-For` - Client IP (proxy)
- `X-Real-IP` - Client IP (direct)
- `RemoteAddr` - Fallback IP

---

## Architecture Summary

### New Services

1. **WebhookService** - HTTP callback notifications
2. **SchedulerService** - Delayed job execution
3. **BatchService** - Multi-file processing
4. **CDNService** - CDN integration
5. **MetricsService** - Resource monitoring (from previous session)
6. **DLQService** - Dead-letter queue (from previous session)

### New Infrastructure

1. **WebSocket Manager** - Real-time connections
2. **Preview Generator** - Thumbnails and GIFs
3. **Storage Adapters** - Cloud storage abstraction
4. **Rate Limiter** - API protection
5. **Enhanced Probe** - Detailed metadata extraction

### New Endpoints

```
GET  /health                    - Health check
POST /transcode                 - Create job
POST /schedule                  - Schedule job
POST /batch                     - Create batch
GET  /batch/{id}                - Get batch status
POST /analyze                   - Analyze video
GET  /jobs                      - List jobs
GET  /jobs/{id}                 - Get job
GET  /jobs/{id}/progress        - WebSocket progress
DELETE /jobs/{id}               - Delete job
POST /jobs/{id}/resume          - Resume job
```

---

## Feature Implementation Status

| Feature                   | Status      | Implementation % | Notes                         |
| ------------------------- | ----------- | ---------------- | ----------------------------- |
| **WebSocket Progress**    | ✅ Complete | 100%             | Real-time updates working     |
| **Webhook Notifications** | ✅ Complete | 100%             | With retry and HMAC           |
| **Job Scheduling**        | ✅ Complete | 100%             | Timer-based execution         |
| **Batch Processing**      | ✅ Complete | 100%             | Multi-file support            |
| **Output Format Options** | ⚠️ Partial  | 30%              | HLS only, MP4/WebM pending    |
| **Preview Generation**    | ✅ Complete | 100%             | Thumbnails + GIF              |
| **Content Analysis**      | ✅ Complete | 100%             | Detailed metadata             |
| **Storage Integration**   | ⚠️ Partial  | 40%              | Structure ready, SDKs pending |
| **CDN Integration**       | ⚠️ Partial  | 40%              | Structure ready, APIs pending |
| **API Rate Limiting**     | ✅ Complete | 100%             | Token bucket algorithm        |

---

## Configuration

### Environment Variables

```env
# Server
PORT=8080
SHUTDOWN_TIMEOUT=30

# Workers
MAX_WORKERS=2
MAX_RETRIES=3

# Storage
OUTPUTS_DIR=outputs
LOGS_DIR=logs
CACHE_DIR=cache

# Rate Limiting
RATE_LIMIT_PER_MINUTE=100
RATE_LIMIT_BURST=20
```

---

## Next Steps

### High Priority

1. Implement MP4 output format
2. Implement WebM output format
3. Complete S3 storage adapter
4. Complete CloudFront CDN integration

### Medium Priority

5. Add GCS storage adapter
6. Add Azure storage adapter
7. Add Cloudflare CDN integration
8. Enhance preview generation options

### Low Priority

9. Add API key authentication
10. Add user-based rate limiting
11. Add webhook retry dashboard
12. Add batch operation analytics

---

## Testing

### WebSocket Test

```bash
# Using wscat
wscat -c ws://localhost:8080/jobs/job_123/progress
```

### Webhook Test

```bash
# Create job with webhook
curl -X POST http://localhost:8080/transcode \
  -H "Content-Type: application/json" \
  -d '{
    "inputPath": "videos/sample.mp4",
    "webhookUrl": "https://webhook.site/your-unique-url"
  }'
```

### Batch Test

```bash
curl -X POST http://localhost:8080/batch \
  -H "Content-Type: application/json" \
  -d '{
    "files": ["videos/video1.mp4", "videos/video2.mp4"],
    "batchId": "test_batch"
  }'
```

### Schedule Test

```bash
curl -X POST http://localhost:8080/schedule \
  -H "Content-Type: application/json" \
  -d '{
    "inputPath": "videos/sample.mp4",
    "scheduledAt": "2025-11-26T02:00:00Z"
  }'
```

---

## Production Checklist

- [ ] Configure rate limits for production load
- [ ] Set up webhook endpoint monitoring
- [ ] Implement webhook signature verification
- [ ] Configure CDN provider credentials
- [ ] Set up cloud storage credentials
- [ ] Test WebSocket reconnection logic
- [ ] Monitor batch processing performance
- [ ] Set up scheduled job monitoring
- [ ] Configure preview generation quality
- [ ] Test rate limiting under load

---

## Support

For issues or questions:

- Check logs in `logs/` directory
- Review job state in `outputs/{jobId}/state.json`
- Monitor WebSocket connections
- Check webhook delivery logs
- Review batch statistics

---

**Version**: 2.0.0  
**Last Updated**: November 25, 2025  
**Status**: Production Ready (with noted limitations)
