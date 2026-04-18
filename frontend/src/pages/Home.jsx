import { useState, useEffect, useRef } from 'react';
import { motion } from 'framer-motion';
import { BsPlayFill, BsDownload, BsFire, BsHeadphones, BsUpload, BsRewind, BsFastForward, BsSearch } from 'react-icons/bs';
import Top24 from './Top24';
import UploadModal from './UploadModal';
import ListeningNow from './ListeningNow';
import TrackPlayerModal from './TrackPlayerModal';

export default function Home() {
  const [tracks, setTracks] = useState([]);
  const [currentTrack, setCurrentTrack] = useState(null);
  const [listeners, setListeners] = useState({});
  const [isUploadModalOpen, setIsUploadModalOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [isPlayerModalOpen, setIsPlayerModalOpen] = useState(false);
  const audioRef = useRef(null);
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

  const loadListeners = async () => {
    try {
      const res = await fetch('/stats/live');
      const data = await res.json();
      setListeners(data);
    } catch (err) {
      setListeners({});
    }
  };

  useEffect(() => {
    loadTracks();
    loadListeners();
    const interval = setInterval(loadListeners, 5000);
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
      const res = await fetch(`/api/download/${track.id}`);
      const data = await res.json();
      if (data.download_url) {
        audioRef.current.src = data.download_url;
        await audioRef.current.play();
        await startPlayback(track.id);
        pingInterval.current = setInterval(() => sendPing(track.id), 10000);
      } else {
        throw new Error('No download URL');
      }
    } catch (err) {
      alert('Cannot play track');
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
    playTrack(track);
    setIsPlayerModalOpen(true);
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
            <h2><BsHeadphones /> Now Playing</h2>
            <div className="track-title">{currentTrack?.filename || '—'}</div>
            <audio ref={audioRef} controls />
            <div className="skip-buttons">
              <button className="skip-btn" onClick={() => skip(-10)}><BsRewind /><span>10</span></button>
              <button className="skip-btn" onClick={() => skip(10)}><BsFastForward /><span>10</span></button>
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
                    <span className="fire-badge"><BsFire /> {listeners[track.id] || 0}</span>
                    <button className="icon-btn" onClick={() => playTrack(track)}><BsPlayFill /></button>
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
          <Top24 onPlay={playTrack} onDownload={downloadTrack} onTrackClick={handleTrackClick} />
          <ListeningNow listeners={listeners} tracks={tracks} onPlay={playTrack} onDownload={downloadTrack} onTrackClick={handleTrackClick} />
        </div>
      </div>

      <UploadModal isOpen={isUploadModalOpen} onClose={() => setIsUploadModalOpen(false)} onUpload={loadTracks} />
      <TrackPlayerModal isOpen={isPlayerModalOpen} onClose={() => setIsPlayerModalOpen(false)} track={currentTrack} audioRef={audioRef} />
    </motion.div>
  );
}