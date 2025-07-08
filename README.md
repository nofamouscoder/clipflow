# ClipFlow - Professional Video Editing Made Simple

ClipFlow is a powerful yet simple video editing platform that lets you merge videos, add YouTube clips, and create professional content without complex software.

## ✨ Features

- **Video Merging**: Combine multiple video files seamlessly
- **YouTube Integration**: Add clips from YouTube videos with custom time segments
- **Audio Mixing**: Add background music with volume control
- **Professional Effects**: Apply slow motion, mute audio, and trim videos
- **Multiple Formats**: Support for MP4, MOV, AVI, WMV, FLV, MKV, WEBM
- **High Quality Output**: Choose from various aspect ratios and frame rates
- **Task History**: Track all your processing tasks with progress monitoring
- **Download Ready**: Get your processed videos ready for download

## 🎨 New Design Features

### Professional Header & Navigation
- **Sticky Header**: Modern glassmorphism design with blur effects
- **Logo**: Custom SVG logo representing video editing and flow
- **Navigation**: Clean navigation with active page indicators
- **Mobile Responsive**: Hamburger menu for mobile devices

### Comprehensive Footer
- **About Section**: Information about ClipFlow
- **Feature Links**: Quick access to key features
- **Support Links**: FAQ, documentation, and contact information
- **Legal Links**: Privacy policy, terms of service, etc.
- **Social Media**: Links to social platforms

### FAQ Section
- **Interactive FAQ**: Expandable questions and answers
- **Comprehensive Coverage**: Covers all major user questions
- **Smooth Animations**: Professional toggle animations
- **Mobile Optimized**: Responsive design for all devices

## 🚀 Getting Started

### Prerequisites
- Go 1.19 or higher
- FFmpeg installed on your system
- yt-dlp for YouTube video downloading

### Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/clipflow.git
cd clipflow
```

2. Install dependencies:
```bash
go mod download
```

3. Set up configuration (optional):
```bash
cp .env.example .env
# Edit .env with your settings
```

4. Run the server:
```bash
go run main.go
```

5. Open your browser and navigate to `http://localhost:8080`

## 📁 Project Structure

```
clipflow/
├── main.go              # Main server file
├── index.html           # Home page with video editor
├── history.html         # Task history page
├── static/              # Static assets
│   └── logo.svg         # ClipFlow logo
├── models/              # Database models
├── auth/                # Authentication
├── middleware/          # HTTP middleware
├── config/              # Configuration
├── utils/               # Utility functions
├── uploads/             # Uploaded files
├── output/              # Processed videos
└── temp/                # Temporary files
```

## 🎯 Usage

### Creating a Video Project

1. **Upload Videos**: Drag and drop video files or click to browse
2. **Add YouTube Clips**: Paste YouTube URLs and select time segments
3. **Add Audio**: Upload background music with volume control
4. **Edit Settings**: Adjust playback speed, trim videos, apply effects
5. **Configure Output**: Choose aspect ratio and frame rate
6. **Generate**: Click "Generate Merged Video" to start processing

### Managing Tasks

- **View History**: Check the History page to see all your tasks
- **Track Progress**: Real-time progress updates for processing tasks
- **Download Results**: Download completed videos when ready
- **Task Details**: View detailed information about each task
- **Delete Tasks**: Remove completed or failed tasks

## 🔧 Configuration

### Environment Variables

Create a `.env` file in the root directory:

```env
# Server Configuration
SERVER_HOST=localhost
SERVER_PORT=8080

# Security
JWT_SECRET=your-secret-key-here

# File Paths
YTDLP_PATH=/usr/local/bin/yt-dlp
UPLOADS_DIR=./uploads
OUTPUT_DIR=./output
TEMP_DIR=./temp

# Database
DATABASE_PATH=./clipflow.db
```

## 🎨 Customization

### Logo
The logo is an SVG file located at `static/logo.svg`. You can customize it by:
- Editing the SVG directly
- Changing colors in the gradient definitions
- Modifying the text or icon elements

### Styling
All styles are included in the HTML files. Key customization areas:
- Color scheme in CSS variables
- Header and footer layouts
- FAQ section styling
- Responsive breakpoints

## 📱 Mobile Support

ClipFlow is fully responsive and works on:
- Desktop browsers (Chrome, Firefox, Safari, Edge)
- Mobile browsers (iOS Safari, Chrome Mobile)
- Tablet devices

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- FFmpeg for video processing
- yt-dlp for YouTube video downloading
- Gin framework for the web server
- SQLite for data storage

## 📞 Support

- **FAQ**: Check the FAQ section on the homepage
- **Issues**: Report bugs on GitHub Issues
- **Documentation**: See inline code comments and this README

---

Made with ❤️ for content creators

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
