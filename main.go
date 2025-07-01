package main

import (
	"crypto/md5"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"clipflow/auth"
	"clipflow/config"
	"clipflow/middleware"
	"clipflow/models"
	"clipflow/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type VideoRequest struct {
	UserID     string        `json:"userID"`
	OutputSize string        `json:"outputsize"`
	Videos     []VideoFile   `json:"videos"`
	YouTube    []YouTubeClip `json:"youtube"`
	Audio      []AudioFile   `json:"audio"`
}

type VideoFile struct {
	File    string       `json:"file"`
	Options VideoOptions `json:"options"`
}

type YouTubeClip struct {
	URL      string           `json:"url"`
	Quality  string           `json:"quality"`
	Segments []YouTubeSegment `json:"segments"`
}

type YouTubeSegment struct {
	Index    int             `json:"index"`
	Timeline TimelineOptions `json:"timeline"`
	Options  SegmentOptions  `json:"options"`
}

type TimelineOptions struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type VideoOptions struct {
	Slowmotion bool   `json:"slowmotion"`
	Index      int    `json:"index"`
	Mute       bool   `json:"mute"`
	StartTime  string `json:"startTime,omitempty"`
	EndTime    string `json:"endTime,omitempty"`
}

type SegmentOptions struct {
	Slowmotion bool `json:"slowmotion"`
	Mute       bool `json:"mute"`
}

type AudioFile struct {
	File string `json:"file"`
}

