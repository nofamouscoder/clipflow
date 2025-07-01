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
	File    interface{}  `json:"file"`
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
	File interface{} `json:"file"`
}

type TaskResponse struct {
	TaskID  string `json:"taskId"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type TaskStatus struct {
	ID          string     `json:"id"`
	UserID      string     `json:"userId"`
	Status      string     `json:"status"`
	Progress    int        `json:"progress"`
	Message     string     `json:"message"`
	OutputFile  string     `json:"outputFile,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

var tasks = make(map[string]*TaskStatus)
var workDir = "./temp"
var outputDir = "./output"

func main() {
	// Create necessary directories
	os.MkdirAll(workDir, 0755)
	os.MkdirAll(outputDir, 0755)

	router := gin.Default()

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	router.Use(cors.New(config))

	// Serve static files
	router.Static("/output", "./output")

	// API routes
	api := router.Group("/api")
	{
		api.POST("/generate-video", generateVideoHandler)
		api.GET("/task/:taskId", getTaskStatusHandler)
		api.GET("/tasks/:userId", getUserTasksHandler)
	}

	log.Println("Server starting on :8080")
	router.Run(":8080")
}

func generateVideoHandler(c *gin.Context) {
	var req VideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate required fields
	if req.UserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userID is required"})
		return
	}

	if len(req.Videos) == 0 && len(req.YouTube) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one video or YouTube clip is required"})
		return
	}

	// Create task
	taskID := uuid.New().String()
	task := &TaskStatus{
		ID:        taskID,
		UserID:    req.UserID,
		Status:    "pending",
		Progress:  0,
		Message:   "Task created, preparing for processing",
		CreatedAt: time.Now(),
	}

	tasks[taskID] = task

	// Start processing in goroutine
	go processVideoRequest(taskID, req)

	c.JSON(http.StatusOK, TaskResponse{
		TaskID:  taskID,
		Status:  "pending",
		Message: "Video generation task created successfully",
	})
}

func getTaskStatusHandler(c *gin.Context) {
	taskID := c.Param("taskId")
	task, exists := tasks[taskID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func getUserTasksHandler(c *gin.Context) {
	userID := c.Param("userId")
	var userTasks []*TaskStatus

	for _, task := range tasks {
		if task.UserID == userID {
			userTasks = append(userTasks, task)
		}
	}

	c.JSON(http.StatusOK, userTasks)
}

func processVideoRequest(taskID string, req VideoRequest) {
	task := tasks[taskID]
	task.Status = "processing"
	task.Message = "Starting video processing"
	task.Progress = 10

	// Create temporary directory for this task
	taskDir := filepath.Join(workDir, taskID)
	os.MkdirAll(taskDir, 0755)
	defer os.RemoveAll(taskDir) // Clean up after processing

	var videoFiles []string
	videoIndex := 0

	// Process YouTube videos
	for _, ytClip := range req.YouTube {
		task.Message = fmt.Sprintf("Downloading YouTube video: %s", ytClip.URL)
		task.Progress = 20 + (videoIndex * 10)

		for _, segment := range ytClip.Segments {
			fileName := fmt.Sprintf("yt_%d_%s.mp4", segment.Index, generateFileHash(ytClip.URL+segment.Timeline.Start+segment.Timeline.End))
			outputPath := filepath.Join(taskDir, fileName)

			if err := downloadYouTubeSegment(ytClip.URL, ytClip.Quality, segment.Timeline, outputPath); err != nil {
				task.Status = "failed"
				task.Message = fmt.Sprintf("Failed to download YouTube video: %v", err)
				return
			}

			// Apply segment options (slowmotion, mute)
			if segment.Options.Slowmotion || segment.Options.Mute {
				processedPath := filepath.Join(taskDir, fmt.Sprintf("processed_%s", fileName))
				if err := applyVideoEffects(outputPath, processedPath, segment.Options.Slowmotion, segment.Options.Mute); err != nil {
					task.Status = "failed"
					task.Message = fmt.Sprintf("Failed to apply effects: %v", err)
					return
				}
				os.Remove(outputPath)
				os.Rename(processedPath, outputPath)
			}

			videoFiles = append(videoFiles, outputPath)
			videoIndex++
		}
	}

	// TODO: Process uploaded video files (would need multipart form handling)
	// For now, this is a placeholder for file uploads

	if len(videoFiles) == 0 {
		task.Status = "failed"
		task.Message = "No video files to process"
		return
	}

	// Merge videos
	task.Message = "Merging videos"
	task.Progress = 70

	outputFileName := fmt.Sprintf("merged_%s_%s.mp4", req.UserID, taskID)
	outputPath := filepath.Join(outputDir, outputFileName)

	if err := mergeVideos(videoFiles, outputPath, req.OutputSize); err != nil {
		task.Status = "failed"
		task.Message = fmt.Sprintf("Failed to merge videos: %v", err)
		return
	}

	// Complete task
	now := time.Now()
	task.Status = "completed"
	task.Progress = 100
	task.Message = "Video processing completed successfully"
	task.OutputFile = fmt.Sprintf("/output/%s", outputFileName)
	task.CompletedAt = &now
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

