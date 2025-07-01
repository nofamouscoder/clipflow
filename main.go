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
	"github.com/joho/godotenv"
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

type MeResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
	New   bool         `json:"new"`
}

var db *models.Database

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, using default configuration")
	}

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

	// Serve static frontend (Next.js build)
	// router.Static("/", "./static")
	// Fallback for SPA routing
	router.NoRoute(func(c *gin.Context) {
		htmlContent, err := os.ReadFile("./static/index.html")
		if err != nil {
			log.Printf("Error loading static/index.html: %v", err)
			c.String(http.StatusInternalServerError, "Error loading frontend")
			return
		}
		c.Header("Content-Type", "text/html")
		c.Data(http.StatusOK, "text/html", htmlContent)
	})

	// API routes
	api := router.Group("/api")
	{
		// Public routes
		api.POST("/register", registerHandler)
		api.POST("/login", loginHandler)
		api.GET("/me", meHandler)

		// File upload route
		api.POST("/upload", uploadFileHandler)

		// Protected routes (using optional auth middleware)
		protected := api.Group("/")
		protected.Use(middleware.OptionalAuthMiddleware(db))
		{
			protected.POST("/generate-video", generateVideoHandler)
			protected.GET("/tasks", getUserTasksHandler)
			protected.GET("/task/:taskId", getTaskStatusHandler)
			protected.DELETE("/task/:taskId", deleteTaskHandler)
		}
	}

	log.Printf("Server starting on %s:%s", config.AppConfig.Server.Host, config.AppConfig.Server.Port)
	router.Run(fmt.Sprintf("%s:%s", config.AppConfig.Server.Host, config.AppConfig.Server.Port))
}

