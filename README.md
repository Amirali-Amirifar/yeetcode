# YeetCode

A LeetCode clone built with Go and PostgreSQL.

## Prerequisites
- Docker Desktop installed and running
- Go 1.24 or later
- Git

## Setup Instructions

### 1. Clone the Repository
```bash
git clone https://github.com/yourusername/yeetcode.git
cd yeetcode
```

### 2. Configure Environment Variables
Create a `.env` file in the root directory with the following content:
```env
POSTGRES_USER=admin
POSTGRES_PASSWORD=123456
POSTGRES_DB=yeetcode
PGADMIN_DEFAULT_EMAIL=admin@admin.com
PGADMIN_DEFAULT_PASSWORD=admin
```

### 3. Start Docker Containers
```bash
# Start the containers
docker-compose up -d

# Verify containers are running
docker ps
```

### 4. Access pgAdmin
- Open http://localhost:5050 in your browser
- Login credentials:
  - Email: admin@admin.com
  - Password: admin

### 5. Connect to PostgreSQL in pgAdmin
1. Right-click on "Servers" in the left sidebar
2. Select "Register" > "Server"
3. In the "General" tab:
   - Name: YeetCode DB
4. In the "Connection" tab:
   - Host: postgres
   - Port: 5432
   - Database: yeetcode
   - Username: admin
   - Password: 123456

### 6. Build and Run the Application
```bash
# Navigate to the backend directory
cd ./backend
# Install dependencies
go mod tidy

# Build the application
go build ./... 

# Run the application
go run ./...                                                   
```

The application will be available at http://localhost:8081

## Development
- Backend API runs on port 8081
- PostgreSQL runs on port 5432
- pgAdmin runs on port 5050

## Troubleshooting
1. If containers fail to start:
   ```bash
   # Stop all containers
   docker-compose down
   
   # Remove volumes
   docker-compose down -v
   
   # Start fresh
   docker-compose up -d
   ```

2. If you can't access pgAdmin:
   - Verify Docker Desktop is running
   - Check if port 5050 is available
   - Try restarting Docker Desktop

3. If database connection fails:
   - Verify PostgreSQL container is running
   - Check credentials in .env file
   - Ensure pgAdmin connection details are correct