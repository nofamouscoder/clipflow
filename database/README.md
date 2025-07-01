# Database Configuration for Clipflow

## Database Options

### 1. SQLite (Recommended for Development & Small Scale)
**Pros:**
- ✅ **Free** - No hosting costs
- ✅ **Zero configuration** - File-based database
- ✅ **Easy to maintain** - Single file, easy backups
- ✅ **No external dependencies** - Works offline
- ✅ **Perfect for development** - Fast setup and testing
- ✅ **Built-in Go support** - No additional drivers needed

**Cons:**
- ❌ Limited concurrent users (good for up to ~100 concurrent users)
- ❌ No built-in authentication (need to implement JWT)
- ❌ Manual backup management

**Best for:** Development, small projects, MVP, personal use

### 2. Firebase (Recommended for Production & Scale)
**Pros:**
- ✅ **Free tier** - Generous free limits
- ✅ **Built-in authentication** - Email/password, Google, Facebook, etc.
- ✅ **Real-time updates** - Automatic sync across clients
- ✅ **Scalable** - Handles millions of users
- ✅ **Managed service** - No server maintenance
- ✅ **Security rules** - Fine-grained access control
- ✅ **Analytics** - Built-in user analytics

**Cons:**
- ❌ Requires internet connection
- ❌ Vendor lock-in
- ❌ Learning curve for security rules
- ❌ Can be expensive at scale

**Best for:** Production apps, multi-user applications, when you need authentication

### 3. PostgreSQL (Alternative for Production)
**Pros:**
- ✅ **Free** - Open source
- ✅ **Powerful** - Full SQL support
- ✅ **Scalable** - Handles large datasets
- ✅ **ACID compliance** - Data integrity

**Cons:**
- ❌ Requires server setup and maintenance
- ❌ More complex configuration
- ❌ Need to implement authentication separately

## Current Setup

The project is configured to use **SQLite** by default, which is perfect for:
- Development and testing
- Small to medium user bases
- Quick prototyping
- Personal projects

## Switching to Firebase

If you want to use Firebase instead:

1. **Create a Firebase project:**
   - Go to [Firebase Console](https://console.firebase.google.com/)
   - Create a new project
   - Enable Authentication and Firestore Database

2. **Get Firebase credentials:**
   - Go to Project Settings > Service Accounts
   - Generate new private key
   - Download the JSON file

3. **Update environment variables:**
   ```bash
   # In .env file
   DB_TYPE=firebase
   FIREBASE_PROJECT_ID=your-project-id
   FIREBASE_PRIVATE_KEY=your-private-key
   FIREBASE_CLIENT_EMAIL=your-client-email
   ```

4. **Install Firebase Go SDK:**
   ```bash
   go get firebase.google.com/go/v4
   ```

## Database Schema

### Users Table
```sql
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Tasks Table
```sql
CREATE TABLE tasks (
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
```

## Migration Guide

### From SQLite to Firebase
1. Export data from SQLite:
   ```bash
   sqlite3 database/clipflow.db ".dump" > backup.sql
   ```

2. Set up Firebase and update code to use Firebase SDK

3. Import data to Firebase (write migration script)

### From Firebase to SQLite
1. Export data from Firebase:
   ```bash
   # Use Firebase Admin SDK to export data
   ```

2. Update environment to use SQLite

3. Import data to SQLite

## Backup Strategy

### SQLite Backup
```bash
# Create backup
cp database/clipflow.db database/clipflow_backup_$(date +%Y%m%d_%H%M%S).db

# Restore backup
cp database/clipflow_backup_20231201_143022.db database/clipflow.db
```

### Firebase Backup
- Use Firebase Admin SDK to export data
- Firebase automatically handles backups
- Use Firebase Console for manual exports

## Security Considerations

### SQLite
- Store database file securely
- Implement proper authentication
- Use HTTPS in production
- Regular backups

### Firebase
- Configure security rules properly
- Use Firebase Authentication
- Monitor usage and costs
- Set up proper user roles

## Recommendation

**For your current project, I recommend starting with SQLite because:**

1. **Quick setup** - No external dependencies
2. **Free** - No hosting costs
3. **Easy to maintain** - Single file database
4. **Perfect for development** - Fast iteration
5. **Easy to migrate later** - Can switch to Firebase when needed

You can always migrate to Firebase later when you need:
- User authentication
- Real-time updates
- Multi-user support
- Production scaling

The setup script creates a SQLite database by default, but includes Firebase configuration options in the `.env` file for future use. 