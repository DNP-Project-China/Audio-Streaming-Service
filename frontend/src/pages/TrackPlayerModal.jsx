import { motion, AnimatePresence } from 'framer-motion';
import { BsX, BsPlayFill, BsPauseFill, BsRewind, BsFastForward } from 'react-icons/bs';

export default function TrackPlayerModal({ 
  isOpen, 
  onClose, 
  track, 
  currentTime, 
  duration, 
  isPlaying, 
  onSeek, 
  onTogglePlay, 
  onSkip 
}) {
  const formatTime = (time) => {
    if (isNaN(time) || time === Infinity) return '0:00';
    const minutes = Math.floor(time / 60);
    const seconds = Math.floor(time % 60);
    return `${minutes}:${seconds < 10 ? '0' : ''}${seconds}`;
  };

  if (!track) return null;

  return (
    <AnimatePresence>
      {isOpen && (
        <motion.div
          className="modal-overlay"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          onClick={onClose}
        >
          <motion.div
            className="modal-content track-player-modal"
            initial={{ scale: 0.9, y: 20 }}
            animate={{ scale: 1, y: 0 }}
            exit={{ scale: 0.9, y: 20 }}
            onClick={(e) => e.stopPropagation()}
          >
            <button className="modal-close" onClick={onClose}>
              <BsX />
            </button>

            <div className="modal-gif">
              <img
                src="https://media3.giphy.com/media/v1.Y2lkPTc5MGI3NjExeWYxZ2xkNXFnbWV1eW8zYWRlODR4N3B5ZGJscmtwbGQxc3Bsc3MzMyZlcD12MV9pbnRlcm5hbF9naWZfYnlfaWQmY3Q9Zw/ytu2GUYbvhz7zShGwS/giphy.gif"
                alt="Music animation"
                style={{ width: '100%', borderRadius: '20px' }}
              />
            </div>

            <h3 className="modal-track-title">{track.filename}</h3>

            <div className="track-progress-container" style={{ width: '100%', marginBottom: '20px', padding: '0', boxSizing: 'border-box' }}>
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
                onChange={onSeek}
                style={{ width: '100%', cursor: 'pointer', accentColor: '#00f3ff' }}
              />
            </div>

            <div className="custom-audio-controls" style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', gap: '20px' }}>
              <button className="skip-btn" onClick={() => onSkip(-10)}>
                <BsRewind /> <span>10</span>
              </button>
              <button className="play-pause-btn" onClick={onTogglePlay} style={{ background: '#00f3ff', color: 'black', border: 'none', borderRadius: '50%', width: '50px', height: '50px', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '24px', cursor: 'pointer', boxShadow: '0 0 10px #00f3ff' }}>
                {isPlaying ? <BsPauseFill /> : <BsPlayFill />}
              </button>
              <button className="skip-btn" onClick={() => onSkip(10)}>
                <BsFastForward /> <span>10</span>
              </button>
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}