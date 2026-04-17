import { useState, useEffect, useRef } from 'react';
import { motion } from 'framer-motion';
import Hls from 'hls.js';
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

  // ---------- МОКОВЫЕ ДАННЫЕ ДЛЯ ТЕСТИРОВАНИЯ (15 треков для скролла) ----------
  const mockTracks = [
    { id: '1', filename: 'Test Track 1 (Mock)' },
    { id: '2', filename: 'Another Mock Track' },
    { id: '3', filename: 'Chill Beats' },
    { id: '4', filename: 'Summer Vibes' },
    { id: '5', filename: 'Midnight Rain' },
    { id: '6', filename: 'Electric Dreams' },
    { id: '7', filename: 'Lost in Space' },
    { id: '8', filename: 'Neon Lights' },
    { id: '9', filename: 'Ocean Drive' },
    { id: '10', filename: 'Retro Wave' },
    { id: '11', filename: 'Funk Odyssey' },
    { id: '12', filename: 'Deep House' },
    { id: '13', filename: 'Jazz Lofi' },
    { id: '14', filename: 'Rock Classic' },
    { id: '15', filename: 'Acoustic Session' }
  ];
  const mockListeners = {
    '1': 3, '2': 1, '3': 0, '4': 2, '5': 5, '6': 0,
    '7': 1, '8': 4, '9': 2, '10': 3, '11': 0, '12': 1,
    '13': 2, '14': 3, '15': 1
  };

  useEffect(() => {
    setTracks(mockTracks);
    setListeners(mockListeners);

    const mockInterval = setInterval(() => {
      setListeners(prev => {
        const newListeners = { ...prev };
        Object.keys(newListeners).forEach(id => {
          newListeners[id] = Math.max(0, (newListeners[id] || 0) + Math.floor(Math.random() * 3) - 1);
        });
        return newListeners;
      });
    }, 5000);

    return () => {
      clearInterval(mockInterval);
      if (pingInterval.current) clearInterval(pingInterval.current);
    };
  }, []);

  const sendEvent = (event, trackId) => {
    fetch('/tracking/event', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        event,
        track_id: trackId,
        session_id: localStorage.getItem('sessionId'),
        ts: Date.now()
      })
    }).catch(() => {});
  };

  const playTrack = (track) => {
    if (pingInterval.current) {
      clearInterval(pingInterval.current);
      if (currentTrack) sendEvent('stop', currentTrack.id);
    }
    setCurrentTrack(track);
    const hlsUrl = `/api/stream/${track.id}/master.m3u8`;
    if (Hls.isSupported()) {
      const hls = new Hls();
      hls.loadSource(hlsUrl);
      hls.attachMedia(audioRef.current);
      hls.on(Hls.Events.MANIFEST_PARSED, () => {
        audioRef.current.play();
        sendEvent('start', track.id);
        pingInterval.current = setInterval(() => sendEvent('ping', track.id), 10000);
      });
    } else if (audioRef.current.canPlayType('application/vnd.apple.mpegurl')) {
      audioRef.current.src = hlsUrl;
      audioRef.current.addEventListener('canplay', () => {
        audioRef.current.play();
        sendEvent('start', track.id);
        pingInterval.current = setInterval(() => sendEvent('ping', track.id), 10000);
      });
    }
  };

  const downloadTrack = async (id) => {
    try {
      const res = await fetch(`/api/download/${id}`);
      const data = await res.json();
      if (data.url) window.open(data.url, '_blank');
    } catch (err) {
      alert('Download link unavailable');
    }
  };

  const skip = (seconds) => {
    if (audioRef.current) {
      audioRef.current.currentTime += seconds;
    }
  };

  useEffect(() => {
    return () => {
      if (currentTrack && pingInterval.current) {
        sendEvent('stop', currentTrack.id);
      }
    };
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
        {/* ЛЕВАЯ КОЛОНКА */}
        <div className="left-column">
          <div className="now-playing-card no-cover">
            <h2><BsHeadphones /> Now Playing</h2>
            <div className="track-title">{currentTrack?.filename || '—'}</div>
            <audio ref={audioRef} controls />
            <div className="skip-buttons">
              <button className="skip-btn" onClick={() => skip(-10)}>
                <BsRewind /> <span>10</span>
              </button>
              <button className="skip-btn" onClick={() => skip(10)}>
                <BsFastForward /> <span>10</span>
              </button>
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
                    <button
                      className="icon-btn"
                      onClick={(e) => {
                        e.stopPropagation();
                        playTrack(track);
                      }}
                    >
                      <BsPlayFill />
                    </button>
                    <button
                      className="icon-btn"
                      onClick={(e) => {
                        e.stopPropagation();
                        downloadTrack(track.id);
                      }}
                    >
                      <BsDownload />
                    </button>
                  </div>
                </motion.div>
              ))}
              {filteredTracks.length === 0 && <div className="empty-message">No tracks found</div>}
            </div>
          </div>
        </div>

        {/* ПРАВАЯ КОЛОНКА */}
        <div className="right-column">
          <div className="upload-block">
            <h2><BsUpload /> Upload your music</h2>
            <button className="upload-center-btn" onClick={() => setIsUploadModalOpen(true)}>
              <BsUpload /> Upload
            </button>
          </div>

          <Top24
            onPlay={playTrack}
            onDownload={downloadTrack}
            onTrackClick={handleTrackClick}
          />

          <ListeningNow
            listeners={listeners}
            tracks={tracks}
            onPlay={playTrack}
            onDownload={downloadTrack}
            onTrackClick={handleTrackClick}
          />
        </div>
      </div>

      <UploadModal isOpen={isUploadModalOpen} onClose={() => setIsUploadModalOpen(false)} />

      <TrackPlayerModal
        isOpen={isPlayerModalOpen}
        onClose={() => setIsPlayerModalOpen(false)}
        track={currentTrack}
        audioRef={audioRef}
      />
    </motion.div>
  );
}