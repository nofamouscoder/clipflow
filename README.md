# Clipflow

A video processing API for merging videos and YouTube clips with professional editing tools.

## Quick Start

```bash
# Install dependencies
chmod +x setup.sh
./setup.sh

# Start server
go run main.go
```

Visit `http://localhost:8080` to use the web interface.

## API

- `POST /api/upload` - Upload video/audio files
- `POST /api/generate-video` - Create video processing task
- `GET /api/tasks?userID=...` - Get user tasks
- `GET /api/task/:id` - Get task status

See [API.md](API.md) for complete documentation.

## Features

- Upload and merge video files
- YouTube video processing with segments
- Background audio support
- Video effects (slow motion, mute)
- Multiple output formats
- Real-time progress tracking

## Requirements

- Go 1.20+
- FFmpeg
- yt-dlp
- SQLite

kill -9 $(lsof -ti:8080)
