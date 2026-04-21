import { motion, AnimatePresence } from 'framer-motion';
import { BsX, BsPlayFill, BsPauseFill, BsRewind, BsFastForward } from 'react-icons/bs';
import { useState, useEffect } from 'react';

export default function TrackPlayerModal({ isOpen, onClose, track, audioRef }) {
  const [isPlaying, setIsPlaying] = useState(false);

  useEffect(() => {
    const audioEl = audioRef.current;
    if (audioEl) {
      const handlePlay = () => setIsPlaying(true);
      const handlePause = () => setIsPlaying(false);
      const handleError = (e) => console.error('Audio error:', e);

      audioEl.addEventListener('play', handlePlay);
      audioEl.addEventListener('pause', handlePause);
      audioEl.addEventListener('error', handleError);

      return () => {
        audioEl.removeEventListener('play', handlePlay);
        audioEl.removeEventListener('pause', handlePause);
        audioEl.removeEventListener('error', handleError);
      };
    }
  }, [audioRef]);

  // При закрытии модалки останавливаем воспроизведение
  useEffect(() => {
    if (!isOpen && audioRef.current) {
      audioRef.current.pause();
      // Не очищаем src, чтобы можно было продолжить с того же места при открытии
    }
  }, [isOpen, audioRef]);

  const togglePlay = () => {
    if (audioRef.current) {
      if (isPlaying) {
        audioRef.current.pause();
      } else {
        audioRef.current.play().catch(err => console.error('Play failed:', err));
      }
    }
  };

  const skip = (seconds) => {
    if (audioRef.current) {
      audioRef.current.currentTime += seconds;
    }
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

            <div className="custom-audio-controls">
              <button className="skip-btn" onClick={() => skip(-10)}>
                <BsRewind /> <span>10</span>
              </button>
              <button className="play-pause-btn" onClick={togglePlay}>
                {isPlaying ? <BsPauseFill /> : <BsPlayFill />}
              </button>
              <button className="skip-btn" onClick={() => skip(10)}>
                <BsFastForward /> <span>10</span>
              </button>
            </div>

            <audio ref={audioRef} style={{ display: 'none' }} />
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}