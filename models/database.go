package models

import (
	"database/sql"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Don't expose password hash in JSON
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Task struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Status      string     `json:"status"`
	Progress    int        `json:"progress"`
	Message     string     `json:"message"`
	OutputFile  string     `json:"output_file,omitempty"`
	TaskDetails string     `json:"task_details,omitempty"` // JSON string containing input videos, options, etc.
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type Database struct {
	db *sql.DB
}

func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Create tables if they don't exist
	if err := createTables(db); err != nil {
		return nil, err
	}

	return &Database{db: db}, nil
}

func createTables(db *sql.DB) error {
	// Users table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Tasks table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			progress INTEGER DEFAULT 0,
			message TEXT,
			output_file TEXT,
			task_details TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)

	// Add task_details column if it doesn't exist (for existing databases)
	_, err = db.Exec(`ALTER TABLE tasks ADD COLUMN task_details TEXT`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}

	// Create indexes
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks(user_id)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`)
	if err != nil {
		return err
	}

	return nil
}

// User methods
func (d *Database) CreateUser(user *User) error {
	_, err := d.db.Exec(`
		INSERT INTO users (id, email, username, password_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, user.ID, user.Email, user.Username, user.PasswordHash, user.CreatedAt, user.UpdatedAt)
	return err
}

func (d *Database) GetUserByID(id string) (*User, error) {
	user := &User{}
	err := d.db.QueryRow(`
		SELECT id, email, username, password_hash, created_at, updated_at
		FROM users WHERE id = ?
	`, id).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (d *Database) GetUserByEmail(email string) (*User, error) {
	user := &User{}
	err := d.db.QueryRow(`
		SELECT id, email, username, password_hash, created_at, updated_at
		FROM users WHERE email = ?
	`, email).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (d *Database) UpdateUser(user *User) error {
	_, err := d.db.Exec(`
		UPDATE users SET email = ?, username = ?, password_hash = ?, updated_at = ?
		WHERE id = ?
	`, user.Email, user.Username, user.PasswordHash, time.Now(), user.ID)
	return err
}

// Task methods
func (d *Database) CreateTask(task *Task) error {
	_, err := d.db.Exec(`
		INSERT INTO tasks (id, user_id, status, progress, message, output_file, task_details, created_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, task.ID, task.UserID, task.Status, task.Progress, task.Message, task.OutputFile, task.TaskDetails, task.CreatedAt, task.CompletedAt)
	return err
}

func (d *Database) GetTaskByID(id string) (*Task, error) {
	task := &Task{}
	err := d.db.QueryRow(`
		SELECT id, user_id, status, progress, message, output_file, task_details, created_at, completed_at
		FROM tasks WHERE id = ?
	`, id).Scan(&task.ID, &task.UserID, &task.Status, &task.Progress, &task.Message, &task.OutputFile, &task.TaskDetails, &task.CreatedAt, &task.CompletedAt)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (d *Database) GetTasksByUserID(userID string) ([]*Task, error) {
	rows, err := d.db.Query(`
		SELECT id, user_id, status, progress, message, output_file, task_details, created_at, completed_at
		FROM tasks WHERE user_id = ? ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		task := &Task{}
		err := rows.Scan(&task.ID, &task.UserID, &task.Status, &task.Progress, &task.Message, &task.OutputFile, &task.TaskDetails, &task.CreatedAt, &task.CompletedAt)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	if tasks == nil {
		tasks = make([]*Task, 0)
	}
	return tasks, nil
}

func (d *Database) UpdateTask(task *Task) error {
	_, err := d.db.Exec(`
		UPDATE tasks SET status = ?, progress = ?, message = ?, output_file = ?, task_details = ?, completed_at = ?
		WHERE id = ?
	`, task.Status, task.Progress, task.Message, task.OutputFile, task.TaskDetails, task.CompletedAt, task.ID)
	return err
}

func (d *Database) DeleteTask(id string) error {
	_, err := d.db.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	return err
}

func (d *Database) Close() error {
	return d.db.Close()
}
