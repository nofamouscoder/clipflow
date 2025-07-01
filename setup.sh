#!/bin/bash

# Clipflow Setup Script
# This script installs all dependencies and sets up the environment

set -e  # Exit on any error

echo "🚀 Starting Clipflow setup..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to detect OS
detect_os() {
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        if command_exists apt-get; then
            echo "ubuntu"
        elif command_exists yum; then
            echo "centos"
        else
            echo "linux"
        fi
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        echo "macos"
    else
        echo "unknown"
    fi
}

OS=$(detect_os)
print_status "Detected OS: $OS"

# Install Go
install_go() {
    print_status "Checking Go installation..."
    if command_exists go; then
        GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        print_success "Go is already installed: $GO_VERSION"
    else
        print_status "Installing Go..."
        case $OS in
            "macos")
                if command_exists brew; then
                    brew install go
                else
                    print_error "Homebrew not found. Please install Homebrew first: https://brew.sh/"
                    exit 1
                fi
                ;;
            "ubuntu")
                sudo apt-get update
                sudo apt-get install -y golang-go
                ;;
            "centos")
                sudo yum install -y golang
                ;;
            *)
                print_error "Please install Go manually from https://golang.org/dl/"
                exit 1
                ;;
        esac
    fi
}

# Install FFmpeg
install_ffmpeg() {
    print_status "Checking FFmpeg installation..."
    if command_exists ffmpeg; then
        FFMPEG_VERSION=$(ffmpeg -version | head -n1 | awk '{print $3}')
        print_success "FFmpeg is already installed: $FFMPEG_VERSION"
    else
        print_status "Installing FFmpeg..."
        case $OS in
            "macos")
                if command_exists brew; then
                    brew install ffmpeg
                else
                    print_error "Homebrew not found. Please install Homebrew first: https://brew.sh/"
                    exit 1
                fi
                ;;
            "ubuntu")
                sudo apt-get update
                sudo apt-get install -y ffmpeg
                ;;
            "centos")
                sudo yum install -y epel-release
                sudo yum install -y ffmpeg
                ;;
            *)
                print_error "Please install FFmpeg manually from https://ffmpeg.org/download.html"
                exit 1
                ;;
        esac
    fi
}

# Install yt-dlp
install_ytdlp() {
    print_status "Checking yt-dlp installation..."
    if command_exists yt-dlp; then
        YTDLP_VERSION=$(yt-dlp --version)
        print_success "yt-dlp is already installed: $YTDLP_VERSION"
    else
        print_status "Installing yt-dlp..."
        case $OS in
            "macos")
                if command_exists brew; then
                    brew install yt-dlp
                else
                    print_error "Homebrew not found. Please install Homebrew first: https://brew.sh/"
                    exit 1
                fi
                ;;
            "ubuntu")
                sudo apt-get update
                sudo apt-get install -y python3-pip
                pip3 install yt-dlp
                ;;
            "centos")
                sudo yum install -y python3-pip
                pip3 install yt-dlp
                ;;
            *)
                print_error "Please install yt-dlp manually: pip install yt-dlp"
                exit 1
                ;;
        esac
    fi
}

# Install Node.js (for potential frontend or database tools)
install_nodejs() {
    print_status "Checking Node.js installation..."
    if command_exists node; then
        NODE_VERSION=$(node --version)
        print_success "Node.js is already installed: $NODE_VERSION"
    else
        print_status "Installing Node.js..."
        case $OS in
            "macos")
                if command_exists brew; then
                    brew install node
                else
                    print_error "Homebrew not found. Please install Homebrew first: https://brew.sh/"
                    exit 1
                fi
                ;;
            "ubuntu")
                curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
                sudo apt-get install -y nodejs
                ;;
            "centos")
                curl -fsSL https://rpm.nodesource.com/setup_18.x | sudo bash -
                sudo yum install -y nodejs
                ;;
            *)
                print_error "Please install Node.js manually from https://nodejs.org/"
                exit 1
                ;;
        esac
    fi
}

# Install SQLite (for local database)
install_sqlite() {
    print_status "Checking SQLite installation..."
    if command_exists sqlite3; then
        SQLITE_VERSION=$(sqlite3 --version | awk '{print $1}')
        print_success "SQLite is already installed: $SQLITE_VERSION"
    else
        print_status "Installing SQLite..."
        case $OS in
            "macos")
                if command_exists brew; then
                    brew install sqlite
                else
                    print_error "Homebrew not found. Please install Homebrew first: https://brew.sh/"
                    exit 1
                fi
                ;;
            "ubuntu")
                sudo apt-get update
                sudo apt-get install -y sqlite3
                ;;
            "centos")
                sudo yum install -y sqlite
                ;;
            *)
                print_error "Please install SQLite manually"
                exit 1
                ;;
        esac
    fi
}

