# Clipflow API Documentation

## Base URL
```
http://localhost:8080/api
```

## Authentication

The API uses optional JWT authentication. Users can:

1. **Anonymous Access**: Use the API without authentication - the system will automatically create anonymous users
2. **Authenticated Access**: Include a JWT token in the Authorization header for persistent user sessions

**For authenticated requests:**
```
Authorization: Bearer <your-jwt-token>
```

**Getting Started:**
1. Call `GET /api/me` to get or create a user session
2. Store the returned token locally
3. Include the token in subsequent requests

## Endpoints

### Public Endpoints

#### GET /api/me
Get or create a user session. This is the recommended starting point for all applications.

**Headers:** (Optional)
```
Authorization: Bearer <existing-jwt-token>
```

**Response (New User):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "user-id",
    "email": "anonymous_abc12345@clipflow.local",
    "username": "Anonymous_abc12345",
    "created_at": "2023-12-01T10:00:00Z",
    "updated_at": "2023-12-01T10:00:00Z"
  },
  "new": true
}
```

**Response (Existing User):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "user-id",
    "email": "user@example.com",
    "username": "username",
    "created_at": "2023-12-01T10:00:00Z",
    "updated_at": "2023-12-01T10:00:00Z"
  },
  "new": false
}
```

#### POST /api/register
Register a new user account with email and password.

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

### API Endpoints

#### POST /api/generate-video
Create a new video processing task with file uploads.

**Headers:** (Optional)
```
Authorization: Bearer <jwt-token>
```

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

**Headers:** (Optional)
```
Authorization: Bearer <jwt-token>
```

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
Get all tasks for a specific user. Requires userID parameter.

**Query Parameters:**
- `userID` (required) - The user ID to filter tasks by

**Headers:** (Optional)
```
Authorization: Bearer <jwt-token>
```

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

**Headers:** (Optional)
```
Authorization: Bearer <jwt-token>
```

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

### 1. Get or create user session (Recommended starting point)
```bash
curl -X GET http://localhost:8080/api/me
```

### 2. Create video task with file upload (Anonymous)
```bash
curl -X POST http://localhost:8080/api/generate-video \
  -F "data={\"outputsize\":\"16:9\",\"videos\":[{\"file\":\"video_0\",\"options\":{\"slowmotion\":false,\"mute\":false}}]}" \
  -F "video_0=@/path/to/your/video.mp4"
```

### 3. Create video task with file upload (Authenticated)
```bash
curl -X POST http://localhost:8080/api/generate-video \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "data={\"outputsize\":\"16:9\",\"videos\":[{\"file\":\"video_0\",\"options\":{\"slowmotion\":false,\"mute\":false}}]}" \
  -F "video_0=@/path/to/your/video.mp4"
```

### 4. Check task status
```bash
curl -X GET http://localhost:8080/api/task/TASK_ID
```

### 5. Get all user tasks
```bash
curl -X GET "http://localhost:8080/api/tasks?userID=YOUR_USER_ID" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### 6. Register a new user (Optional)
```bash
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "username": "testuser",
    "password": "password123"
  }'
```

### 7. Login (Optional)
```bash
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

## Security Notes

1. **JWT Tokens**: Tokens expire after 24 hours
2. **File Validation**: All uploaded files are validated for type and size
3. **User Isolation**: Authenticated users can only access their own tasks
4. **Anonymous Users**: Anonymous users can access any task (for simplicity)
5. **Input Sanitization**: All inputs are validated and sanitized
6. **CORS**: Configured for development (allows all origins)

## Rate Limiting

Currently, no rate limiting is implemented. Consider implementing rate limiting for production use.

## Database

The application uses SQLite by default. The database file is located at `./database/clipflow.db`.

## Setup & Testing

### Quick Setup
```bash
# Automated setup
chmod +x setup.sh
./setup.sh

# Start server
go run main.go

# Test API
chmod +x test_api.sh
./test_api.sh
```

### Environment Variables

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

### Testing
```bash
# Run all tests
./test_api.sh

# Test specific endpoint
curl http://localhost:8080/api/me

# Test with authentication
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8080/api/tasks
``` 