func meHandler(c *gin.Context) {
	// Check if user is authenticated
	userID, exists := c.Get("userID")
	if !exists {
		// Create anonymous user
		user := &models.User{
			ID:           uuid.New().String(),
			Email:        fmt.Sprintf("anonymous_%s@clipflow.local", uuid.New().String()[:8]),
			Username:     fmt.Sprintf("Anonymous_%s", uuid.New().String()[:8]),
			PasswordHash: "", // No password for anonymous users
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

		c.JSON(http.StatusOK, MeResponse{
			Token: token,
			User:  user,
			New:   true,
		})
		return
	}

	// User is authenticated, return existing user info
	user, err := db.GetUserByID(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	// Generate new token
	token, err := auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, MeResponse{
		Token: token,
		User:  user,
		New:   false,
	})
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
	// Get or create user
	userID, exists := c.Get("userID")
	if !exists {
		// Create anonymous user
		user := &models.User{
			ID:           uuid.New().String(),
			Email:        fmt.Sprintf("anonymous_%s@clipflow.local", uuid.New().String()[:8]),
			Username:     fmt.Sprintf("Anonymous_%s", uuid.New().String()[:8]),
			PasswordHash: "",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := db.CreateUser(user); err != nil {
			log.Printf("Failed to create anonymous user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}

		userID = user.ID
		log.Printf("Created anonymous user: %s", userID)
	}

	// Only accept application/json
	contentType := c.GetHeader("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		log.Printf("Invalid content type for video generation: %s", contentType)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content-Type must be application/json"})
		return
	}

	var req VideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Failed to parse video generation request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Video generation request from user %s: %d videos, %d YouTube clips, %d audio files",
		userID, len(req.Videos), len(req.YouTube), len(req.Audio))

	// Validate required fields
	if len(req.Videos) == 0 && len(req.YouTube) == 0 {
		log.Printf("Video generation rejected - no videos or YouTube clips provided")
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one video or YouTube clip is required"})
		return
	}

	// Validate and collect uploaded video file paths
	var uploadedVideos []string
	for i, v := range req.Videos {
		if v.File == "" {
			log.Printf("Video generation failed - missing file URL for video %d", i)
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Missing file URL for video %d", i)})
			return
		}
		if !strings.HasPrefix(v.File, "/uploads/") {
			log.Printf("Video generation failed - invalid file URL for video %d: %s", i, v.File)
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid file URL for video %d", i)})
			return
		}
		filePath := "." + v.File
		if _, err := os.Stat(filePath); err != nil {
			log.Printf("Video generation failed - file does not exist for video %d: %s", i, filePath)
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("File does not exist for video %d", i)})
			return
		}
		log.Printf("Video file validated: %s", filePath)
		uploadedVideos = append(uploadedVideos, filePath)
	}

	// Validate and collect uploaded audio file paths
	var uploadedAudio []string
	for i, a := range req.Audio {
		if a.File == "" {
			log.Printf("Video generation failed - missing file URL for audio %d", i)
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Missing file URL for audio %d", i)})
			return
		}
		if !strings.HasPrefix(a.File, "/uploads/") {
			log.Printf("Video generation failed - invalid file URL for audio %d: %s", i, a.File)
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid file URL for audio %d", i)})
			return
		}
		filePath := "." + a.File
		if _, err := os.Stat(filePath); err != nil {
			log.Printf("Video generation failed - file does not exist for audio %d: %s", i, filePath)
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("File does not exist for audio %d", i)})
			return
		}
		log.Printf("Audio file validated: %s", filePath)
		uploadedAudio = append(uploadedAudio, filePath)
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
		log.Printf("Failed to create task for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	log.Printf("Created video generation task %s for user %s", taskID, userID)

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
	userID, exists := c.Get("userID")

	task, err := db.GetTaskByID(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// If user is authenticated, check ownership
	if exists && task.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func getUserTasksHandler(c *gin.Context) {
	// Get userID from query parameter
	userID := c.Query("userID")
	if userID == "" {
		log.Printf("Tasks request failed - missing userID parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": "userID parameter is required"})
		return
	}

	log.Printf("Fetching tasks for user: %s", userID)

	tasks, err := db.GetTasksByUserID(userID)
	if err != nil {
		log.Printf("Failed to fetch tasks for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
		return
	}

	log.Printf("Found %d tasks for user %s", len(tasks), userID)
	c.JSON(http.StatusOK, tasks)
}

func deleteTaskHandler(c *gin.Context) {
	taskID := c.Param("taskId")
	userID, exists := c.Get("userID")

	task, err := db.GetTaskByID(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// If user is authenticated, check ownership
	if exists && task.UserID != userID {
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
	log.Printf("Starting video processing for task %s", taskID)

	task, err := db.GetTaskByID(taskID)
	if err != nil {
		log.Printf("Failed to get task %s: %v", taskID, err)
		return
	}

	task.Status = "processing"
	task.Message = "Starting video processing"
	task.Progress = 10
	if err := db.UpdateTask(task); err != nil {
		log.Printf("Failed to update task %s status: %v", taskID, err)
	}

	log.Printf("Task %s status updated to processing", taskID)

	// Create temporary directory for this task
	taskDir := filepath.Join(config.AppConfig.File.TempDir, taskID)
	os.MkdirAll(taskDir, 0755)
	defer os.RemoveAll(taskDir) // Clean up after processing

	var videoFiles []string
	videoIndex := 0

	// Process YouTube videos
	for _, ytClip := range req.YouTube {
		log.Printf("Processing YouTube video: %s", ytClip.URL)
		task.Message = fmt.Sprintf("Downloading YouTube video: %s", ytClip.URL)
		task.Progress = 20 + (videoIndex * 10)
		if err := db.UpdateTask(task); err != nil {
			log.Printf("Failed to update task %s progress: %v", taskID, err)
		}

		for _, segment := range ytClip.Segments {
			fileName := fmt.Sprintf("yt_%d_%s.mp4", segment.Index, generateFileHash(ytClip.URL+segment.Timeline.Start+segment.Timeline.End))
			outputPath := filepath.Join(taskDir, fileName)

			log.Printf("Downloading YouTube segment: %s (%s - %s)", ytClip.URL, segment.Timeline.Start, segment.Timeline.End)

			if err := downloadYouTubeSegment(ytClip.URL, ytClip.Quality, segment.Timeline, outputPath); err != nil {
				log.Printf("Failed to download YouTube segment for task %s: %v", taskID, err)
				task.Status = "failed"
				task.Message = fmt.Sprintf("Failed to download YouTube video: %v", err)
				if err := db.UpdateTask(task); err != nil {
					log.Printf("Failed to update failed task %s: %v", taskID, err)
				}
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
			log.Printf("Processing uploaded video %d: %s", i, videoPath)

			// Apply video options if specified
			if video.Options.Slowmotion || video.Options.Mute {
				processedPath := filepath.Join(taskDir, fmt.Sprintf("processed_upload_%d.mp4", i))
				log.Printf("Applying effects to video %d: slowmotion=%v, mute=%v", i, video.Options.Slowmotion, video.Options.Mute)

				if err := applyVideoEffects(videoPath, processedPath, video.Options.Slowmotion, video.Options.Mute); err != nil {
					log.Printf("Failed to apply effects to uploaded video %d: %v", i, err)
					task.Status = "failed"
					task.Message = fmt.Sprintf("Failed to apply effects to uploaded video: %v", err)
					if err := db.UpdateTask(task); err != nil {
						log.Printf("Failed to update failed task %s: %v", taskID, err)
					}
					return
				}
				videoFiles = append(videoFiles, processedPath)
			} else {
				videoFiles = append(videoFiles, videoPath)
			}
		}
	}

	if len(videoFiles) == 0 {
		log.Printf("No video files to process for task %s", taskID)
		task.Status = "failed"
		task.Message = "No video files to process"
		if err := db.UpdateTask(task); err != nil {
			log.Printf("Failed to update failed task %s: %v", taskID, err)
		}
		return
	}

	// Merge videos
	log.Printf("Merging %d videos for task %s", len(videoFiles), taskID)
	task.Message = "Merging videos"
	task.Progress = 70
	if err := db.UpdateTask(task); err != nil {
		log.Printf("Failed to update task %s progress: %v", taskID, err)
	}

	outputFileName := fmt.Sprintf("merged_%s_%s.mp4", task.UserID, taskID)
	outputPath := filepath.Join(config.AppConfig.File.OutputDir, outputFileName)
	log.Printf("Output path for task %s: %s", taskID, outputPath)

	if err := mergeVideos(videoFiles, outputPath, req.OutputSize); err != nil {
		log.Printf("Failed to merge videos for task %s: %v", taskID, err)
		task.Status = "failed"
		task.Message = fmt.Sprintf("Failed to merge videos: %v", err)
		if err := db.UpdateTask(task); err != nil {
			log.Printf("Failed to update failed task %s: %v", taskID, err)
		}
		return
	}

	// Complete task
	now := time.Now()
	task.Status = "completed"
	task.Progress = 100
	task.Message = "Video processing completed successfully"
	task.OutputFile = fmt.Sprintf("/output/%s", outputFileName)
	task.CompletedAt = &now

	log.Printf("Task %s completed successfully. Output: %s", taskID, task.OutputFile)

	if err := db.UpdateTask(task); err != nil {
		log.Printf("Failed to update completed task %s: %v", taskID, err)
	}
}

func downloadYouTubeSegment(url, quality string, timeline TimelineOptions, outputPath string) error {
	log.Printf("Downloading YouTube segment: %s (%s - %s) to %s", url, timeline.Start, timeline.End, outputPath)

	// Convert time format from MM:SS to seconds
	startSeconds, err := timeToSeconds(timeline.Start)
	if err != nil {
		log.Printf("Invalid start time for YouTube download: %s", timeline.Start)
		return fmt.Errorf("invalid start time: %v", err)
	}

	endSeconds, err := timeToSeconds(timeline.End)
	if err != nil {
		log.Printf("Invalid end time for YouTube download: %s", timeline.End)
		return fmt.Errorf("invalid end time: %v", err)
	}

	duration := endSeconds - startSeconds
	if duration <= 0 {
		log.Printf("Invalid time range for YouTube download: %s - %s", timeline.Start, timeline.End)
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

	log.Printf("Running yt-dlp command: %s %v", config.AppConfig.File.YTDLPPath, args)
	cmd := exec.Command(config.AppConfig.File.YTDLPPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("%s failed for %s: %v, output: %s", config.AppConfig.File.YTDLPPath, url, err, string(output))
		return fmt.Errorf("%s failed: %v, output: %s", config.AppConfig.File.YTDLPPath, err, string(output))
	}

	log.Printf("YouTube segment downloaded successfully: %s", outputPath)
	return nil
}

func applyVideoEffects(inputPath, outputPath string, slowmotion, mute bool) error {
	log.Printf("Applying video effects: %s -> %s (slowmotion=%v, mute=%v)", inputPath, outputPath, slowmotion, mute)

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

	log.Printf("Running ffmpeg command: ffmpeg %v", args)
	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("ffmpeg failed for effects: %v, output: %s", err, string(output))
		return fmt.Errorf("ffmpeg failed: %v, output: %s", err, string(output))
	}

	log.Printf("Video effects applied successfully: %s", outputPath)
	return nil
}

func mergeVideos(inputFiles []string, outputPath, outputSize string) error {
	log.Printf("Merging %d videos to %s with size %s", len(inputFiles), outputPath, outputSize)

	if len(inputFiles) == 0 {
		log.Printf("No input files provided for merge")
		return fmt.Errorf("no input files provided")
	}

	// Create input file list for ffmpeg
	listFile := strings.TrimSuffix(outputPath, ".mp4") + "_list.txt"
	listContent := ""
	for _, file := range inputFiles {
		// Check if file exists
		if _, err := os.Stat(file); os.IsNotExist(err) {
			log.Printf("Input file does not exist: %s", file)
			return fmt.Errorf("input file does not exist: %s", file)
		}

		// Convert to absolute path to avoid relative path issues
		absPath, err := filepath.Abs(file)
		if err != nil {
			log.Printf("Failed to get absolute path for %s: %v", file, err)
			return fmt.Errorf("failed to resolve file path: %v", err)
		}
		listContent += fmt.Sprintf("file '%s'\n", absPath)
		log.Printf("Adding file to merge list: %s (absolute: %s)", file, absPath)
	}

	log.Printf("Creating input list file: %s", listFile)
	if err := os.WriteFile(listFile, []byte(listContent), 0644); err != nil {
		log.Printf("Failed to create input list file: %v", err)
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

	log.Printf("Output dimensions: %dx%d", width, height)

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

	// Get current working directory for debugging
	if cwd, err := os.Getwd(); err == nil {
		log.Printf("Current working directory: %s", cwd)
	}

	log.Printf("Running ffmpeg merge command: ffmpeg %v", args)
	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("ffmpeg merge failed: %v, output: %s", err, string(output))
		return fmt.Errorf("ffmpeg merge failed: %v, output: %s", err, string(output))
	}

	log.Printf("Videos merged successfully: %s", outputPath)
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

// uploadFileHandler handles video/audio file uploads with validation
func uploadFileHandler(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		log.Printf("Upload error - no file: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	log.Printf("File upload attempt: %s, size: %d bytes, type: %s",
		header.Filename, header.Size, header.Header.Get("Content-Type"))

	// Validate file size (max 100MB for video, 20MB for audio)
	maxVideoSize := int64(100 * 1024 * 1024) // 100MB
	maxAudioSize := int64(20 * 1024 * 1024)  // 20MB
	fileType := header.Header.Get("Content-Type")

	if strings.HasPrefix(fileType, "video/") {
		if header.Size > maxVideoSize {
			log.Printf("Upload rejected - video file too large: %s (%d bytes)", header.Filename, header.Size)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Video file too large (max 100MB)"})
			return
		}
		if err := utils.ValidateVideoFile(header); err != nil {
			log.Printf("Upload rejected - invalid video file %s: %v", header.Filename, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else if strings.HasPrefix(fileType, "audio/") {
		if header.Size > maxAudioSize {
			log.Printf("Upload rejected - audio file too large: %s (%d bytes)", header.Filename, header.Size)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Audio file too large (max 20MB)"})
			return
		}
		if err := utils.ValidateAudioFile(header); err != nil {
			log.Printf("Upload rejected - invalid audio file %s: %v", header.Filename, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		log.Printf("Upload rejected - unsupported file type: %s (%s)", header.Filename, fileType)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported file type"})
		return
	}

	// Save file
	fileInfo, err := utils.SaveFile(header, config.AppConfig.File.UploadsDir)
	if err != nil {
		log.Printf("Upload failed - save error for %s: %v", header.Filename, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Return file URL
	fileURL := "/uploads/" + fileInfo.StoredName
	log.Printf("Upload successful: %s -> %s", header.Filename, fileURL)
	c.JSON(http.StatusOK, gin.H{"url": fileURL})
}
