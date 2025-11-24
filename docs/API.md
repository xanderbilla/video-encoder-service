# API Documentation

## Base URL

```
http://localhost:8080
```

## Response Format

All API responses follow this structure:

```json
{
  "success": true,
  "status": 200,
  "timestamp": "2025-11-24T19:00:00Z",
  "requestId": "uuid-string",
  "message": "Optional message",
  "data": {},
  "error": null
}
```

### Error Response

```json
{
  "success": false,
  "status": 400,
  "timestamp": "2025-11-24T19:00:00Z",
  "requestId": "uuid-string",
  "error": {
    "code": "ERR_CODE",
    "message": "Error description",
    "details": {},
    "timestamp": "2025-11-24T19:00:00Z"
  }
}
```

## Endpoints

| Method | Endpoint     | Description            |
| ------ | ------------ | ---------------------- |
| GET    | `/health`    | Service health check   |
| POST   | `/transcode` | Create transcoding job |
| GET    | `/jobs`      | List all jobs          |
| GET    | `/jobs/{id}` | Get job details        |
| DELETE | `/jobs/{id}` | Delete job             |

## 1. Health Check

Check service health and get system statistics.

**Endpoint:** `GET /health`

**Response:**

```json
{
  "success": true,
  "status": 200,
  "timestamp": "2025-11-24T19:00:00Z",
  "requestId": "uuid",
  "message": "Service is healthy and running",
  "data": {
    "status": "ok",
    "service": "video-encoder",
    "version": "2.0.0",
    "uptime": "1h30m45s",
    "workers": {
      "active": 1,
      "max": 2
    },
    "queue": {
      "size": 0,
      "capacity": 100
    },
    "jobs": {
      "total": 10,
      "queued": 0,
      "running": 1,
      "completed": 8,
      "failed": 1
    },
    "disk": {
      "usagePercent": 45.5,
      "totalGB": 1000.0,
      "usedGB": 455.0,
      "freeGB": 545.0
    }
  }
}
```

## 2. Create Transcoding Job

Submit a video for multi-quality HLS transcoding.

**Endpoint:** `POST /transcode`

**Request Body:**

