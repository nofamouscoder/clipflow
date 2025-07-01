# Clipflow API Documentation

## Base URL
```
http://localhost:8080/api
```

## Authentication

All protected endpoints require a JWT token in the Authorization header:
```
Authorization: Bearer <your-jwt-token>
```

## Endpoints

### Public Endpoints

#### POST /api/register
Register a new user account.

**Request Body:**
```json
{
  "email": "user@example.com",
  "username": "username",
  "password": "password123"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "user-id",
    "email": "user@example.com",
    "username": "username",
    "created_at": "2023-12-01T10:00:00Z",
    "updated_at": "2023-12-01T10:00:00Z"
  }
}
```

#### POST /api/login
Login with existing credentials.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "user-id",
    "email": "user@example.com",
    "username": "username",
    "created_at": "2023-12-01T10:00:00Z",
    "updated_at": "2023-12-01T10:00:00Z"
  }
}
```

### Protected Endpoints

#### POST /api/generate-video
Create a new video processing task with file uploads.

**Content-Type:** `multipart/form-data`

**Form Fields:**
- `data` (JSON string): Video processing configuration
- `video_0`, `video_1`, etc.: Uploaded video files
- `audio_0`, `audio_1`, etc.: Uploaded audio files

**JSON Data Structure:**
```json
{
  "outputsize": "16:9",
  "videos": [
    {
      "file": "video_0",
      "options": {
        "slowmotion": false,
        "index": 0,
        "mute": false,
        "startTime": "0:00",
        "endTime": "1:30"
      }
    }
  ],
  "youtube": [
    {
      "url": "https://youtube.com/watch?v=VIDEO_ID",
      "quality": "1080p",
      "segments": [
        {
          "index": 0,
          "timeline": {
            "start": "0:10",
            "end": "0:20"
          },
          "options": {
            "slowmotion": true,
            "mute": false
          }
        }
      ]
    }
  ],
  "audio": [
    {
      "file": "audio_0"
    }
  ]
}
```

**Response:**
```json
{
  "taskId": "task-uuid",
  "status": "pending",
  "message": "Video generation task created successfully"
}
```

#### GET /api/task/:taskId
Get the status of a specific task.

**Response:**
```json
{
  "id": "task-uuid",
  "user_id": "user-id",
  "status": "completed",
  "progress": 100,
  "message": "Video processing completed successfully",
  "output_file": "/output/merged_user-id_task-uuid.mp4",
  "created_at": "2023-12-01T10:00:00Z",
  "completed_at": "2023-12-01T10:05:00Z"
}
```

#### GET /api/tasks
Get all tasks for the authenticated user.

**Response:**
```json
[
  {
    "id": "task-uuid-1",
    "user_id": "user-id",
    "status": "completed",
    "progress": 100,
    "message": "Video processing completed successfully",
    "output_file": "/output/merged_user-id_task-uuid-1.mp4",
    "created_at": "2023-12-01T10:00:00Z",
    "completed_at": "2023-12-01T10:05:00Z"
  },
  {
    "id": "task-uuid-2",
    "user_id": "user-id",
    "status": "processing",
    "progress": 50,
    "message": "Merging videos",
    "created_at": "2023-12-01T11:00:00Z"
  }
]
```

#### DELETE /api/task/:taskId
Delete a specific task.

**Response:**
```json
{
  "message": "Task deleted successfully"
}
```

## File Upload Guidelines

### Supported Video Formats
- MP4 (.mp4)
- AVI (.avi)
- MOV (.mov)
- WMV (.wmv)
- FLV (.flv)
- WebM (.webm)

### Supported Audio Formats
- MP3 (.mp3)
- WAV (.wav)
- AAC (.aac)
- OGG (.ogg)
- FLAC (.flac)

### File Size Limits
- Maximum file size: 100MB per file
- Maximum form data: 32MB

### File Naming Convention
Files are automatically renamed to prevent conflicts:
```
YYYYMMDD_HHMMSS_XXXXXXXX.ext
```

## Output Formats

### Supported Aspect Ratios
- `16:9` - 1920x1080 (Landscape)
- `9:16` - 1080x1920 (Portrait)
- `1:1` - 1080x1080 (Square)
- `4:3` - 1440x1080 (Classic)
- `3:4` - 1080x1440 (Portrait Classic)

## Error Responses

### Authentication Errors
```json
{
  "error": "Authorization header required"
}
```

```json
{
  "error": "Invalid or expired token"
}
```

### Validation Errors
```json
{
  "error": "File size exceeds maximum allowed size of 104857600 bytes"
}
```

```json
{
  "error": "File format not allowed. Allowed formats: [.mp4, .avi, .mov, .wmv, .flv, .webm]"
}
```

### Processing Errors
```json
{
  "error": "Failed to download YouTube video: invalid time format, expected MM:SS"
}
```

## Task Status Values

- `pending` - Task created, waiting to start
- `processing` - Task is currently running
- `completed` - Task finished successfully
- `failed` - Task failed with error

## Example Usage

### 1. Register a new user
```bash
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "username": "testuser",
    "password": "password123"
  }'
```

### 2. Login
```bash
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

### 3. Create video task with file upload
```bash
curl -X POST http://localhost:8080/api/generate-video \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "data={\"outputsize\":\"16:9\",\"videos\":[{\"file\":\"video_0\",\"options\":{\"slowmotion\":false,\"mute\":false}}]}" \
  -F "video_0=@/path/to/your/video.mp4"
```

### 4. Check task status
```bash
curl -X GET http://localhost:8080/api/task/TASK_ID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### 5. Get all user tasks
```bash
curl -X GET http://localhost:8080/api/tasks \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## Security Notes

1. **JWT Tokens**: Tokens expire after 24 hours
2. **File Validation**: All uploaded files are validated for type and size
3. **User Isolation**: Users can only access their own tasks
4. **Input Sanitization**: All inputs are validated and sanitized
5. **CORS**: Configured for development (allows all origins)

## Rate Limiting

Currently, no rate limiting is implemented. Consider implementing rate limiting for production use.

## Database

The application uses SQLite by default. The database file is located at `./database/clipflow.db`.

## Environment Variables

Create a `.env` file with the following variables:

```env
PORT=8080
HOST=localhost
DB_TYPE=sqlite
DB_PATH=./database/clipflow.db
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
SESSION_SECRET=your-session-secret-change-this-in-production
TEMP_DIR=./temp
OUTPUT_DIR=./output
UPLOADS_DIR=./uploads
MAX_FILE_SIZE=104857600
``` 