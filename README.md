# Clipflow

A powerful video processing tool that combines local video files and/or YouTube clips to create professional video content. Built with Go, featuring a web interface, automatic user management, and real-time video processing.

## 🚀 Features

### Video Processing
- **Local Video Upload**: Support for multiple video formats (MP4, AVI, MOV, WMV, FLV, WebM)
- **YouTube Integration**: Download and process specific segments from YouTube videos
- **Audio Support**: Upload and merge audio tracks (MP3, WAV, AAC, OGG, FLAC)
- **Video Effects**: Apply slow motion, mute, and other effects
- **Multiple Output Formats**: 16:9, 9:16, 1:1, 4:3, 3:4 aspect ratios

### User Experience
- **Web Interface**: Modern, responsive web UI for easy video processing
- **Zero-Friction Access**: Start using immediately without registration
- **Automatic User Creation**: Anonymous users created automatically
- **Optional Authentication**: Register for persistent sessions
- **Real-time Progress**: Track video processing status in real-time
- **File Management**: Automatic cleanup and secure storage

### Technical Features
- **RESTful API**: Clean, well-documented endpoints for integration
- **Database Persistence**: SQLite for user and task management
- **File Validation**: Type and size validation for uploads
- **Error Handling**: Comprehensive error reporting
- **CORS Support**: Frontend integration ready

## 🛠️ Prerequisites

- **Go 1.20+** - [Download Go](https://golang.org/dl/)
- **FFmpeg** - Video processing engine
- **yt-dlp** - YouTube video downloader
- **SQLite3** - Database (included in most systems)

## ⚡ Quick Start

### Automated Setup
```bash
# Clone the repository
git clone <repository-url>
cd clipflow

# Run automated setup
chmod +x setup.sh
./setup.sh

# Start the server
go run main.go
```

### Manual Setup
```bash
# Install dependencies
go mod tidy

# Create directories
mkdir -p temp output uploads database logs

# Start the server
go run main.go
```

### Verify Installation
```bash
# Test the API
chmod +x test_api.sh
./test_api.sh
```

## 🌐 Using Clipflow

### Web Interface
1. **Start the server**: `go run main.go`
2. **Open browser**: Navigate to http://localhost:8080
3. **Upload videos**: Drag and drop video files or select from your computer
4. **Add YouTube clips**: Paste YouTube URLs and select segments
5. **Configure settings**: Choose output format and effects
6. **Process videos**: Click generate and watch real-time progress
7. **Download results**: Get your processed video when complete

### API Integration
For developers who want to integrate Clipflow into their applications:
- See [API.md](API.md) for complete API documentation
- See [FRONTEND_INTEGRATION.md](FRONTEND_INTEGRATION.md) for integration examples

## 🔧 Configuration

Create a `.env` file for custom configuration:

```env
PORT=8080
HOST=localhost
JWT_SECRET=your-secret-key
DB_PATH=./database/clipflow.db
MAX_FILE_SIZE=104857600
# Path to yt-dlp executable (default: yt-dlp)
YTDLP_PATH=yt-dlp
```

## 🧪 Testing

```bash
# Run API tests
./test_api.sh

# Test web interface
open http://localhost:8080

# Test specific functionality
curl http://localhost:8080/api/me
```

## 🔒 Security

- JWT-based authentication
- File type and size validation
- Input sanitization
- User data isolation
- Secure file handling

## 📊 Database

SQLite database with automatic schema creation:
- User management
- Task tracking
- Progress monitoring
- File associations

## 🚨 Production Deployment

1. **Update Configuration**
   ```bash
   # Change default secrets
   JWT_SECRET=your-production-secret
   GIN_MODE=release
   ```

2. **Build Binary**
   ```bash
   go build -o clipflow main.go
   ```

3. **Setup Reverse Proxy**
   ```nginx
   # Nginx configuration
   server {
       listen 80;
       server_name your-domain.com;
       
       location / {
           proxy_pass http://localhost:8080;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
       }
   }
   ```

4. **Systemd Service**
   ```bash
   sudo systemctl enable clipflow
   sudo systemctl start clipflow
   ```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📄 License

MIT License - see LICENSE file for details

## 🆘 Support

- **Issues**: Create an issue on GitHub
- **Documentation**: Check API.md and FRONTEND_INTEGRATION.md
- **Testing**: Run test_api.sh for troubleshooting
- **Web Interface**: Open http://localhost:8080 for the main tool

---

**Clipflow** - Professional video processing made simple.
