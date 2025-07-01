#!/bin/bash

# Clipflow API Test Script
# Tests all API endpoints and functionality

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_status() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

print_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Configuration
BASE_URL="http://localhost:8080/api"
TEST_USER_EMAIL="test@clipflow.com"
TEST_USER_PASSWORD="testpass123"
TEST_USERNAME="testuser"

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to run a test
run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_status="$3"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    print_status "Running: $test_name"
    
    if eval "$test_command" > /tmp/test_output.json 2>&1; then
        if [ "$expected_status" = "success" ]; then
            print_success "$test_name"
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            print_error "$test_name (unexpected success)"
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
    else
        if [ "$expected_status" = "failure" ]; then
            print_success "$test_name (expected failure)"
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            print_error "$test_name"
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
    fi
}



# Function to check if server is running
check_server() {
    print_status "Checking if server is running..."
    if curl -s "$BASE_URL/me" > /dev/null 2>&1; then
        print_success "Server is running"
        return 0
    else
        print_error "Server is not running. Please start the server first:"
        echo "  go run main.go"
        exit 1
    fi
}

# Function to create test video file
create_test_video() {
    print_status "Creating test video file..."
    
    # Create a simple test video using ffmpeg
    if command -v ffmpeg > /dev/null 2>&1; then
        ffmpeg -f lavfi -i testsrc=duration=5:size=320x240:rate=1 -c:v libx264 -t 5 test_video.mp4 -y > /dev/null 2>&1
        print_success "Test video created"
    else
        print_warning "ffmpeg not found, using dummy file"
        echo "dummy video content" > test_video.mp4
    fi
}

# Function to cleanup test files
cleanup() {
    rm -f test_video.mp4 /tmp/test_output.json
    print_status "Test files cleaned up"
}

