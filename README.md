# a2web - Go Audio Streamer

A minimal, real-time audio streaming application built with Go. This application allows a user to "broadcast" live microphone audio from one device (e.g., a child's playroom) to another device (e.g., a parent's dashboard) via WebSockets. It includes authentication, session management, and audio recording.

## Features
- **Real-time Audio Streaming**: Low-latency streaming using WebSockets and the MediaRecorder API (WebM/Opus).
- **Secure Authentication**: User registration and login using JWT (stored in HTTP-only cookies).
- **Session Management**: Users can create unique streaming sessions.
- **Audio Recording**: All streamed audio is automatically saved to the server storage.
- **Admin Panel**: Dashboard identifying users and sessions, with deletion capabilities.
- **Dockerized**: specific for production deployment.

## Tech Stack
- **Backend**: Go (Golang) standard library + `gorilla/websocket`, `mattn/go-sqlite3`, `golang-jwt/jwt`
- **Database**: SQLite3
- **Frontend**: Server-side rendered HTML templates + [HTMX](https://htmx.org/) + [DaisyUI](https://daisyui.com/) (Tailwind CSS)
- **Deployment**: Docker (Alpine based)

## Prerequisites
- Docker & Docker Compose OR Go 1.25+
- GCC (if running locally, for SQLite CGO)

## Getting Started

### Using Docker (Recommended)
1. **Build and Run**:
   ```bash
   docker compose up --build -d
   ```
2. **Access**:
   Open browser at [http://localhost:8080](http://localhost:8080).

### Running Locally
1. **Install Dependencies**:
   ```bash
   go mod download
   ```
2. **Run Server**:
   ```bash
   go run cmd/server/main.go
   ```
   The server will start on port `8080`.

### Deployment with SSL (HTTPS)
This project uses **Caddy** to automatically handle SSL certificates.

1.  **Prerequisites**:
    -   Ensure your domain (e.g., `apis.imzami.com`) points to your server's IP.
    -   Ensure ports **80** and **443** are open on your firewall.

2.  **Deploy**:
    ```bash
    docker compose up -d
    ```

Caddy will automatically obtain a certificate from Let's Encrypt and serve the app securely at `https://apis.imzami.com`.

## Configuration
Create a `.env` file (or set environment variables) to configure the application:

| Variable | Description | Default |
| :--- | :--- | :--- |
| `PORT` | Server listening port | `8080` |
| `ALLOWED_ORIGIN` | Allowed Origin for WebSockets/CORS | `*` (dev) or `https://apis.imzami.com` (prod) |
| `JWT_SECRET` | Secret key for session tokens | `secret` |
| `DB_PATH` | Path to SQLite database | `./storage/audio_streamer.db` |


## Usage Guide
1. **Register**: Go to `/register-page` to create an account.
2. **Login**: Login with your mobile credentials.
3. **Create Session**: On the Dashboard, click "Create Session".
4. **Start Stream**: 
   - Share the **Kids Link** (`/kids/{id}`) with the broadcasting device.
   - **Auto-Start**: The page will automatically request microphone permission and start streaming immediately. No login required.
5. **Listen**:
   - Open the Session link (`/user/{id}`) on the listening device.
   - Audio will play automatically (you may need to interact with the page first due to browser autoplay policies).

## Directory Structure
- `cmd/server`: Entry point.
- `internal/`: Application logic (Auth, Handlers, Database, Models).
- `web/templates`: HTML templates.
- `storage`: SQLite database and recorded audio files.

## API Endpoints
- `POST /register`: Register user.
- `POST /login`: Login user.
- `GET /ws/kid/{id}`: WebSocket for sending audio.
- `GET /ws/parent/{id}`: WebSocket for receiving audio.

## License
MIT
