import { motion } from 'framer-motion';
import { BsPlayFill, BsPauseFill, BsDownload, BsActivity } from 'react-icons/bs';

export default function ListeningNow({ listeners, tracks, onPlay, onDownload, onTrackClick, currentTrack, isPlaying, onTogglePlay }) {
  // Фильтруем треки, у которых есть хотя бы один слушатель
  const activeTracks = tracks.filter(track => (listeners[track.id] || 0) > 0);

  return (
    <div className="listening-now-card">
      <h2><BsActivity /> Listening now</h2>
      <div className="listening-list">
        {activeTracks.length > 0 ? (
          activeTracks.map(track => (
            <motion.div
              key={track.id}
              whileHover={{ scale: 1.01 }}
              className="track-item"
              onClick={() => onTrackClick(track)}
              style={{ cursor: 'pointer' }}
            >
              <span className="track-name">{track.filename}</span>
              <div className="track-actions" onClick={(e) => e.stopPropagation()}>
                <span className="fire-badge">
                  <BsActivity /> {listeners[track.id]}
                </span>
                <button
                  className="icon-btn"
                  onClick={() => (currentTrack && currentTrack.id === track.id) ? onTogglePlay() : onPlay(track)}
                >
                  {currentTrack && currentTrack.id === track.id && isPlaying ? <BsPauseFill /> : <BsPlayFill />}
                </button>
                <button
                  className="icon-btn"
                  onClick={() => onDownload(track.id)}
                >
                  <BsDownload />
                </button>
              </div>
            </motion.div>
          ))
        ) : (
          <div className="empty-message">No one is listening right now</div>
        )}
      </div>
    </div>
  );
}