```json
{
  "inputPath": "videos/sample.mp4",
  "chunkDuration": 4
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| inputPath | string | Yes | Path to input video file |
| chunkDuration | integer | No | HLS chunk duration in seconds (default: 4) |

**Response:**

```json
{
  "success": true,
  "status": 200,
  "timestamp": "2025-11-24T19:00:00Z",
  "requestId": "uuid",
  "message": "Transcoding job created and queued successfully",
  "data": {
    "jobId": "job_1234567890",
    "status": "QUEUED"
  }
}
```

**Error Responses:**

| Status | Code                   | Description                   |
| ------ | ---------------------- | ----------------------------- |
| 400    | ERR_MISSING_FIELD      | inputPath is required         |
| 400    | ERR_FILE_NOT_FOUND     | Input file not found          |
| 400    | ERR_FILE_TOO_LARGE     | File exceeds size limit       |
| 400    | ERR_UNSUPPORTED_FORMAT | File format not supported     |
| 400    | ERR_CORRUPTED_FILE     | Video file is corrupted       |
| 503    | ERR_DISK_FULL          | Disk space threshold exceeded |

## 3. Get Job Details

Retrieve detailed information about a specific job.

**Endpoint:** `GET /jobs/{id}`

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | string | Job ID |

**Response:**

```json
{
  "success": true,
  "status": 200,
  "timestamp": "2025-11-24T19:00:00Z",
  "requestId": "uuid",
  "data": {
    "jobId": "job_1234567890",
    "status": "COMPLETED",
    "inputPath": "videos/sample.mp4",
    "inputHash": "sha256-hash",
    "chunkDuration": 4,
    "input": {
      "fileName": "sample.mp4",
      "sourcePath": "videos/sample.mp4",
      "resolution": "1920x1080",
      "duration": 11.98,
      "generatedQualities": [
        {
          "quality": "240p",
          "resolution": "426x240",
          "bitrate": "400k"
        },
        {
          "quality": "360p",
          "resolution": "640x360",
          "bitrate": "800k"
        },
        {
          "quality": "480p",
          "resolution": "854x480",
          "bitrate": "1200k"
        },
        {
          "quality": "720p",
          "resolution": "1280x720",
          "bitrate": "2500k"
        },
        {
          "quality": "1080p",
          "resolution": "1920x1080",
          "bitrate": "5000k"
        }
      ]
    },
    "output": {
      "outputDir": "outputs/job_1234567890/hls",
      "masterPlaylist": "outputs/job_1234567890/hls/master.m3u8",
      "qualities": [
        {
          "quality": "720p",
          "playlist": "outputs/job_1234567890/hls/720p/main.m3u8",
          "chunksDir": "outputs/job_1234567890/hls/720p",
          "chunkCount": 3
        }
      ]
    },
    "progress": {
      "percentage": 100,
      "currentStep": "Completed"
    },
    "meta": {
      "createdAt": "2025-11-24T19:00:00Z",
      "completedAt": "2025-11-24T19:00:35Z",
      "processingTimeSec": 35.4
    },
    "maxRetries": 3,
    "message": "Job created and queued for processing"
  }
}
```

**Job Status Values:**

- `QUEUED` - Job is waiting to be processed
- `RUNNING` - Job is currently being processed
- `COMPLETED` - Job completed successfully (all qualities)
- `PARTIAL_SUCCESS` - Job completed but some qualities failed
- `FAILED` - Job failed after retries
- `INTERRUPTED` - Job was interrupted

## 4. List All Jobs

Get a list of all transcoding jobs.

**Endpoint:** `GET /jobs`

**Response:**

```json
{
  "success": true,
  "status": 200,
  "timestamp": "2025-11-24T19:00:00Z",
  "requestId": "uuid",
  "data": {
    "jobs": [
      {
        "jobId": "job_1234567890",
        "status": "COMPLETED",
        "inputPath": "videos/sample.mp4",
        "meta": {
          "createdAt": "2025-11-24T19:00:00Z",
          "completedAt": "2025-11-24T19:00:35Z"
        }
      }
    ],
    "count": 1
  }
}
```

## 5. Delete Job

Delete a job and all associated output files.

**Endpoint:** `DELETE /jobs/{id}`

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | string | Job ID |

**Response:**

```json
{
  "success": true,
  "status": 200,
  "timestamp": "2025-11-24T19:00:00Z",
  "requestId": "uuid",
  "message": "Job and all associated files deleted successfully",
  "data": {
    "jobId": "job_1234567890",
    "deleted": true
  }
}
```

## Error Codes

| Code                    | Description                             |
| ----------------------- | --------------------------------------- |
| ERR_INVALID_REQUEST     | Malformed JSON or invalid request       |
| ERR_MISSING_FIELD       | Required field is missing               |
| ERR_FILE_NOT_FOUND      | Input file not found                    |
| ERR_FILE_TOO_LARGE      | File exceeds maximum size (1GB)         |
| ERR_UNSUPPORTED_FORMAT  | File format not supported               |
| ERR_CORRUPTED_FILE      | Video file is corrupted                 |
| ERR_NO_VIDEO_STREAM     | File has no valid video stream          |
| ERR_INVALID_VIDEO       | Failed to analyze video                 |
| ERR_RESOLUTION_TOO_HIGH | Video resolution exceeds 4K (3840×2160) |
| ERR_DURATION_TOO_SHORT  | Video duration less than 0.1 seconds    |
| ERR_JOB_NOT_FOUND       | Job ID does not exist                   |
| ERR_ENCODING_FAILED     | Video encoding failed                   |
| ERR_FFMPEG_FAILED       | FFmpeg process failed                   |
| ERR_DISK_FULL           | Disk space threshold exceeded (80%)     |
| ERR_JOB_SIZE_EXCEEDED   | Job size exceeds 1.5GB limit            |
| ERR_DELETE_FAILED       | Failed to delete files                  |
| ERR_INTERNAL_ERROR      | Internal server error                   |

## Rate Limits

Currently no rate limits are enforced. This may change in future versions.

## Authentication

Currently no authentication is required. This may change in future versions.
