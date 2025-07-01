# Clipflow

A comprehensive Go-based API for professional video editing, merging, and processing with support for YouTube clips, local videos, and audio tracks.

## 🚀 Features

### Core Video Processing
- **YouTube Integration**: Download and process specific segments from YouTube videos with precise timing
- **Local Video Support**: Process uploaded videos with trimming, effects, and format conversion
- **Audio Track Support**: Support for multiple audio formats with volume control, fade effects
- **Multi-format Output**: Generate videos in various aspect ratios optimized for different platforms
- **Real-time Progress Tracking**: Task-based async processing with detailed progress updates

### Advanced Editing Capabilities
- **Segment-based Editing**: Process multiple YouTube segments from the same video independently
- **Video Effects**: Slow motion, mute, volume adjustment, fade in/out
- **Smart Scaling**: Automatic video scaling and padding to maintain quality across aspect ratios
- **Batch Processing**: Handle multiple videos and audio tracks in a single merge operation
- **User Management**: Isolated processing and file management per user

## 📋 Prerequisites

### Required Software
- **Go 1.21 or higher** - [Download Go](https://golang.org/dl/)
- **FFmpeg** - Required for video processing
- **yt-dlp** - Required for YouTube video downloads
- **Node.js 18+** - For running the full application stack

### Installation Instructions

#### macOS
```bash
# Install using Homebrew
brew install go ffmpeg yt-dlp node

# Verify installations
go version
ffmpeg -version
yt-dlp --version
node --version
```

#### Ubuntu/Debian
```bash
# Update package list
sudo apt update

# Install Go
sudo apt install golang-go

# Install FFmpeg and yt-dlp
sudo apt install ffmpeg python3-pip
pip3 install yt-dlp

# Install Node.js
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt-get install -y nodejs

# Verify installations
go version
ffmpeg -version
yt-dlp --version
node --version
```

#### Windows
```bash
# Install Go from https://golang.org/dl/
# Install FFmpeg from https://ffmpeg.org/download.html
# Install yt-dlp
pip install yt-dlp

# Install Node.js from https://nodejs.org/
```

## 🛠️ Installation & Setup

### 1. Clone and Setup Backend
```bash
git clone <repository-url>
cd backend

# Install Go dependencies
go mod download

# Create required directories
mkdir -p temp output uploads

# Set up environment (optional)
cp .env.example .env  # if available
```

### 2. Frontend Setup (Optional)
```bash
cd ../frontend
npm install
npm run build  # for production
```

### 3. Database Setup (Optional)
```bash
cd ../backend

# Install Node.js dependencies for database
npm install

# Run database migrations
npx sequelize-cli db:migrate

# Seed initial data
npx sequelize-cli db:seed:all
```

## 🚀 Running the Application

### Development Mode
```bash
# Start the Go API server
go run main.go

# In another terminal, start frontend (optional)
cd ../frontend
npm run dev
```

### Production Mode
```bash
# Build the Go binary
go build -o clipflow main.go

# Run the binary
./clipflow

# The API will start on http://localhost:8080
```

### Verify Installation
```bash
# Test API health
curl http://localhost:8080/api/tasks/test-user

# Expected response: []
```

### Video Input Formats
- **Primary**: MP4, MOV, AVI
- **Additional**: WMV, FLV, MKV, WEBM
- **Max File Size**: 500MB per file
- **Resolution**: Up to 4K (3840x2160)

### Audio Input Formats
- **Primary**: MP3, WAV, AAC
- **Additional**: M4A, OGG, FLAC
- **Max File Size**: 100MB per file
- **Bitrate**: Up to 320kbps

### YouTube Quality Options
- `2160p` - 4K (if available)
- `1440p` - 2K (if available)
- `1080p` - Full HD (recommended)
- `720p` - HD
- `480p` - Standard definition
- `360p` - Low quality (faster processing)

### Playback Speed Options
- `0.1x` - Extreme slow motion
- `0.25x` - Quarter speed
- `0.5x` - Half speed
- `1.0x` - Normal speed (default)
- `2.0x` - Double speed

## 📊 Task Status Lifecycle

1. **pending** - Task created, queued for processing
2. **processing** - Active video processing with progress updates
3. **completed** - Successfully processed, file ready for download
4. **failed** - Processing failed, error details available

### Progress Tracking
Progress is reported as percentage (0-100) with detailed messages:
- 0-20%: Downloading YouTube videos
- 20-40%: Processing video effects
- 40-60%: Processing audio tracks
- 60-80%: Merging videos and audio
- 80-100%: Finalizing output and cleanup

## 🔧 Configuration

### Environment Variables
Create a `.env` file in the backend directory:

```env
# Server Configuration
PORT=8080
HOST=localhost

# Directories
WORK_DIR=./temp
OUTPUT_DIR=./output
UPLOAD_DIR=./uploads

# Processing Options
MAX_FILE_SIZE=500MB
MAX_AUDIO_SIZE=100MB
CLEANUP_INTERVAL=3600  # seconds

# YouTube Processing
YOUTUBE_QUALITY=1080p
MAX_DOWNLOAD_RETRIES=3

# Database (if using Node.js components)
DB_HOST=localhost
DB_PORT=5432
DB_NAME=storicut
DB_USER=postgres
DB_PASS=password
```

### Docker Setup (Optional)
```dockerfile
# Dockerfile example
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN go build -o clipflow main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates ffmpeg python3 py3-pip
RUN pip3 install yt-dlp
WORKDIR /root/
COPY --from=builder /app/clipflow .
EXPOSE 8080
CMD ["./clipflow"]
```

### Contributing
1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Update documentation
5. Submit a pull request

---

**Clipflow Backend API** - Professional video editing and merging platform 
