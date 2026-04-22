import { useState, useEffect, useRef } from 'react';
import { motion } from 'framer-motion';
import { BsPlayFill, BsPauseFill, BsDownload, BsFire, BsHeadphones, BsUpload, BsRewind, BsFastForward, BsSearch, BsActivity, BsVolumeUpFill } from 'react-icons/bs';
import Hls from 'hls.js';
import Top24 from './Top24';
import UploadModal from './UploadModal';
import ListeningNow from './ListeningNow';
import TrackPlayerModal from './TrackPlayerModal';

export default function Home() {
  const [tracks, setTracks] = useState([]);
  const [currentTrack, setCurrentTrack] = useState(null);
  const [listeners, setListeners] = useState({});
  const [plays, setPlays] = useState([]);
  const [isUploadModalOpen, setIsUploadModalOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [isPlayerModalOpen, setIsPlayerModalOpen] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [volume, setVolume] = useState(1);
  const [isPlaying, setIsPlaying] = useState(false);
  const audioRef = useRef(null);
  const hlsRef = useRef(null);
  const pingInterval = useRef(null);

  const loadTracks = async () => {
    try {
      const res = await fetch('/api/tracks');
      const data = await res.json();
      const mapped = data.items.map(track => ({
        id: track.track_id,
        filename: `${track.artist} - ${track.title}`,
        original_filename: track.original_filename,
        status: track.status
      }));
      setTracks(mapped);
    } catch (err) {
      console.error('Failed to load tracks', err);
      setTracks([]);
    }
  };

const loadStats = async () => {
  try {
    const res = await fetch('/stats/live');
    const data = await res.json();
    const listenersMap = {};
    const playsMap = {};

    if (data.items && Array.isArray(data.items)) {
      data.items.forEach(item => {
        listenersMap[item.track_id] = item.online_now || 0;
        playsMap[item.track_id] = item.total_plays || 0;
      });
    }

    setListeners(listenersMap);
    setPlays(playsMap);
  } catch (err) {
    console.error('Failed to load statistics', err);
    setListeners({});
    setPlays({})
  }
};

  useEffect(() => {
    loadTracks();
    loadStats();

    const interval = setInterval(() => {
      loadStats();
      loadTracks();
    }, 5000);

    return () => clearInterval(interval);
  }, []);

  // --- Трекинг через playback-api ---
  const startPlayback = async (trackId) => {
    try {
      await fetch(`/tracking/play/${trackId}`); // GET запрос (инициирует сессию)
    } catch (err) {
      console.error('Failed to start playback tracking', err);
    }
  };

  const sendPing = async (trackId) => {
    try {
      await fetch('/tracking/ping', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          track_id: trackId,
          user_session: localStorage.getItem('sessionId')
        })
      });
    } catch (err) {}
  };

  const sendStop = async (trackId) => {
    try {
      await fetch('/tracking/stop', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          track_id: trackId,
          user_session: localStorage.getItem('sessionId')
        })
      });
    } catch (err) {}
  };

  const playTrack = async (track) => {
    if (pingInterval.current) {
      clearInterval(pingInterval.current);
      if (currentTrack) await sendStop(currentTrack.id);
    }
    setCurrentTrack(track);
    try {
      const res = await fetch(`/tracking/play/${track.id}`);
      if (!res.ok) {
        const errorData = await res.json().catch(() => ({}));
        throw new Error(errorData.detail || 'Failed to start playback');
      }
      
      const data = await res.json();
      
      if (data.playlist_url) {
        if (Hls.isSupported()) {
          if (hlsRef.current) {
            hlsRef.current.destroy();
          }
          const hls = new Hls();
          hlsRef.current = hls;
          hls.loadSource(data.playlist_url);
          hls.attachMedia(audioRef.current);
          hls.on(Hls.Events.MANIFEST_PARSED, function () {
            audioRef.current.play().catch(e => console.error('Play error:', e));
          });
        } else if (audioRef.current.canPlayType('application/vnd.apple.mpegurl')) {
          audioRef.current.src = data.playlist_url;
          audioRef.current.addEventListener('loadedmetadata', function () {
            audioRef.current.play().catch(e => console.error('Play error:', e));
          });
        } else {
          throw new Error('HLS is not supported in this browser');
        }
        
        pingInterval.current = setInterval(() => sendPing(track.id), 10000);
      } else {
        throw new Error('No playlist URL');
      }
    } catch (err) {
      console.error('playTrack error:', err);
      alert('Cannot play track: ' + err.message);
    }
  };

  const downloadTrack = async (id) => {
    try {
      const res = await fetch(`/api/download/${id}`);
      const data = await res.json();
      if (data.download_url) window.open(data.download_url, '_blank');
      else alert('Download link not available');
    } catch (err) {
      alert('Download link unavailable');
    }
  };

  const skip = (seconds) => {
    if (audioRef.current) {
      audioRef.current.currentTime += seconds;
    }
  };

  // Остановка трекинга при паузе или закрытии
  useEffect(() => {
    const handlePause = () => {
      if (currentTrack && pingInterval.current) {
        sendStop(currentTrack.id);
        clearInterval(pingInterval.current);
        pingInterval.current = null;
      }
    };
    const handleBeforeUnload = () => {
      if (currentTrack) sendStop(currentTrack.id);
    };
    const audio = audioRef.current;
    if (audio) {
      audio.addEventListener('pause', handlePause);
      window.addEventListener('beforeunload', handleBeforeUnload);
      return () => {
        audio.removeEventListener('pause', handlePause);
        window.removeEventListener('beforeunload', handleBeforeUnload);
        if (currentTrack) sendStop(currentTrack.id);
      };
    }
  }, [currentTrack]);

  const filteredTracks = tracks.filter(track =>
    track.filename.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const handleTrackClick = (track) => {
    // Не запускаем заново, если это тот же трек
    if (!currentTrack || currentTrack.id !== track.id) {
      playTrack(track);
    }
    setIsPlayerModalOpen(true);
  };

   const handleSeek = (e) => {
    const newTime = Number(e.target.value);
    setCurrentTime(newTime);
    if (audioRef.current) {
      audioRef.current.currentTime = newTime;
    }
  };

  const togglePlay = () => {
    if (audioRef.current) {
      if (isPlaying) {
        audioRef.current.pause();
      } else {
        audioRef.current.play();
      }
    }
  };

  const handleVolumeChange = (e) => {
    const newVol = parseFloat(e.target.value);
    setVolume(newVol);
    if (audioRef.current) {
      audioRef.current.volume = newVol;
    }
  };

  const handleTimeUpdate = () => {
    if (audioRef.current) {
      setCurrentTime(audioRef.current.currentTime);
    }
  };

  const handleLoadedMetadata = () => {
    if (audioRef.current) {
      setDuration(audioRef.current.duration);
    }
  };

  const formatTime = (time) => {
    if (isNaN(time) || time === Infinity) return '0:00';
    const minutes = Math.floor(time / 60);
    const seconds = Math.floor(time % 60);
    return `${minutes}:${seconds < 10 ? '0' : ''}${seconds}`;
  };

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      className="home-container"
    >
      <div className="two-columns">
        {/* Левая колонка */}
        <div className="left-column">
          <div className="now-playing-card no-cover">
            <h2><BsHeadphones /> Playing Now</h2>
            <div className="track-title">{currentTrack?.filename || '—'}</div>
            <audio 
              ref={audioRef}
              onTimeUpdate={handleTimeUpdate}
              onLoadedMetadata={handleLoadedMetadata}
              onPlay={() => setIsPlaying(true)}
              onPause={() => setIsPlaying(false)}
              onEnded={() => setIsPlaying(false)}
            />

            <div className="track-progress-container" style={{ width: '100%', marginBottom: '20px', padding: '0 20px', boxSizing: 'border-box' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.85rem', color: '#8b9bb4', marginBottom: '8px' }}>
                <span>{formatTime(currentTime)}</span>
                <span>{formatTime(duration)}</span>
              </div>
              <input
                type="range"
                min={0}
                max={isFinite(duration) ? duration : 0}
                step="0.01"
                value={currentTime}
                onChange={handleSeek}
                style={{ width: '100%', cursor: 'pointer', accentColor: '#00f3ff' }}
              />
            </div>
            
            <div style={{ display: 'grid', gridTemplateColumns: '1fr auto 1fr', alignItems: 'center', gap: '20px', marginTop: '15px' }}>
              
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-end', gap: '5px', color: '#8b9bb4' }}>
                <BsVolumeUpFill />
                <input
                  type="range"
                  min="0"
                  max="1"
                  step="0.01"
                  value={volume}
                  onChange={handleVolumeChange}
                  style={{ width: '80px', accentColor: '#00f3ff', cursor: 'pointer' }}
                />
              </div>
                          
              <div className="skip-buttons" style={{ margin: 0, padding: 0 }}>
                <button className="skip-btn" onClick={() => skip(-10)}><BsRewind /><span>10</span></button>
                <button className="play-pause-btn" onClick={togglePlay}>
                  {isPlaying ? <BsPauseFill /> : <BsPlayFill />}
                </button>
                <button className="skip-btn" onClick={() => skip(10)}><BsFastForward /><span>10</span></button>
              </div>
              
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-start', gap: '15px', color: '#8b9bb4' }}>
                <span className="fire-badge" style={{ fontSize: '1rem', marginRight: '5px' }}>
                  <BsFire /> {currentTrack ? (plays[currentTrack.id] || 0) : 0}
                </span>
                <span className="fire-badge" style={{ fontSize: '1rem' }}>
                  <BsActivity /> {currentTrack ? (listeners[currentTrack.id] || 0) : 0}
                </span>
                <button className="icon-btn" onClick={() => currentTrack && downloadTrack(currentTrack.id)} disabled={!currentTrack}><BsDownload /></button>
              </div>
            </div>
          </div>

          <div className="search-music-block">
            <h2><BsSearch /> Search for music</h2>
            <div className="search-wrapper">
              <input
                type="text"
                placeholder="Search by track name..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="search-input-wide"
              />
            </div>
            <div className="tracks-list">
              {filteredTracks.map(track => (
                <motion.div
                  key={track.id}
                  whileHover={{ scale: 1.01 }}
                  className="track-item"
                  onClick={() => handleTrackClick(track)}
                  style={{ cursor: 'pointer' }}
                >
                  <span className="track-name">{track.filename}</span>
                  <div className="track-actions" onClick={(e) => e.stopPropagation()}>
                    <span className="fire-badge" style={{ marginRight: '5px' }}>
                      <BsFire /> {plays[track.id] || 0}
                    </span>
                    <span className="fire-badge">
                      <BsActivity /> {listeners[track.id] || 0}
                    </span>
                    <button className="icon-btn" onClick={() => (currentTrack && currentTrack.id === track.id) ? togglePlay() : playTrack(track)}>
                      {currentTrack && currentTrack.id === track.id && isPlaying ? <BsPauseFill /> : <BsPlayFill />}
                    </button>
                    <button className="icon-btn" onClick={() => downloadTrack(track.id)}><BsDownload /></button>
                  </div>
                </motion.div>
              ))}
              {filteredTracks.length === 0 && <div className="empty-message">No tracks found</div>}
            </div>
          </div>
        </div>

        {/* Правая колонка */}
        <div className="right-column">
          <div className="upload-block">
            <h2><BsUpload /> Upload your music</h2>
            <button className="upload-center-btn" onClick={() => setIsUploadModalOpen(true)}><BsUpload /> Upload</button>
          </div>
          <Top24
            onPlay={playTrack}
            onDownload={downloadTrack}
            onTrackClick={handleTrackClick}
            currentTrack={currentTrack}
            isPlaying={isPlaying}
            onTogglePlay={togglePlay}
          />
          <ListeningNow
            listeners={listeners}
            tracks={tracks}
            onPlay={playTrack}
            onDownload={downloadTrack}
            onTrackClick={handleTrackClick}
            currentTrack={currentTrack}
            isPlaying={isPlaying}
            onTogglePlay={togglePlay}
          />
        </div>
      </div>

      <UploadModal isOpen={isUploadModalOpen} onClose={() => setIsUploadModalOpen(false)} onUpload={loadTracks} />
      <TrackPlayerModal 
        isOpen={isPlayerModalOpen} 
        onClose={() => setIsPlayerModalOpen(false)} 
        track={currentTrack} 
        currentTime={currentTime}
        duration={duration}
        isPlaying={isPlaying}
        onSeek={handleSeek}
        onTogglePlay={togglePlay}
        onSkip={skip}
      />
    </motion.div>
  );
}