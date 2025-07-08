package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	OutputSize string        `json:"outputsize"`
	FPS        int           `json:"fps"`
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
	File    string       `json:"file"`
	Options AudioOptions `json:"options"`
}

type AudioOptions struct {
	Volume  float64 `json:"volume"`
	FadeIn  bool    `json:"fadeIn"`
	FadeOut bool    `json:"fadeOut"`
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
	router.Static("/static", "./static")
	router.StaticFile("/clipflow-logo.svg", "./clipflow-logo.svg")

	// Serve frontend dynamically
	router.GET("/", func(c *gin.Context) {
		htmlContent, err := os.ReadFile("./index.html")
		if err != nil {
			log.Printf("Error loading index.html: %v", err)
			c.String(http.StatusInternalServerError, "Error loading frontend")
			return
		}
		c.Header("Content-Type", "text/html")
		c.Data(http.StatusOK, "text/html", htmlContent)
	})

	// Serve history page
	router.GET("/history", func(c *gin.Context) {
		htmlContent, err := os.ReadFile("./history.html")
		if err != nil {
			log.Printf("Error loading history.html: %v", err)
			c.String(http.StatusInternalServerError, "Error loading history page")
			return
		}
		c.Header("Content-Type", "text/html")
		c.Data(http.StatusOK, "text/html", htmlContent)
	})

	// Serve debug page
	router.GET("/debug", func(c *gin.Context) {
		htmlContent, err := os.ReadFile("./debug.html")
		if err != nil {
			log.Printf("Error loading debug.html: %v", err)
			c.String(http.StatusInternalServerError, "Error loading debug page")
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

		// Protected routes (using optional auth middleware)
		protected := api.Group("")
		protected.Use(middleware.OptionalAuthMiddleware(db))
		{
			protected.POST("/upload", uploadFileHandler)
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
	// 1. Check for JWT via middleware (userID in context)
	if userID, exists := c.Get("userID"); exists {
		log.Printf("/api/me: Authenticated via JWT, userID: %v", userID)
		user, err := db.GetUserByID(userID.(string))
		if err != nil {
			log.Printf("/api/me: Failed to get user by JWT userID: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
			return
		}
		token, err := auth.GenerateToken(user.ID, user.Email)
		if err != nil {
			log.Printf("/api/me: Failed to generate token for JWT userID: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}
		c.JSON(http.StatusOK, MeResponse{
			Token: token,
			User:  user,
			New:   false,
		})
		return
	}

	// 2. No JWT, check for userID in query param
	providedUserID := c.Query("userID")
	if providedUserID != "" {
		log.Printf("/api/me: No JWT, userID param provided: %s", providedUserID)
		user, err := db.GetUserByID(providedUserID)
		if err == nil && user != nil {
			log.Printf("/api/me: Found user for provided userID: %s", providedUserID)
			token, err := auth.GenerateToken(user.ID, user.Email)
			if err != nil {
				log.Printf("/api/me: Failed to generate token for provided userID: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
				return
			}
			c.JSON(http.StatusOK, MeResponse{
				Token: token,
				User:  user,
				New:   false,
			})
			return
		} else {
			log.Printf("/api/me: Provided userID not found in database, creating new user with provided ID: %s", providedUserID)
			// Create new user with the provided userID
			user := &models.User{
				ID:           providedUserID,
				Email:        fmt.Sprintf("anonymous_%s@clipflow.local", uuid.New().String()[:8]),
				Username:     fmt.Sprintf("Anonymous_%s", uuid.New().String()[:8]),
				PasswordHash: "",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			if err := db.CreateUser(user); err != nil {
				log.Printf("/api/me: Failed to create new user with provided ID: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
				return
			}
			token, err := auth.GenerateToken(user.ID, user.Email)
			if err != nil {
				log.Printf("/api/me: Failed to generate token for new user with provided ID: %v", err)
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
	}

	// 3. No JWT, no userID provided: create new user with new ID
	log.Printf("/api/me: Creating new anonymous user with new ID.")
	user := &models.User{
		ID:           uuid.New().String(),
		Email:        fmt.Sprintf("anonymous_%s@clipflow.local", uuid.New().String()[:8]),
		Username:     fmt.Sprintf("Anonymous_%s", uuid.New().String()[:8]),
		PasswordHash: "",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.CreateUser(user); err != nil {
		log.Printf("/api/me: Failed to create new user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}
	token, err := auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		log.Printf("/api/me: Failed to generate token for new user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, MeResponse{
		Token: token,
		User:  user,
		New:   true,
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
	userID, exists := c.Get("userID")
	if !exists || userID == nil {
		log.Printf("Video generation request failed - missing userID in JWT context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: userID required in token"})
		return
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

	// Create task details JSON
	taskDetails := map[string]interface{}{
		"outputSize": req.OutputSize,
		"fps":        req.FPS,
		"videos":     req.Videos,
		"youtube":    req.YouTube,
		"audio":      req.Audio,
	}

	taskDetailsJSON, err := json.Marshal(taskDetails)
	if err != nil {
		log.Printf("Failed to marshal task details for task %s: %v", taskID, err)
	}

	task := &models.Task{
		ID:          taskID,
		UserID:      userID.(string),
		Status:      "pending",
		Progress:    0,
		Message:     "Task created, preparing for processing",
		TaskDetails: string(taskDetailsJSON),
		CreatedAt:   time.Now(),
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
	userID, exists := c.Get("userID")
	if !exists || userID == nil {
		log.Printf("Tasks request failed - missing userID in JWT context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: userID required in token"})
		return
	}

	log.Printf("Fetching tasks for user: %s", userID)

	tasks, err := db.GetTasksByUserID(userID.(string))
	if err != nil {
		log.Printf("Failed to fetch tasks for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
		return
	}

	// Ensure we always return an array, even if empty
	if tasks == nil || len(tasks) == 0 {
		c.Header("Content-Type", "application/json")
		c.Writer.Write([]byte("[]"))
		return
	}

	log.Printf("Found %d tasks for user %s", len(tasks), userID)
	c.JSON(http.StatusOK, tasks)
}

func deleteTaskHandler(c *gin.Context) {
	taskID := c.Param("taskId")
	userID, exists := c.Get("userID")
	if !exists || userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: userID required in token"})
		return
	}
	task, err := db.GetTaskByID(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}
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

	// Create a slice to hold all video clips with their indices
	type VideoClip struct {
		Index     int
		FilePath  string
		IsYouTube bool
		Original  interface{} // Store original request data for reference
	}

	var videoClips []VideoClip
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

			log.Printf("Downloading YouTube segment: %s (%s - %s) with index %d", ytClip.URL, segment.Timeline.Start, segment.Timeline.End, segment.Index)

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

			videoClips = append(videoClips, VideoClip{
				Index:     segment.Index,
				FilePath:  outputPath,
				IsYouTube: true,
				Original:  segment,
			})
			videoIndex++
		}
	}

	// Process uploaded video files
	for i, video := range req.Videos {
		log.Printf("Processing uploaded video %d: %s with index %d", i, video.File, video.Options.Index)

		// Find the corresponding uploaded video path
		var videoPath string
		for _, uploadedPath := range uploadedVideos {
			if strings.HasSuffix(uploadedPath, filepath.Base(video.File)) {
				videoPath = uploadedPath
				break
			}
		}

		if videoPath == "" {
			log.Printf("Failed to find uploaded video path for %s", video.File)
			task.Status = "failed"
			task.Message = fmt.Sprintf("Failed to find uploaded video: %s", video.File)
			if err := db.UpdateTask(task); err != nil {
				log.Printf("Failed to update failed task %s: %v", taskID, err)
			}
			return
		}

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
			videoClips = append(videoClips, VideoClip{
				Index:     video.Options.Index,
				FilePath:  processedPath,
				IsYouTube: false,
				Original:  video,
			})
		} else {
			videoClips = append(videoClips, VideoClip{
				Index:     video.Options.Index,
				FilePath:  videoPath,
				IsYouTube: false,
				Original:  video,
			})
		}
	}

	if len(videoClips) == 0 {
		log.Printf("No video files to process for task %s", taskID)
		task.Status = "failed"
		task.Message = "No video files to process"
		if err := db.UpdateTask(task); err != nil {
			log.Printf("Failed to update failed task %s: %v", taskID, err)
		}
		return
	}

	// Sort video clips by index to ensure proper ordering
	sort.Slice(videoClips, func(i, j int) bool {
		return videoClips[i].Index < videoClips[j].Index
	})

	log.Printf("Video clips sorted by index: %v", func() []int {
		var indices []int
		for _, clip := range videoClips {
			indices = append(indices, clip.Index)
		}
		return indices
	}())

	// Extract file paths in sorted order
	var videoFiles []string
	for _, clip := range videoClips {
		videoFiles = append(videoFiles, clip.FilePath)
		log.Printf("Adding video file to merge (index %d): %s", clip.Index, clip.FilePath)
	}

	// Process audio files if provided
	var audioFiles []string
	if len(req.Audio) > 0 {
		log.Printf("Processing %d audio files", len(req.Audio))
		task.Message = "Processing audio files"
		task.Progress = 60
		if err := db.UpdateTask(task); err != nil {
			log.Printf("Failed to update task %s progress: %v", taskID, err)
		}

		for i, audio := range req.Audio {
			// Find the corresponding uploaded audio path
			var audioPath string
			for _, uploadedPath := range uploadedAudio {
				if strings.HasSuffix(uploadedPath, filepath.Base(audio.File)) {
					audioPath = uploadedPath
					break
				}
			}

			if audioPath == "" {
				log.Printf("Failed to find uploaded audio path for %s", audio.File)
				continue
			}

			// Process audio with options (volume, fade in/out)
			processedAudioPath := filepath.Join(taskDir, fmt.Sprintf("processed_audio_%d.mp3", i))
			if err := processAudioFile(audioPath, processedAudioPath, audio.Options); err != nil {
				log.Printf("Failed to process audio file %d: %v", i, err)
				continue
			}
			audioFiles = append(audioFiles, processedAudioPath)
		}
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

	// If we have audio files, merge them with the video
	if len(audioFiles) > 0 {
		if err := mergeVideosWithAudio(videoFiles, audioFiles, outputPath, req.OutputSize, req.FPS); err != nil {
			log.Printf("Failed to merge videos with audio for task %s: %v", taskID, err)
			task.Status = "failed"
			task.Message = fmt.Sprintf("Failed to merge videos with audio: %v", err)
			if err := db.UpdateTask(task); err != nil {
				log.Printf("Failed to update failed task %s: %v", taskID, err)
			}
			return
		}
	} else {
		if err := mergeVideos(videoFiles, outputPath, req.OutputSize, req.FPS); err != nil {
			log.Printf("Failed to merge videos for task %s: %v", taskID, err)
			task.Status = "failed"
			task.Message = fmt.Sprintf("Failed to merge videos: %v", err)
			if err := db.UpdateTask(task); err != nil {
				log.Printf("Failed to update failed task %s: %v", taskID, err)
			}
			return
		}
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

func mergeVideos(inputFiles []string, outputPath, outputSize string, fps int) error {
	log.Printf("Merging %d videos to %s with size %s and FPS %d", len(inputFiles), outputPath, outputSize, fps)

	if len(inputFiles) == 0 {
		log.Printf("No input files provided for merge")
		return fmt.Errorf("no input files provided")
	}

	// Normalize all videos to the target FPS first
	var normalizedFiles []string
	for i, file := range inputFiles {
		// Check if file exists
		if _, err := os.Stat(file); os.IsNotExist(err) {
			log.Printf("Input file does not exist: %s", file)
			return fmt.Errorf("input file does not exist: %s", file)
		}

		// Normalize FPS if needed
		normalizedPath := file
		if i > 0 || len(inputFiles) > 1 {
			// For multiple files or if we want to ensure consistency, normalize FPS
			normalizedPath = strings.TrimSuffix(file, filepath.Ext(file)) + "_normalized.mp4"
			if err := normalizeVideoFPS(file, normalizedPath, fps); err != nil {
				log.Printf("Failed to normalize FPS for %s: %v", file, err)
				return fmt.Errorf("failed to normalize FPS: %v", err)
			}
			defer os.Remove(normalizedPath) // Clean up temporary file
		}
		normalizedFiles = append(normalizedFiles, normalizedPath)
	}

	// Create input file list for ffmpeg
	listFile := strings.TrimSuffix(outputPath, ".mp4") + "_list.txt"
	listContent := ""
	for _, file := range normalizedFiles {
		// Convert to absolute path to avoid relative path issues
		absPath, err := filepath.Abs(file)
		if err != nil {
			log.Printf("Failed to get absolute path for %s: %v", file, err)
			return fmt.Errorf("failed to resolve file path: %v", err)
		}
		listContent += fmt.Sprintf("file '%s'\n", absPath)
		log.Printf("Adding normalized file to merge list: %s (absolute: %s)", file, absPath)
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
		"-r", fmt.Sprintf("%d", fps),
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
	log.Printf("📤 [Upload] Request received for path: %s", c.Request.URL.Path)
	log.Printf("📤 [Upload] Authorization header: %s", c.GetHeader("Authorization"))
	log.Printf("📤 [Upload] All headers: %v", c.Request.Header)

	userID, exists := c.Get("userID")
	log.Printf("📤 [Upload] userID from context: %v, exists: %v", userID, exists)

	// Check all context keys
	keys := make([]string, 0)
	for key := range c.Keys {
		keys = append(keys, key)
	}
	log.Printf("📤 [Upload] All context keys: %v", keys)

	if !exists || userID == nil {
		log.Printf("📤 [Upload] Unauthorized - userID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: userID required in token"})
		return
	}
	log.Printf("📤 [Upload] User authenticated: %s", userID)
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

func processAudioFile(inputPath, outputPath string, options AudioOptions) error {
	log.Printf("Processing audio file: %s with volume %.2f, fadeIn: %v, fadeOut: %v",
		inputPath, options.Volume, options.FadeIn, options.FadeOut)

	// Build ffmpeg command for audio processing
	args := []string{
		"-i", inputPath,
	}

	// Apply volume adjustment
	if options.Volume != 1.0 {
		args = append(args, "-af", fmt.Sprintf("volume=%.2f", options.Volume))
	}

	// Apply fade in/out effects
	if options.FadeIn || options.FadeOut {
		filter := ""
		if options.FadeIn {
			filter += "fade=t=in:st=0:d=2,"
		}
		if options.FadeOut {
			// Get duration first to calculate fade out start time
			durationCmd := exec.Command("ffprobe", "-v", "quiet", "-show_entries", "format=duration", "-of", "csv=p=0", inputPath)
			durationOutput, err := durationCmd.Output()
			if err != nil {
				log.Printf("Failed to get audio duration: %v", err)
				return fmt.Errorf("failed to get audio duration: %v", err)
			}

			durationStr := strings.TrimSpace(string(durationOutput))
			duration, err := strconv.ParseFloat(durationStr, 64)
			if err != nil {
				log.Printf("Failed to parse audio duration: %v", err)
				return fmt.Errorf("failed to parse audio duration: %v", err)
			}

			fadeOutStart := duration - 2.0 // 2 second fade out
			if fadeOutStart < 0 {
				fadeOutStart = 0
			}
			filter += fmt.Sprintf("fade=t=out:st=%.2f:d=2", fadeOutStart)
		}

		// Remove trailing comma if both fade in and out
		if options.FadeIn && options.FadeOut {
			filter = strings.TrimSuffix(filter, ",")
		}

		args = append(args, "-af", filter)
	}

	// Output settings
	args = append(args,
		"-c:a", "aac",
		"-b:a", "128k",
		outputPath,
	)

	log.Printf("Running ffmpeg audio processing command: ffmpeg %v", args)
	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("ffmpeg audio processing failed: %v, output: %s", err, string(output))
		return fmt.Errorf("ffmpeg audio processing failed: %v, output: %s", err, string(output))
	}

	log.Printf("Audio processing completed successfully: %s", outputPath)
	return nil
}

func mergeVideosWithAudio(videoFiles, audioFiles []string, outputPath, outputSize string, fps int) error {
	log.Printf("Merging %d videos with %d audio files to %s", len(videoFiles), len(audioFiles), outputPath)

	if len(videoFiles) == 0 {
		log.Printf("No video files provided for merge")
		return fmt.Errorf("no video files provided")
	}

	// First, merge videos without audio
	tempVideoPath := strings.TrimSuffix(outputPath, ".mp4") + "_temp_video.mp4"
	if err := mergeVideos(videoFiles, tempVideoPath, outputSize, fps); err != nil {
		log.Printf("Failed to merge videos: %v", err)
		return fmt.Errorf("failed to merge videos: %v", err)
	}
	defer os.Remove(tempVideoPath)

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

	// Build ffmpeg command to merge video with audio
	args := []string{
		"-i", tempVideoPath,
	}

	// Add audio inputs
	for _, audioFile := range audioFiles {
		args = append(args, "-i", audioFile)
	}

	// Build complex filter for mixing audio
	if len(audioFiles) > 0 {
		filter := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2", width, height, width, height)

		// Mix all audio tracks
		audioMix := ""
		for i := range audioFiles {
			if i > 0 {
				audioMix += "+"
			}
			audioMix += fmt.Sprintf("[%d:a]", i+1) // +1 because first input is video
		}
		if audioMix != "" {
			filter += fmt.Sprintf(";[%s]amix=inputs=%d:duration=longest", audioMix, len(audioFiles))
		}

		args = append(args, "-filter_complex", filter)
	}

	// Output settings
	args = append(args,
		"-c:v", "libx264",
		"-crf", "23",
		"-preset", "medium",
		"-r", fmt.Sprintf("%d", fps),
		"-c:a", "aac",
		"-b:a", "128k",
		outputPath,
	)

	log.Printf("Running ffmpeg merge with audio command: ffmpeg %v", args)
	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("ffmpeg merge with audio failed: %v, output: %s", err, string(output))
		return fmt.Errorf("ffmpeg merge with audio failed: %v, output: %s", err, string(output))
	}

	log.Printf("Videos merged with audio successfully: %s", outputPath)
	return nil
}

func normalizeVideoFPS(inputPath, outputPath string, targetFPS int) error {
	log.Printf("Normalizing video FPS: %s -> %s (target: %d fps)", inputPath, outputPath, targetFPS)

	// Build ffmpeg command to normalize FPS
	args := []string{
		"-i", inputPath,
		"-vf", fmt.Sprintf("fps=fps=%d:round=up", targetFPS),
		"-c:v", "libx264",
		"-crf", "23",
		"-preset", "medium",
		"-c:a", "aac",
		"-b:a", "128k",
		outputPath,
	}

	log.Printf("Running ffmpeg FPS normalization command: ffmpeg %v", args)
	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("ffmpeg FPS normalization failed: %v, output: %s", err, string(output))
		return fmt.Errorf("ffmpeg FPS normalization failed: %v, output: %s", err, string(output))
	}

	log.Printf("Video FPS normalized successfully: %s", outputPath)
	return nil
}
