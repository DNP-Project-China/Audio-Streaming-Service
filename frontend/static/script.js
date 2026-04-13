let sessionId = localStorage.getItem('sessionId');
if (!sessionId) {
    sessionId = crypto.randomUUID();
    localStorage.setItem('sessionId', sessionId);
}

let currentTrackId = null;
let pingInterval = null;
let activeAudioElement = null;

const API_BASE = '/api';
const TRACKING_BASE = '/tracking';
const STATS_BASE = '/stats';

const fileInput = document.getElementById('fileInput');
const uploadBtn = document.getElementById('uploadBtn');
const uploadStatus = document.getElementById('uploadStatus');
const trackListEl = document.getElementById('trackList');
const chartListEl = document.getElementById('chartList');

uploadBtn.addEventListener('click', async () => {
    const file = fileInput.files[0];
    if (!file) {
        uploadStatus.textContent = 'Select a file!';
        return;
    }
    uploadStatus.textContent = 'Uploading...';
    const formData = new FormData();
    formData.append('file', file);
    try {
        const res = await fetch(`${API_BASE}/upload`, { method: 'POST', body: formData });
        if (res.ok) {
            uploadStatus.textContent = '✅ Track uploaded, processing...';
            fileInput.value = '';
            setTimeout(loadTracks, 2000);
        } else {
            uploadStatus.textContent = '❌ Upload failed';
        }
    } catch (err) {
        uploadStatus.textContent = '❌ Network error';
        console.error(err);
    }
});

async function loadTracks() {
    try {
        const res = await fetch(`${API_BASE}/tracks`);
        const tracks = await res.json();
        renderTracks(tracks);
    } catch (err) {
        console.error('Failed to load tracks:', err);
        trackListEl.innerHTML = '<li>Unable to load tracks (API not ready)</li>';
    }
}

async function loadChart() {
    try {
        const res = await fetch(`${STATS_BASE}/chart`);
        const chart = await res.json();
        renderChart(chart);
    } catch (err) {
        console.error('Failed to load chart:', err);
        chartListEl.innerHTML = '<li>No data</li>';
    }
}

function renderTracks(tracks) {
    if (!tracks.length) {
        trackListEl.innerHTML = '<li>No tracks uploaded yet</li>';
        return;
    }
    trackListEl.innerHTML = tracks.map(track => `
        <li data-track-id="${track.id}">
            <span class="track-title">${escapeHtml(track.filename)}</span>
            <span class="fire" id="fire-${track.id}">🔥 0</span>
            <div class="player">
                <button class="play-btn" data-id="${track.id}">▶️ Play</button>
                <button class="pause-btn" data-id="${track.id}" style="display:none;">⏸️ Pause</button>
                <audio id="audio-${track.id}" preload="none"></audio>
            </div>
            <button class="download-btn" data-id="${track.id}">⬇️ Download</button>
        </li>
    `).join('');
    document.querySelectorAll('.play-btn').forEach(btn => {
        btn.addEventListener('click', () => playTrack(btn.dataset.id));
    });
    document.querySelectorAll('.pause-btn').forEach(btn => {
        btn.addEventListener('click', () => pauseTrack(btn.dataset.id));
    });
    document.querySelectorAll('.download-btn').forEach(btn => {
        btn.addEventListener('click', () => downloadTrack(btn.dataset.id));
    });
}

function renderChart(chart) {
    if (!chart || !chart.length) {
        chartListEl.innerHTML = '<li>No chart data</li>';
        return;
    }
    chartListEl.innerHTML = chart.map(item => `<li>${escapeHtml(item.title)} — ${item.plays} plays</li>`).join('');
}

function playTrack(trackId) {
    if (currentTrackId === trackId && activeAudioElement && !activeAudioElement.paused) return;
    if (activeAudioElement) {
        activeAudioElement.pause();
        if (pingInterval) clearInterval(pingInterval);
        sendStopEvent(currentTrackId);
    }
    currentTrackId = trackId;
    const audio = document.getElementById(`audio-${trackId}`);
    const playBtn = document.querySelector(`.play-btn[data-id="${trackId}"]`);
    const pauseBtn = document.querySelector(`.pause-btn[data-id="${trackId}"]`);
    if (!audio) return;
    const hlsUrl = `${API_BASE}/stream/${trackId}/master.m3u8`;
    if (Hls.isSupported()) {
        const hls = new Hls();
        hls.loadSource(hlsUrl);
        hls.attachMedia(audio);
        hls.on(Hls.Events.MANIFEST_PARSED, () => {
            audio.play();
            playBtn.style.display = 'none';
            pauseBtn.style.display = 'inline-block';
            startPing(trackId);
        });
        window.hls = hls;
    } else if (audio.canPlayType('application/vnd.apple.mpegurl')) {
        audio.src = hlsUrl;
        audio.play();
        playBtn.style.display = 'none';
        pauseBtn.style.display = 'inline-block';
        startPing(trackId);
    }
    activeAudioElement = audio;
    audio.onpause = () => {
        if (currentTrackId === trackId) {
            sendStopEvent(trackId);
            if (pingInterval) clearInterval(pingInterval);
            playBtn.style.display = 'inline-block';
            pauseBtn.style.display = 'none';
            currentTrackId = null;
            activeAudioElement = null;
        }
    };
}

function pauseTrack(trackId) {
    const audio = document.getElementById(`audio-${trackId}`);
    if (audio) audio.pause();
}

function startPing(trackId) {
    if (pingInterval) clearInterval(pingInterval);
    pingInterval = setInterval(() => {
        fetch(`${TRACKING_BASE}/ping`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ track_id: trackId, session_id: sessionId, status: 'playing' })
        }).catch(e => console.error('Ping error', e));
    }, 10000);
}

function sendStopEvent(trackId) {
    if (!trackId) return;
    fetch(`${TRACKING_BASE}/stop`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ track_id: trackId, session_id: sessionId, status: 'stopped' })
    }).catch(e => console.error('Stop error', e));
}

async function downloadTrack(trackId) {
    try {
        const res = await fetch(`${API_BASE}/download/${trackId}`);
        const data = await res.json();
        if (data.download_url) window.open(data.download_url, '_blank');
    } catch (err) {
        console.error('Download error', err);
    }
}

async function updateFireCounters() {
    try {
        const res = await fetch(`${STATS_BASE}/live`);
        const counters = await res.json();
        for (const [trackId, count] of Object.entries(counters)) {
            const fireSpan = document.getElementById(`fire-${trackId}`);
            if (fireSpan) fireSpan.innerHTML = `🔥 ${count}`;
        }
    } catch (err) {
        console.error('Fire update error', err);
    }
}

function escapeHtml(str) {
    return str.replace(/[&<>]/g, function(m) {
        if (m === '&') return '&amp;';
        if (m === '<') return '&lt;';
        if (m === '>') return '&gt;';
        return m;
    });
}

setInterval(updateFireCounters, 3000);
setInterval(loadChart, 15000);
loadTracks();
loadChart();
updateFireCounters();

window.addEventListener('beforeunload', () => {
    if (currentTrackId) sendStopEvent(currentTrackId);
}); 
