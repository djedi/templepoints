# Temple Points Tracker 🏆

A fun, gamified web application to help LDS youth stakes track temple ordinance points as wards race to 1300 points. Features real-time updates, achievements, and an engaging interface designed specifically for young men and young women.

## 🎯 Overview

This application was built for a stake competition where 7 wards compete to reach 1300 temple ordinance points. Youth submit their temple work points, ward leaders approve them, and everyone can watch the real-time leaderboard with fun animations and achievements.

### Participating Wards

- Fountain Green 1st, 2nd, and 3rd Wards
- Moroni 1st, 2nd, and 3rd Wards
- Sanpitch Ward

## ✨ Features

### For Youth

- 🎮 **Gamified Experience** - Progress bars, achievements, animations, and confetti celebrations
- 📱 **Mobile-First Design** - Optimized for phones with responsive layouts
- 🏅 **Achievement System** - Unlock badges for milestones (First to 500, Week Champion, etc.)
- 🔥 **Streak Tracking** - See which wards are on fire with consistent participation
- 📊 **Real-time Updates** - Watch points update live via WebSockets

### For Leaders

- 🛡️ **Admin Dashboard** - Approve or reject point submissions
- 👥 **Ward-Specific Access** - Ward leaders only see their ward's submissions
- 📈 **Statistics** - Track total points, participation, and days active
- 🔐 **Secure Authentication** - Password-protected admin areas

### Technical Features

- ⚡ **Lightning Fast** - Built with Go for optimal performance
- 💾 **SQLite Database** - Simple, reliable data storage
- 🔄 **WebSocket Support** - Real-time updates without refreshing
- 🐳 **Docker Ready** - Easy deployment with Docker Compose
- 🔒 **Auto-SSL** - Caddy provides automatic HTTPS certificates

## 🚀 Quick Start

### Option 1: Docker Deployment (Recommended)

#### Prerequisites

- Docker and Docker Compose installed
- Ports 80 and 443 available

#### Deploy

```bash
# Clone the repository
git clone [your-repo-url]
cd templepoints

# Make deploy script executable
chmod +x deploy

# Run deployment
./deploy
```

The application will be available at `http://localhost`

### Option 2: Local Development

#### Prerequisites

- Go 1.21 or higher
- SQLite3
- Git

#### Setup

```bash
# Clone the repository
git clone [your-repo-url]
cd templepoints

# Download Go dependencies
go mod download

# Run the application
go run .
```

The application will be available at `http://localhost:8080`

## 🔧 Development Setup

### Project Structure

```
templepoints/
├── main.go              # Server initialization and routing
├── models.go            # Data structures and types
├── database.go          # Database setup and migrations
├── handlers.go          # HTTP request handlers
├── leaderboard.html     # Main leaderboard page
├── submit-points.html   # Points submission form
├── login.html           # Admin login page
├── admin.html           # Admin dashboard
├── go.mod               # Go module definition
├── go.sum               # Go module checksums
├── Dockerfile           # Container definition
├── docker-compose.yml   # Multi-container orchestration
├── Caddyfile           # Reverse proxy configuration
├── deploy              # Deployment script
└── README.md           # This file
```

### Running Locally

1. **Start the server:**

```bash
go run .
```

2. **Build for production:**

```bash
go build -o templepoints
./templepoints
```

3. **Run tests:**

```bash
go test ./...
```

### Database Management

The SQLite database (`templepoints.db`) is created automatically on first run with:

- 7 pre-configured wards
- Default admin account
- Sample data for demonstration

To reset the database:

```bash
rm templepoints.db
go run .  # Database will be recreated
```

### Making Changes

1. **Frontend changes:** Edit the HTML files directly
2. **Backend changes:** Modify Go files and restart the server
3. **Database schema:** Update `database.go` and delete the database file to recreate

## 🌐 Production Deployment

### Using Docker Compose

1. **Configure your domain:**
   Edit `Caddyfile` and replace `templepoints.example.com` with your actual domain:

```caddyfile
yourdomain.com {
    reverse_proxy app:8080
    # ... rest of config
}
```

2. **Deploy with Docker Compose:**