# Create project directories
create_directories() {
    print_status "Creating project directories..."
    
    # Create necessary directories
    mkdir -p temp
    mkdir -p output
    mkdir -p uploads
    mkdir -p database
    mkdir -p logs
    
    # Set permissions
    chmod 755 temp output uploads database logs
    
    print_success "Directories created successfully"
}

# Initialize Go modules
init_go_modules() {
    print_status "Initializing Go modules..."
    
    # Download dependencies
    go mod download
    
    # Verify dependencies
    go mod verify
    
    print_success "Go modules initialized successfully"
}

# Create SQLite database
create_database() {
    print_status "Creating SQLite database..."
    
    # Create database file
    sqlite3 database/clipflow.db << 'EOF'
-- Users table
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Tasks table
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    progress INTEGER DEFAULT 0,
    message TEXT,
    output_file TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Insert sample data (optional)
INSERT OR IGNORE INTO users (id, email, username, password_hash) 
VALUES ('sample-user', 'demo@clipflow.com', 'demo', 'demo-hash');
EOF
    
    print_success "SQLite database created successfully"
}

# Create environment file
create_env_file() {
    print_status "Creating environment configuration..."
    
    cat > .env << 'EOF'
# Clipflow Environment Configuration

# Server Configuration
PORT=8080
HOST=localhost

# Database Configuration
DB_TYPE=sqlite
DB_PATH=./database/clipflow.db

# File Storage
TEMP_DIR=./temp
OUTPUT_DIR=./output
UPLOADS_DIR=./uploads

# Security
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
SESSION_SECRET=your-session-secret-change-this-in-production

# External Services (Optional)
# FIREBASE_PROJECT_ID=your-firebase-project-id
# FIREBASE_PRIVATE_KEY=your-firebase-private-key
# FIREBASE_CLIENT_EMAIL=your-firebase-client-email

# Logging
LOG_LEVEL=info
LOG_FILE=./logs/clipflow.log

# Video Processing
MAX_FILE_SIZE=100MB
ALLOWED_VIDEO_FORMATS=mp4,avi,mov,wmv,flv,webm
ALLOWED_AUDIO_FORMATS=mp3,wav,aac,ogg,flac
EOF
    
    print_success "Environment file created: .env"
    print_warning "Please update the .env file with your specific configuration"
}

# Create systemd service file (Linux only)
create_service_file() {
    if [[ "$OS" == "ubuntu" || "$OS" == "centos" ]]; then
        print_status "Creating systemd service file..."
        
        sudo tee /etc/systemd/system/clipflow.service > /dev/null << 'EOF'
[Unit]
Description=Clipflow Video Processing API
After=network.target

[Service]
Type=simple
User=$USER
WorkingDirectory=$(pwd)
ExecStart=$(which go) run main.go
Restart=always
RestartSec=5
Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target
EOF
        
        print_success "Systemd service file created"
        print_status "To enable the service: sudo systemctl enable clipflow"
        print_status "To start the service: sudo systemctl start clipflow"
    fi
}

# Test installations
test_installations() {
    print_status "Testing installations..."
    
    # Test Go
    if command_exists go; then
        print_success "✓ Go is working"
    else
        print_error "✗ Go is not working"
    fi
    
    # Test FFmpeg
    if command_exists ffmpeg; then
        print_success "✓ FFmpeg is working"
    else
        print_error "✗ FFmpeg is not working"
    fi
    
    # Test yt-dlp
    if command_exists yt-dlp; then
        print_success "✓ yt-dlp is working"
    else
        print_error "✗ yt-dlp is not working"
    fi
    
    # Test SQLite
    if command_exists sqlite3; then
        print_success "✓ SQLite is working"
    else
        print_error "✗ SQLite is not working"
    fi
    
    # Test database
    if sqlite3 database/clipflow.db "SELECT COUNT(*) FROM users;" > /dev/null 2>&1; then
        print_success "✓ Database is working"
    else
        print_error "✗ Database is not working"
    fi
}

# Main setup function
main() {
    print_status "Starting Clipflow setup..."
    
    # Install dependencies
    install_go
    install_ffmpeg
    install_ytdlp
    install_nodejs
    install_sqlite
    
    # Setup project
    create_directories
    init_go_modules
    create_database
    create_env_file
    
    if [[ "$OS" == "ubuntu" || "$OS" == "centos" ]]; then
        create_service_file
    fi
    
    # Test everything
    test_installations
    
    print_success "🎉 Clipflow setup completed successfully!"
    print_status ""
    print_status "Next steps:"
    print_status "1. Update the .env file with your configuration"
    print_status "2. Run: go run main.go"
    print_status "3. Open http://localhost:8080 in your browser"
    print_status ""
    print_status "For production deployment:"
    print_status "1. Change JWT_SECRET and SESSION_SECRET in .env"
    print_status "2. Set GIN_MODE=release"
    print_status "3. Use a reverse proxy (nginx) for SSL"
}

# Run main function
main "$@" 