# Main test functions
test_me_endpoint() {
    print_status "Testing /api/me endpoint..."
    
    # Test without token
    run_test "GET /me (no token)" \
        "curl -s '$BASE_URL/me' > /tmp/test_output.json" \
        "success"
    
    # Extract token from response
    TOKEN=$(cat /tmp/test_output.json | jq -r '.token' 2>/dev/null || echo "")
    
    if [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
        print_success "Token received: ${TOKEN:0:20}..."
    else
        print_error "No token received"
        return 1
    fi
    
    # Test with token
    run_test "GET /me (with token)" \
        "curl -s -H 'Authorization: Bearer $TOKEN' '$BASE_URL/me' > /tmp/test_output.json" \
        "success"
}

test_register_login() {
    print_status "Testing registration and login..."
    
    # Test registration
    run_test "POST /register" \
        "curl -s -X POST -H 'Content-Type: application/json' -d '{\"email\":\"$TEST_USER_EMAIL\",\"username\":\"$TEST_USERNAME\",\"password\":\"$TEST_USER_PASSWORD\"}' '$BASE_URL/register' > /tmp/test_output.json" \
        "success"
    
    # Test login
    run_test "POST /login" \
        "curl -s -X POST -H 'Content-Type: application/json' -d '{\"email\":\"$TEST_USER_EMAIL\",\"password\":\"$TEST_USER_PASSWORD\"}' '$BASE_URL/login' > /tmp/test_output.json" \
        "success"
    
    # Extract token from login response
    LOGIN_TOKEN=$(cat /tmp/test_output.json | jq -r '.token' 2>/dev/null || echo "")
    
    if [ -n "$LOGIN_TOKEN" ] && [ "$LOGIN_TOKEN" != "null" ]; then
        print_success "Login successful, token: ${LOGIN_TOKEN:0:20}..."
        TOKEN="$LOGIN_TOKEN"
    else
        print_error "Login failed"
        return 1
    fi
}

test_video_upload() {
    print_status "Testing video upload..."
    
    # Create test video if it doesn't exist
    if [ ! -f test_video.mp4 ]; then
        create_test_video
    fi
    
    # Test video upload without authentication (using existing uploaded file)
    run_test "POST /generate-video (anonymous)" \
        "curl -s -X POST -H 'Content-Type: application/json' -d '{\"userID\":\"test_user_123\",\"outputsize\":\"16:9\",\"videos\":[{\"file\":\"/uploads/20250701_160420_a7d805cd.mp4\",\"options\":{\"slowmotion\":false,\"index\":0,\"mute\":false}}],\"youtube\":[],\"audio\":[]}' '$BASE_URL/generate-video' > /tmp/test_output.json" \
        "success"
    
    # Extract task ID
    TASK_ID=$(cat /tmp/test_output.json | jq -r '.taskId' 2>/dev/null || echo "")
    
    if [ -n "$TASK_ID" ] && [ "$TASK_ID" != "null" ]; then
        print_success "Task created: $TASK_ID"
    else
        print_error "No task ID received"
        return 1
    fi
    
    # Test video upload with authentication
    if [ -n "$TOKEN" ]; then
        run_test "POST /generate-video (authenticated)" \
            "curl -s -X POST -H 'Content-Type: application/json' -H 'Authorization: Bearer $TOKEN' -d '{\"userID\":\"test_user_123\",\"outputsize\":\"16:9\",\"videos\":[{\"file\":\"/uploads/20250701_160420_a7d805cd.mp4\",\"options\":{\"slowmotion\":false,\"index\":0,\"mute\":false}}],\"youtube\":[],\"audio\":[]}' '$BASE_URL/generate-video' > /tmp/test_output.json" \
            "success"
        
        AUTH_TASK_ID=$(cat /tmp/test_output.json | jq -r '.taskId' 2>/dev/null || echo "")
        if [ -n "$AUTH_TASK_ID" ] && [ "$AUTH_TASK_ID" != "null" ]; then
            print_success "Authenticated task created: $AUTH_TASK_ID"
            TASK_ID="$AUTH_TASK_ID"  # Use authenticated task for further tests
        fi
    fi
}

test_task_status() {
    print_status "Testing task status endpoints..."
    
    if [ -z "$TASK_ID" ]; then
        print_warning "No task ID available, skipping task status tests"
        return
    fi
    
    # Test getting task status
    run_test "GET /task/:taskId" \
        "curl -s '$BASE_URL/task/$TASK_ID' > /tmp/test_output.json" \
        "success"
    
    # Test getting task status with authentication
    if [ -n "$TOKEN" ]; then
        run_test "GET /task/:taskId (authenticated)" \
            "curl -s -H 'Authorization: Bearer $TOKEN' '$BASE_URL/task/$TASK_ID' > /tmp/test_output.json" \
            "success"
    fi
    
    # Test getting user tasks
    if [ -n "$TOKEN" ]; then
        run_test "GET /tasks (authenticated)" \
            "curl -s -H 'Authorization: Bearer $TOKEN' '$BASE_URL/tasks?userID=test_user_123' > /tmp/test_output.json" \
            "success"
    fi
    
    # Test getting tasks without authentication
    run_test "GET /tasks (anonymous)" \
        "curl -s '$BASE_URL/tasks?userID=test_user_123' > /tmp/test_output.json" \
        "success"
    
    # Test getting tasks without userID parameter (should fail)
    run_test "GET /tasks (missing userID)" \
        "curl -s '$BASE_URL/tasks' > /tmp/test_output.json" \
        "failure"
}

test_youtube_processing() {
    print_status "Testing YouTube video processing..."
    
    # Test YouTube processing (this will fail if yt-dlp is not installed, which is expected)
    run_test "POST /generate-video (YouTube)" \
        "curl -s -X POST -H 'Content-Type: application/json' -d '{\"userID\":\"test_user_123\",\"outputsize\":\"16:9\",\"youtube\":[{\"url\":\"https://www.youtube.com/watch?v=dQw4w9WgXcQ\",\"quality\":\"720p\",\"segments\":[{\"index\":0,\"timeline\":{\"start\":\"0:00\",\"end\":\"0:10\"},\"options\":{\"slowmotion\":false,\"mute\":false}}]}],\"videos\":[],\"audio\":[]}' '$BASE_URL/generate-video' > /tmp/test_output.json" \
        "success"
    
    YOUTUBE_TASK_ID=$(cat /tmp/test_output.json | jq -r '.taskId' 2>/dev/null || echo "")
    if [ -n "$YOUTUBE_TASK_ID" ] && [ "$YOUTUBE_TASK_ID" != "null" ]; then
        print_success "YouTube task created: $YOUTUBE_TASK_ID"
    fi
}

test_error_handling() {
    print_status "Testing error handling..."
    
    # Test invalid JSON
    run_test "POST /generate-video (invalid JSON)" \
        "curl -s -X POST -H 'Content-Type: application/json' -d 'invalid-json' '$BASE_URL/generate-video' > /tmp/test_output.json" \
        "failure"
    
    # Test missing required fields
    run_test "POST /generate-video (missing videos)" \
        "curl -s -X POST -H 'Content-Type: application/json' -d '{\"userID\":\"test\",\"outputsize\":\"16:9\"}' '$BASE_URL/generate-video' > /tmp/test_output.json" \
        "failure"
    
    # Test invalid task ID
    run_test "GET /task/invalid-id" \
        "curl -s '$BASE_URL/task/invalid-task-id' > /tmp/test_output.json" \
        "failure"
    
    # Test invalid registration data
    run_test "POST /register (invalid email)" \
        "curl -s -X POST -H 'Content-Type: application/json' -d '{\"email\":\"invalid-email\",\"username\":\"test\",\"password\":\"123\"}' '$BASE_URL/register' > /tmp/test_output.json" \
        "failure"
}

test_file_validation() {
    print_status "Testing file validation..."
    
    # Create invalid file
    echo "not a video file" > invalid_file.txt
    
    # Test invalid file path
    run_test "POST /generate-video (invalid file path)" \
        "curl -s -X POST -H 'Content-Type: application/json' -d '{\"userID\":\"test_user_123\",\"outputsize\":\"16:9\",\"videos\":[{\"file\":\"/uploads/nonexistent_file.mp4\",\"options\":{\"slowmotion\":false,\"index\":0,\"mute\":false}}],\"youtube\":[],\"audio\":[]}' '$BASE_URL/generate-video' > /tmp/test_output.json" \
        "failure"
    
    rm -f invalid_file.txt
}

test_video_merge() {
    print_status "Testing video merge functionality..."
    
    # Check if uploaded files exist
    if [ ! -f "uploads/20250701_160420_a7d805cd.mp4" ]; then
        print_warning "No uploaded files found, skipping merge test"
        return
    fi
    
    print_success "Found uploaded file: uploads/20250701_160420_a7d805cd.mp4"
    
    # Test file path resolution
    TEST_FILE="uploads/20250701_160420_a7d805cd.mp4"
    if [ -f "$TEST_FILE" ]; then
        ABS_PATH=$(realpath "$TEST_FILE")
        print_success "File exists: $TEST_FILE"
        print_status "Absolute path: $ABS_PATH"
    else
        print_error "Test file not found: $TEST_FILE"
        return 1
    fi
    
    # Test video merge API endpoint
    print_status "Testing video merge API endpoint..."
    run_test "POST /generate-video (merge test)" \
        "curl -s -X POST -H 'Content-Type: application/json' -d '{\"userID\":\"test_user_merge_123\",\"outputsize\":\"16:9\",\"videos\":[{\"file\":\"/uploads/20250701_160420_a7d805cd.mp4\",\"options\":{\"slowmotion\":false,\"index\":0,\"mute\":false}}],\"youtube\":[],\"audio\":[]}' '$BASE_URL/generate-video' > /tmp/test_output.json" \
        "success"
    
    # Extract task ID from response
    MERGE_TASK_ID=$(cat /tmp/test_output.json | jq -r '.taskId' 2>/dev/null || echo "")
    
    if [ -n "$MERGE_TASK_ID" ] && [ "$MERGE_TASK_ID" != "null" ]; then
        print_success "Merge task created: $MERGE_TASK_ID"
        
        # Wait for task completion (with timeout)
        print_status "Waiting for task completion..."
        TIMEOUT=60
        COUNTER=0
        
        while [ $COUNTER -lt $TIMEOUT ]; do
            sleep 2
            COUNTER=$((COUNTER + 2))
            
            # Check task status
            TASK_STATUS=$(curl -s "$BASE_URL/task/$MERGE_TASK_ID" | jq -r '.status' 2>/dev/null || echo "unknown")
            
            if [ "$TASK_STATUS" = "completed" ]; then
                print_success "Task completed successfully!"
                
                # Check if output file was created
                OUTPUT_FILE=$(curl -s "$BASE_URL/task/$MERGE_TASK_ID" | jq -r '.output_file' 2>/dev/null || echo "")
                if [ -n "$OUTPUT_FILE" ] && [ "$OUTPUT_FILE" != "null" ]; then
                    OUTPUT_PATH=".${OUTPUT_FILE}"
                    if [ -f "$OUTPUT_PATH" ]; then
                        FILE_SIZE=$(ls -lh "$OUTPUT_PATH" | awk '{print $5}')
                        print_success "Output file created: $OUTPUT_FILE (Size: $FILE_SIZE)"
                    else
                        print_warning "Output file not found at: $OUTPUT_PATH"
                    fi
                fi
                break
            elif [ "$TASK_STATUS" = "failed" ]; then
                ERROR_MSG=$(curl -s "$BASE_URL/task/$MERGE_TASK_ID" | jq -r '.message' 2>/dev/null || echo "Unknown error")
                print_error "Task failed: $ERROR_MSG"
                break
            else
                print_status "Task status: $TASK_STATUS (${COUNTER}s elapsed)"
            fi
        done
        
        if [ $COUNTER -ge $TIMEOUT ]; then
            print_warning "Task timeout after ${TIMEOUT}s"
        fi
        
    else
        print_error "No task ID received for merge test"
        return 1
    fi
}

# Main test execution
main() {
    echo "🚀 Starting Clipflow API Tests"
    echo "================================"
    
    # Check if server is running
    check_server
    
    # Create test video
    create_test_video
    
    # Run tests
    test_me_endpoint
    test_register_login
    test_video_upload
    test_task_status
    test_youtube_processing
    test_error_handling
    test_file_validation
    test_video_merge
    
    # Cleanup
    cleanup
    
    # Print results
    echo ""
    echo "================================"
    echo "📊 Test Results Summary"
    echo "================================"
    echo "Total Tests: $TOTAL_TESTS"
    echo "Passed: $PASSED_TESTS"
    echo "Failed: $FAILED_TESTS"
    
    if [ $FAILED_TESTS -eq 0 ]; then
        echo ""
        print_success "All tests passed! 🎉"
        exit 0
    else
        echo ""
        print_error "Some tests failed! ❌"
        exit 1
    fi
}

# Handle script interruption
trap cleanup EXIT

# Run main function
main "$@" 