```bash
docker-compose up -d
```

3. **View logs:**

```bash
docker-compose logs -f
```

4. **Stop services:**

```bash
docker-compose down
```

### Manual Deployment

1. **Build the binary:**

```bash
CGO_ENABLED=1 go build -o templepoints
```

2. **Copy files to server:**

- Binary: `templepoints`
- HTML files: `*.html`
- Create data directory: `mkdir data`

3. **Run with systemd (example):**

```ini
[Unit]
Description=Temple Points Tracker
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/templepoints
ExecStart=/opt/templepoints/templepoints
Restart=on-failure
Environment="PORT=8080"

[Install]
WantedBy=multi-user.target
```

4. **Configure reverse proxy (nginx/caddy/apache)**

### Environment Variables

- `PORT` - Server port (default: 8080)
- `DATABASE_PATH` - SQLite database location (default: ./templepoints.db)

## 🔐 Security

### Default Credentials

**Admin Account:**

- Email: `admin@templepoints.org`
- Password: `admin123`

⚠️ **IMPORTANT:** Change these immediately in production!

### Creating New Admin Users

Currently, new admin users must be added directly to the database:

```sql
INSERT INTO users (email, password, role, ward_id)
VALUES ('newemail@example.com', '$2a$10$[bcrypt-hash]', 'ward_approver', 1);
```

### Security Best Practices

1. **Change default passwords immediately**
2. **Use HTTPS in production** (Caddy handles this automatically)
3. **Regular database backups:**

```bash
cp data/templepoints.db data/backup-$(date +%Y%m%d).db
```

4. **Monitor logs for suspicious activity**
5. **Keep Go dependencies updated:**

```bash
go get -u ./...
go mod tidy
```

## 📊 Point Values

The application tracks temple ordinance points with these suggested values:

- Baptism/Confirmation: 1 point each
- Initiatory: 1 point
- Endowment: 1 point
- Sealing: 1 point
- Family names: Double points!

Ward leaders can adjust these values when approving submissions.

## 🐛 Troubleshooting

### Common Issues

**Port already in use:**

```bash
# Find process using port 8080
lsof -i :8080
# Kill the process
kill -9 [PID]
```

**Database locked:**

```bash
# Stop the application
# Remove lock file if it exists
rm templepoints.db-shm templepoints.db-wal
# Restart application
```

**Docker build fails:**

```bash
# Clean Docker cache
docker system prune -a
# Rebuild
docker-compose build --no-cache
```

### Logs

**Application logs:**

```bash
# Docker
docker-compose logs app

# Local development
go run . 2>&1 | tee app.log
```

## 🤝 Contributing

This is a private project for a specific stake. If you're part of the stake and want to contribute:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## 📝 API Documentation

### Public Endpoints

#### Get Leaderboard

```
GET /api/leaderboard?sort=verified-desc
```

Sorting options: `verified-desc`, `verified-asc`, `total-desc`, `total-asc`, `ward-asc`, `ward-desc`

#### Submit Points

```
POST /api/points
Content-Type: application/json

{
    "ward_id": 1,
    "submitter_name": "John Doe",
    "points": 5,
    "note": "Family baptisms"
}
```

### Protected Endpoints (Requires Authentication)

#### Login

```
POST /api/login
Content-Type: application/json

{
    "email": "admin@templepoints.org",
    "password": "admin123"
}
```

#### Approve Points

```
POST /api/points/{id}/approve
Cookie: session=...
```

#### Reject Points

```
POST /api/points/{id}/reject
Cookie: session=...
```

#### Get Submissions

```
GET /api/submissions?status=pending
Cookie: session=...
```

### WebSocket

Connect to `/ws` for real-time updates:

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  // Handle updates
};
```

## ⚖️ Disclaimer

This website is not affiliated with, endorsed by, or sponsored by The Church of Jesus Christ of Latter-day Saints. All content and functionality are independently created and maintained.

## 📄 License

This is a private project created for a specific LDS stake. All rights reserved.

## 🆘 Support

For questions or issues, please contact your stake technology specialist or create an issue in the repository.

---

Built with ❤️ to help youth engage with temple work
