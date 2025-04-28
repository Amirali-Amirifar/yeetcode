# YeetCode

A coding platform built with Go and PostgreSQL.

## Prerequisites

- Docker and Docker Compose
- Go (latest version recommended)
- Node.js and npm (for frontend dependencies)

## Setup

1. Clone the repository:
```bash
git clone <repository-url>
cd yeetcode
```

2. Start the database services using Docker Compose:
```bash
docker-compose -f docker-compose-dev.yml up -d
```

3. Install frontend dependencies:
```bash
npm install
```

4. Build the frontend:
```bash
npm run build
```

## Running the Application

1. Start the Go backend server:
```bash
go run cmd/main.go
```

## Database Access

- PostgreSQL is running on port 5432
- pgAdmin is accessible at http://localhost:5050
  - Email: admin@admin.com
  - Password: admin123

## Environment Configuration

The application uses the following default database credentials:
- Database: yeetcode
- Username: admin
- Password: 123456

## Stopping Services

To stop all Docker services:
```bash
docker-compose -f docker-compose-dev.yml down
```

## License

[Add your license information here]