# Frontend Integration Guide

## Overview

The Clipflow API now supports both anonymous and authenticated access, making it easy to integrate with any frontend application. Users can start using the API immediately without registration, and optionally authenticate later for persistent sessions.

## Quick Start

### 1. Initialize User Session

The recommended way to start is by calling the `/api/me` endpoint:

```javascript
// Get or create user session
async function initializeUser() {
  const token = localStorage.getItem('clipflow_token');
  
  const response = await fetch('/api/me', {
    headers: token ? {
      'Authorization': `Bearer ${token}`
    } : {}
  });
  
  const data = await response.json();
  
  // Store the token
  localStorage.setItem('clipflow_token', data.token);
  
  return data;
}

// Usage
const userData = await initializeUser();
console.log('User:', userData.user);
console.log('Is new user:', userData.new);
```

### 2. Make API Calls

All API calls can be made with or without authentication:

```javascript
// With authentication (recommended for persistent sessions)
async function createVideoTask(formData) {
  const token = localStorage.getItem('clipflow_token');
  
  const response = await fetch('/api/generate-video', {
    method: 'POST',
    headers: token ? {
      'Authorization': `Bearer ${token}`
    } : {},
    body: formData
  });
  
  return response.json();
}

// Without authentication (anonymous access)
async function createVideoTaskAnonymous(formData) {
  const response = await fetch('/api/generate-video', {
    method: 'POST',
    body: formData
  });
  
  return response.json();
}
```

## Complete Integration Example

### React/JavaScript Example

```javascript
class ClipflowAPI {
  constructor() {
    this.baseURL = 'http://localhost:8080/api';
    this.token = localStorage.getItem('clipflow_token');
  }

  // Set authorization header
  getHeaders() {
    const headers = {};
    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }
    return headers;
  }

  // Initialize user session
  async initializeUser() {
    const response = await fetch(`${this.baseURL}/me`, {
      headers: this.getHeaders()
    });
    
    const data = await response.json();
    this.token = data.token;
    localStorage.setItem('clipflow_token', data.token);
    
    return data;
  }

  // Create video task
  async createVideoTask(videoConfig, videoFiles) {
    const formData = new FormData();
    
    // Add configuration
    formData.append('data', JSON.stringify(videoConfig));
    
    // Add video files
    videoFiles.forEach((file, index) => {
      formData.append(`video_${index}`, file);
    });
    
    const response = await fetch(`${this.baseURL}/generate-video`, {
      method: 'POST',
      headers: this.getHeaders(),
      body: formData
    });
    
    return response.json();
  }

  // Get task status
  async getTaskStatus(taskId) {
    const response = await fetch(`${this.baseURL}/task/${taskId}`, {
      headers: this.getHeaders()
    });
    
    return response.json();
  }

  // Get user tasks
  async getUserTasks() {
    const response = await fetch(`${this.baseURL}/tasks`, {
      headers: this.getHeaders()
    });
    
    return response.json();
  }

  // Delete task
  async deleteTask(taskId) {
    const response = await fetch(`${this.baseURL}/task/${taskId}`, {
      method: 'DELETE',
      headers: this.getHeaders()
    });
    
    return response.json();
  }

  // Register user
  async registerUser(email, username, password) {
    const response = await fetch(`${this.baseURL}/register`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ email, username, password })
    });
    
    const data = await response.json();
    if (data.token) {
      this.token = data.token;
      localStorage.setItem('clipflow_token', data.token);
    }
    
    return data;
  }

  // Login user
  async loginUser(email, password) {
    const response = await fetch(`${this.baseURL}/login`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ email, password })
    });
    
    const data = await response.json();
    if (data.token) {
      this.token = data.token;
      localStorage.setItem('clipflow_token', data.token);
    }
    
    return data;
  }

  // Logout
  logout() {
    this.token = null;
    localStorage.removeItem('clipflow_token');
  }
}
```

### Usage Example

```javascript
// Initialize the API
const clipflow = new ClipflowAPI();

// Initialize user session (creates anonymous user if needed)
const userData = await clipflow.initializeUser();
console.log('User initialized:', userData.user);

// Create a video task
const videoConfig = {
  outputsize: "16:9",
  videos: [
    {
      file: "video_0",
      options: {
        slowmotion: false,
        mute: false
      }
    }
  ]
};

const videoFiles = [/* File objects from input */];
const task = await clipflow.createVideoTask(videoConfig, videoFiles);
console.log('Task created:', task);

// Poll for task status
const checkStatus = async () => {
  const status = await clipflow.getTaskStatus(task.taskId);
  console.log('Task status:', status);
  
  if (status.status === 'completed') {
    console.log('Video ready:', status.output_file);
  } else if (status.status === 'failed') {
    console.log('Task failed:', status.message);
  } else {
    // Continue polling
    setTimeout(checkStatus, 2000);
  }
};

checkStatus();
```

