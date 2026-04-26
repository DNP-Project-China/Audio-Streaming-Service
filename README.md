# 🎵 Anonymous Audio Streaming Service

A distributed system for anonymous music streaming with real-time stream analytics. Developed as a course project for **Distributed and Network Programming (DNP)**.

This system allows users to upload audio files (MP3, FLAC, WAV), which are automatically transcoded into HLS format for optimal streaming. A key feature of the project is entirely anonymous playback, utilizing Apache Kafka to collect real-time streaming statistics (live listener counts and dynamic top charts).

## 🏗 Architecture & Technology Stack

The project is built on a microservices architecture, heavily utilizing an event-driven approach for asynchronous communication:

* **Core API (Go):** The primary entry point for managing metadata and uploading original files. Implements the **Saga pattern** (compensating transactions) to ensure distributed fault tolerance and eventual consistency across PostgreSQL, S3, and Kafka.
* **Audio Converter (Python):** A **stateless** worker service for CPU-bound audio transcoding (FFmpeg) and uploading it to S3 database. Designed for horizontal scalability, it ensures **At-Least-Once delivery** semantics working with Kafka and prevents race conditions using **Optimistic Locking** at the Postgre SQL database level.
* **Playback API (Python/FastAPI):** Manages audio streaming sessions. Utilizes a **stateless session management** pattern via **Redis Keyspace Notifications** (distributed TTL) to detect client timeouts without running heavy in-memory polling loops.
* **Statistics API (Python/FastAPI):** The real-time stream analytics engine. Based on Redis soted set updates music current online and total plays in time. Additionally implements the **Write-Behind Caching** pattern (aggregating state deltas in Redis (by using hashes) before batch-flushing to PostgreSQL) to protect the persistent storage from high I/O write loads during heavy streaming.
* **Frontend (React/Vite):** Client-side Single Page Application (SPA).
* **Infrastructure:** Apache Kafka (Message Broker), PostgreSQL (Relational DB), Redis (In-memory Cache/Session Store), MinIO (S3-compatible Object Storage), Docker & Docker Compose.

---
## 🌐 Try Out

You can try our service inside Innopolis University network: http://10.90.138.33

---

## 🚀 Installation & Setup

The project is fully containerized. You will need **Docker** and **Docker Compose** installed on your machine.

### Step 1. Clone the repository
Clone the repository to your local machine:
```bash
git clone https://github.com/DNP-Project-China/Audio-Streaming-Service.git
cd Audio-Streaming-Service
```

### Step 2. Configure Environment Variables
In the root directory (and inside microservice folders if necessary), locate the `.env.example` file. Copy and rename it to `.env`. The default settings are already configured for a local Docker deployment. You still cannot run the project unless you set up an S3 server by your own.
```bash
cp .env.example .env
```

### Step 3. Launch the System
Spin up the entire infrastructure (databases, broker) and all microservices with a single command:
```bash
docker-compose up --build -d
```
*The `-d` flag runs the containers in detached mode. Omit it if you want to see the service logs in real-time.*

Please wait for the services to initialize (around 10-30 seconds, as Kafka and PostgreSQL require some startup time).

---

## 🎧 Usage Guide (after localhost setup)

Once successfully launched, the system is ready to use.

### 1. Web Interface (Frontend)
Open your browser and navigate to: **http://localhost:80**
Full functionality is available here:
* **Upload Track:** Click the "Upload" button, enter the title and artist, and select an audio file.
* **Playback:** The main page displays a list of tracks. OAfter some time from uploading, the track and its ability to play will appear.
* **Statistics:** Navigate to the "Top 24 hours" and "Listening now" to view total play counts and real-time online listeners.

### 2. Uploading via API (curl)
If you prefer to test the system via the terminal, send a POST request to the Core API:
```bash
curl -X POST http://localhost:8000/upload \
  -F "artist=Hans Zimmer" \
  -F "title=Interstellar Theme" \
  -F "file=@/path/to/your/audio.mp3"
```
You will receive a JSON response containing the `track_id` and the `pending` status.

### 3. Monitoring Stream Analytics
1.  Start playing any displayed track in the browser.
2.  At this moment, the Playback API publishes a `playback.started` event to Kafka.
3.  Open a second browser tab and go to the statistics page — you will see the "Online Now" counter increment by 1.
4.  When you close the tab or stop the player, the system triggers a `playback.stopped` event (or detects a timeout via Redis), and the user is removed from the track's live online counter.

---

## 🛠 Teardown

To stop all services and remove the containers cleanly, run:
`bash
docker-compose down
`
*(Append the `-v` flag if you want to delete the data volumes and completely wipe the databases: `docker-compose down -v`).*
