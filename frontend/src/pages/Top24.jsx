import { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { BsPlayFill, BsPauseFill, BsDownload, BsFire } from 'react-icons/bs';

export default function Top24({ onPlay, onDownload, onTrackClick, isPlaying, currentTrack, togglePlay, playTrack }) {
  const [topTracks, setTopTracks] = useState([]);

  useEffect(() => {
    const loadTop = async () => {
      try {
        const res = await fetch('/stats/live');
        const data = await res.json();
        if (data.items && Array.isArray(data.items)) {
          // Сортируем по total_plays (убывание) и берём первые 3
          const sorted = [...data.items].sort((a, b) => b.total_plays - a.total_plays);
          setTopTracks(sorted.slice(0, 3));
        } else {
          setTopTracks([]);
        }
      } catch (err) {
        console.error('Failed to load top tracks', err);
        setTopTracks([]);
      }
    };
    loadTop();
    const interval = setInterval(loadTop, 30000);
    return () => clearInterval(interval);
  }, []);

  const getTrack = (track) => ({ 
    id: track.track_id, 
    filename: `${track.artist} - ${track.title}` 
  });

  return (
    <div className="top24-card">
      <h2><BsFire /> Top 24 hours</h2>
      <div className="top24-list">
        {topTracks.map((track, idx) => (
          <motion.div
            key={track.track_id}
            whileHover={{ scale: 1.01 }}
            className="track-item"
            onClick={() => onTrackClick(getTrack(track))}
            style={{ cursor: 'pointer' }}
          >
            <span className="track-name">{idx+1}. {track.artist} - {track.title}</span>
            <div className="track-actions" onClick={(e) => e.stopPropagation()}>
              <span className="fire-badge"><BsFire /> {track.total_plays || 0}</span>
              <button className="icon-btn" onClick={() => (currentTrack && currentTrack.id === track.id) ? togglePlay() : playTrack(track)}>
                {currentTrack && currentTrack.id === track.id && isPlaying ? <BsPauseFill /> : <BsPlayFill />}
              </button>
              <button className="icon-btn" onClick={() => onDownload(track.track_id)}><BsDownload /></button>
            </div>
          </motion.div>
        ))}
        {topTracks.length === 0 && <div className="empty-message">No data for last 24h</div>}
      </div>
    </div>
  );
}