type TaskResponse struct {
	TaskID  string `json:"taskId"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

var db *models.Database

func main() {
	// Load configuration
	if err := config.LoadConfig(); err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Set JWT secret from config
	auth.SetJWTSecret(config.AppConfig.Security.JWTSecret)

	// Initialize database
	var err error
	db, err = models.NewDatabase(config.AppConfig.Database.Path)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Create necessary directories
	os.MkdirAll(config.AppConfig.File.TempDir, 0755)
	os.MkdirAll(config.AppConfig.File.OutputDir, 0755)
	os.MkdirAll(config.AppConfig.File.UploadsDir, 0755)

	router := gin.Default()

	// Configure CORS
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	router.Use(cors.New(corsConfig))

	// Serve static files
	router.Static("/output", config.AppConfig.File.OutputDir)
	router.Static("/uploads", config.AppConfig.File.UploadsDir)

	// Serve frontend
	router.StaticFile("/", "./index.html")

	// API routes
	api := router.Group("/api")
	{
		// Public routes
		api.POST("/register", registerHandler)
		api.POST("/login", loginHandler)

		// Protected routes
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware(db))
		{
			protected.POST("/generate-video", generateVideoHandler)
			protected.GET("/task/:taskId", getTaskStatusHandler)
			protected.GET("/tasks", getUserTasksHandler)
			protected.DELETE("/task/:taskId", deleteTaskHandler)
		}
	}

	log.Printf("Server starting on %s:%s", config.AppConfig.Server.Host, config.AppConfig.Server.Port)
	router.Run(fmt.Sprintf("%s:%s", config.AppConfig.Server.Host, config.AppConfig.Server.Port))
}

func registerHandler(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already exists
	existingUser, _ := db.GetUserByEmail(req.Email)
	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User with this email already exists"})
		return
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}

	// Create user
	user := &models.User{
		ID:           uuid.New().String(),
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := db.CreateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Generate token
	token, err := auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, LoginResponse{
		Token: token,
		User:  user,
	})
}

func loginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user by email
	user, err := db.GetUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check password
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate token
	token, err := auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token: token,
		User:  user,
	})
}

func generateVideoHandler(c *gin.Context) {
	// Get user from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB max memory
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form data"})
		return
	}

	// Get JSON data from form
	jsonData := c.PostForm("data")
	if jsonData == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'data' field in form"})
		return
	}

	// Parse JSON request
	var req VideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate required fields
	if len(req.Videos) == 0 && len(req.YouTube) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one video or YouTube clip is required"})
		return
	}

	// Process uploaded files
	var uploadedVideos []string
	var uploadedAudio []string

	// Process video files
	for i := range req.Videos {
		fileKey := fmt.Sprintf("video_%d", i)
		if file, header, err := c.Request.FormFile(fileKey); err == nil {
			defer file.Close()

			// Validate video file
			if err := utils.ValidateVideoFile(header); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid video file %d: %v", i, err)})
				return
			}

			// Save video file
			fileInfo, err := utils.SaveFile(header, config.AppConfig.File.UploadsDir)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save video file %d: %v", i, err)})
				return
			}

			uploadedVideos = append(uploadedVideos, fileInfo.FilePath)
		}
	}

	// Process audio files
	for i := range req.Audio {
		fileKey := fmt.Sprintf("audio_%d", i)
		if file, header, err := c.Request.FormFile(fileKey); err == nil {
			defer file.Close()

			// Validate audio file
			if err := utils.ValidateAudioFile(header); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid audio file %d: %v", i, err)})
				return
			}

			// Save audio file
			fileInfo, err := utils.SaveFile(header, config.AppConfig.File.UploadsDir)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save audio file %d: %v", i, err)})
				return
			}

			uploadedAudio = append(uploadedAudio, fileInfo.FilePath)
		}
	}

	// Create task
	taskID := uuid.New().String()
	task := &models.Task{
		ID:        taskID,
		UserID:    userID.(string),
		Status:    "pending",
		Progress:  0,
		Message:   "Task created, preparing for processing",
		CreatedAt: time.Now(),
	}

	if err := db.CreateTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	// Start processing in goroutine
	go processVideoRequest(taskID, req, uploadedVideos, uploadedAudio)

	c.JSON(http.StatusOK, TaskResponse{
		TaskID:  taskID,
		Status:  "pending",
		Message: "Video generation task created successfully",
	})
}

func getTaskStatusHandler(c *gin.Context) {
	taskID := c.Param("taskId")
	userID, _ := c.Get("userID")

	task, err := db.GetTaskByID(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Check if user owns this task
	if task.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func getUserTasksHandler(c *gin.Context) {
	userID, _ := c.Get("userID")

	tasks, err := db.GetTasksByUserID(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

func deleteTaskHandler(c *gin.Context) {
	taskID := c.Param("taskId")
	userID, _ := c.Get("userID")

	task, err := db.GetTaskByID(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Check if user owns this task
	if task.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := db.DeleteTask(taskID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task deleted successfully"})
}

func processVideoRequest(taskID string, req VideoRequest, uploadedVideos []string, uploadedAudio []string) {
	task, err := db.GetTaskByID(taskID)
	if err != nil {
		log.Printf("Failed to get task %s: %v", taskID, err)
		return
	}

	task.Status = "processing"
	task.Message = "Starting video processing"
	task.Progress = 10
	db.UpdateTask(task)

	// Create temporary directory for this task
	taskDir := filepath.Join(config.AppConfig.File.TempDir, taskID)
	os.MkdirAll(taskDir, 0755)
	defer os.RemoveAll(taskDir) // Clean up after processing

	var videoFiles []string
	videoIndex := 0

	// Process YouTube videos
	for _, ytClip := range req.YouTube {
		task.Message = fmt.Sprintf("Downloading YouTube video: %s", ytClip.URL)
		task.Progress = 20 + (videoIndex * 10)
		db.UpdateTask(task)

		for _, segment := range ytClip.Segments {
			fileName := fmt.Sprintf("yt_%d_%s.mp4", segment.Index, generateFileHash(ytClip.URL+segment.Timeline.Start+segment.Timeline.End))
			outputPath := filepath.Join(taskDir, fileName)

			if err := downloadYouTubeSegment(ytClip.URL, ytClip.Quality, segment.Timeline, outputPath); err != nil {
				task.Status = "failed"
				task.Message = fmt.Sprintf("Failed to download YouTube video: %v", err)
				db.UpdateTask(task)
				return
			}

			// Apply segment options (slowmotion, mute)
			if segment.Options.Slowmotion || segment.Options.Mute {
				processedPath := filepath.Join(taskDir, fmt.Sprintf("processed_%s", fileName))
				if err := applyVideoEffects(outputPath, processedPath, segment.Options.Slowmotion, segment.Options.Mute); err != nil {
					task.Status = "failed"
					task.Message = fmt.Sprintf("Failed to apply effects: %v", err)
					db.UpdateTask(task)
					return
				}
				os.Remove(outputPath)
				os.Rename(processedPath, outputPath)
			}

			videoFiles = append(videoFiles, outputPath)
			videoIndex++
		}
	}

	// Process uploaded video files
	for i, videoPath := range uploadedVideos {
		if i < len(req.Videos) {
			video := req.Videos[i]

			// Apply video options if specified
			if video.Options.Slowmotion || video.Options.Mute {
				processedPath := filepath.Join(taskDir, fmt.Sprintf("processed_upload_%d.mp4", i))
				if err := applyVideoEffects(videoPath, processedPath, video.Options.Slowmotion, video.Options.Mute); err != nil {
					task.Status = "failed"
					task.Message = fmt.Sprintf("Failed to apply effects to uploaded video: %v", err)
					db.UpdateTask(task)
					return
				}
				videoFiles = append(videoFiles, processedPath)
			} else {
				videoFiles = append(videoFiles, videoPath)
			}
		}
	}

	if len(videoFiles) == 0 {
		task.Status = "failed"
		task.Message = "No video files to process"
		db.UpdateTask(task)
		return
	}

	// Merge videos
	task.Message = "Merging videos"
	task.Progress = 70
	db.UpdateTask(task)

	outputFileName := fmt.Sprintf("merged_%s_%s.mp4", task.UserID, taskID)
	outputPath := filepath.Join(config.AppConfig.File.OutputDir, outputFileName)

	if err := mergeVideos(videoFiles, outputPath, req.OutputSize); err != nil {
		task.Status = "failed"
		task.Message = fmt.Sprintf("Failed to merge videos: %v", err)
		db.UpdateTask(task)
		return
	}

	// Complete task
	now := time.Now()
	task.Status = "completed"
	task.Progress = 100
	task.Message = "Video processing completed successfully"
	task.OutputFile = fmt.Sprintf("/output/%s", outputFileName)
	task.CompletedAt = &now
	db.UpdateTask(task)
}

func downloadYouTubeSegment(url, quality string, timeline TimelineOptions, outputPath string) error {
	// Convert time format from MM:SS to seconds
	startSeconds, err := timeToSeconds(timeline.Start)
	if err != nil {
		return fmt.Errorf("invalid start time: %v", err)
	}

	endSeconds, err := timeToSeconds(timeline.End)
	if err != nil {
		return fmt.Errorf("invalid end time: %v", err)
	}

	duration := endSeconds - startSeconds
	if duration <= 0 {
		return fmt.Errorf("invalid time range")
	}

	// Build yt-dlp command
	args := []string{
		"--format", fmt.Sprintf("best[height<=%s]", strings.TrimSuffix(quality, "p")),
		"--output", outputPath,
		"--external-downloader", "ffmpeg",
		"--external-downloader-args",
		fmt.Sprintf("ffmpeg_i:-ss %d -t %d", startSeconds, duration),
		url,
	}

	cmd := exec.Command("yt-dlp", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("yt-dlp failed: %v, output: %s", err, string(output))
	}

	return nil
}

func applyVideoEffects(inputPath, outputPath string, slowmotion, mute bool) error {
	args := []string{"-i", inputPath}

	// Build filter complex
	var filters []string

	if slowmotion {
		filters = append(filters, "setpts=2.0*PTS") // 0.5x speed
	}

	if mute {
		args = append(args, "-an") // Remove audio
	}

	if len(filters) > 0 {
		args = append(args, "-vf", strings.Join(filters, ","))
	}

	args = append(args, "-c:v", "libx264", outputPath)

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %v, output: %s", err, string(output))
	}

	return nil
}

func mergeVideos(inputFiles []string, outputPath, outputSize string) error {
	if len(inputFiles) == 0 {
		return fmt.Errorf("no input files provided")
	}

	// Create input file list for ffmpeg
	listFile := strings.TrimSuffix(outputPath, ".mp4") + "_list.txt"
	listContent := ""
	for _, file := range inputFiles {
		listContent += fmt.Sprintf("file '%s'\n", file)
	}

	if err := os.WriteFile(listFile, []byte(listContent), 0644); err != nil {
		return fmt.Errorf("failed to create input list: %v", err)
	}
	defer os.Remove(listFile)

	// Parse output size
	var width, height int
	switch outputSize {
	case "16:9":
		width, height = 1920, 1080
	case "9:16":
		width, height = 1080, 1920
	case "1:1":
		width, height = 1080, 1080
	case "4:3":
		width, height = 1440, 1080
	case "3:4":
		width, height = 1080, 1440
	default:
		width, height = 1920, 1080
	}

	// Build ffmpeg command
	args := []string{
		"-f", "concat",
		"-safe", "0",
		"-i", listFile,
		"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2", width, height, width, height),
		"-c:v", "libx264",
		"-crf", "23",
		"-preset", "medium",
		"-c:a", "aac",
		"-b:a", "128k",
		outputPath,
	}

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg merge failed: %v, output: %s", err, string(output))
	}

	return nil
}

func timeToSeconds(timeStr string) (int, error) {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid time format, expected MM:SS")
	}

	minutes, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid minutes: %v", err)
	}

	seconds, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid seconds: %v", err)
	}

	return minutes*60 + seconds, nil
}

func generateFileHash(input string) string {
	hash := md5.Sum([]byte(input))
	return fmt.Sprintf("%x", hash)[:8]
}