## Vue.js Example

```vue
<template>
  <div>
    <div v-if="!user">
      <button @click="initializeUser">Start Using Clipflow</button>
    </div>
    
    <div v-else>
      <h2>Welcome, {{ user.username }}!</h2>
      <p v-if="isNewUser">New user created automatically</p>
      
      <div>
        <input type="file" @change="handleFileSelect" multiple accept="video/*">
        <button @click="createVideoTask" :disabled="!selectedFiles.length">
          Create Video Task
        </button>
      </div>
      
      <div v-if="currentTask">
        <h3>Current Task: {{ currentTask.status }}</h3>
        <p>Progress: {{ currentTask.progress }}%</p>
        <p>{{ currentTask.message }}</p>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      user: null,
      isNewUser: false,
      selectedFiles: [],
      currentTask: null,
      api: null
    }
  },
  
  async mounted() {
    this.api = new ClipflowAPI();
    await this.initializeUser();
  },
  
  methods: {
    async initializeUser() {
      try {
        const data = await this.api.initializeUser();
        this.user = data.user;
        this.isNewUser = data.new;
      } catch (error) {
        console.error('Failed to initialize user:', error);
      }
    },
    
    handleFileSelect(event) {
      this.selectedFiles = Array.from(event.target.files);
    },
    
    async createVideoTask() {
      try {
        const videoConfig = {
          outputsize: "16:9",
          videos: this.selectedFiles.map((_, index) => ({
            file: `video_${index}`,
            options: { slowmotion: false, mute: false }
          }))
        };
        
        const task = await this.api.createVideoTask(videoConfig, this.selectedFiles);
        this.currentTask = task;
        
        // Start polling for status
        this.pollTaskStatus(task.taskId);
      } catch (error) {
        console.error('Failed to create task:', error);
      }
    },
    
    async pollTaskStatus(taskId) {
      const checkStatus = async () => {
        try {
          const status = await this.api.getTaskStatus(taskId);
          this.currentTask = status;
          
          if (status.status === 'completed') {
            console.log('Video ready:', status.output_file);
          } else if (status.status === 'failed') {
            console.log('Task failed:', status.message);
          } else {
            setTimeout(checkStatus, 2000);
          }
        } catch (error) {
          console.error('Failed to check status:', error);
        }
      };
      
      checkStatus();
    }
  }
}
</script>
```

## Angular Example

```typescript
// clipflow.service.ts
import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { BehaviorSubject, Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class ClipflowService {
  private baseURL = 'http://localhost:8080/api';
  private token = localStorage.getItem('clipflow_token');
  private userSubject = new BehaviorSubject<any>(null);
  
  public user$ = this.userSubject.asObservable();

  constructor(private http: HttpClient) {}

  private getHeaders(): HttpHeaders {
    let headers = new HttpHeaders();
    if (this.token) {
      headers = headers.set('Authorization', `Bearer ${this.token}`);
    }
    return headers;
  }

  initializeUser(): Observable<any> {
    return this.http.get(`${this.baseURL}/me`, { headers: this.getHeaders() });
  }

  createVideoTask(videoConfig: any, videoFiles: File[]): Observable<any> {
    const formData = new FormData();
    formData.append('data', JSON.stringify(videoConfig));
    
    videoFiles.forEach((file, index) => {
      formData.append(`video_${index}`, file);
    });
    
    return this.http.post(`${this.baseURL}/generate-video`, formData, {
      headers: this.getHeaders()
    });
  }

  getTaskStatus(taskId: string): Observable<any> {
    return this.http.get(`${this.baseURL}/task/${taskId}`, {
      headers: this.getHeaders()
    });
  }

  getUserTasks(): Observable<any> {
    return this.http.get(`${this.baseURL}/tasks`, {
      headers: this.getHeaders()
    });
  }
}
```

## Key Benefits

1. **Zero Friction**: Users can start using the API immediately without registration
2. **Progressive Enhancement**: Optional authentication for persistent sessions
3. **Simple Integration**: Minimal setup required for frontend applications
4. **Flexible**: Works with any frontend framework or vanilla JavaScript
5. **Future Ready**: Easy to add Gmail login later

## Best Practices

1. **Always call `/api/me` first** to initialize the user session
2. **Store the token locally** for persistent sessions
3. **Handle token expiration** by calling `/api/me` again
4. **Use authentication headers** for better user experience
5. **Implement proper error handling** for network issues

## Error Handling

```javascript
async function handleAPIError(response) {
  if (!response.ok) {
    const error = await response.json();
    
    if (response.status === 401) {
      // Token expired or invalid, reinitialize user
      localStorage.removeItem('clipflow_token');
      await initializeUser();
      return;
    }
    
    throw new Error(error.error || 'API request failed');
  }
  
  return response.json();
}
```

This integration approach makes it very easy for frontend developers to get started with the Clipflow API while maintaining the flexibility to add proper authentication later. 