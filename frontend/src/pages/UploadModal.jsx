import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { BsCloudUpload, BsFileMusic, BsX } from 'react-icons/bs';

// Maximum time to wait for upload completion before aborting
const UPLOAD_TIMEOUT_MS = 300000;

export default function UploadModal({ isOpen, onClose, onUpload }) {
  const [file, setFile] = useState(null);
  const [artist, setArtist] = useState('');
  const [title, setTitle] = useState('');
  const [status, setStatus] = useState('');

  // Store selected file
  const handleFile = (e) => setFile(e.target.files[0]);

  // Handle upload submission
  const upload = async () => {
    // Validate inputs
    if (!file) { setStatus('Select a file'); return; }
    if (!artist.trim() || !title.trim()) { setStatus('Artist and Title are required'); return; }

    const formData = new FormData();
    formData.append('file', file);
    formData.append('artist', artist);
    formData.append('title', title);
    setStatus('Uploading...');

    // Set up abort controller with timeout
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), UPLOAD_TIMEOUT_MS);

    try {
      const res = await fetch('/api/upload', {
        method: 'POST',
        body: formData,
        signal: controller.signal,
      });
      if (res.ok) {
        setStatus('✅ Uploaded! Processing...');
        // Clear form and close modal after short delay
        setFile(null);
        setArtist('');
        setTitle('');
        setTimeout(() => {
          onClose();
          setStatus('');
          if (onUpload) onUpload(); // Refresh track list
        }, 2000);
      } else {
        const err = await res.json();
        setStatus(`❌ Upload failed: ${err.error || 'unknown'}`);
      }
    } catch (err) {
      if (err.name === 'AbortError') {
        setStatus('❌ Upload timeout. Try again (large files may take longer).');
      } else {
        setStatus('❌ API unavailable');
      }
    } finally {
      clearTimeout(timeoutId);
    }
  };

  return (
    <AnimatePresence>
      {isOpen && (
        // Backdrop overlay
        <motion.div
          className="modal-overlay"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          onClick={onClose}
        >
          {/* Modal panel */}
          <motion.div
            className="modal-content"
            initial={{ scale: 0.9, y: 20 }}
            animate={{ scale: 1, y: 0 }}
            exit={{ scale: 0.9, y: 20 }}
            onClick={(e) => e.stopPropagation()}
          >
            <button className="modal-close" onClick={onClose}><BsX /></button>
            <h2><BsCloudUpload /> Upload Music</h2>
            <p>MP3, FLAC, WAV</p>

            {/* Artist and title input fields */}
            <input
              type="text"
              placeholder="Artist"
              value={artist}
              onChange={(e) => setArtist(e.target.value)}
              className="search-input-wide"
              style={{ marginTop:30, marginBottom: 15 }}
            />
            <input
              type="text"
              placeholder="Title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="search-input-wide"
              style={{ marginBottom: 10 }}
            />

            {/* File picker */}
            <label className="file-label">
              <BsFileMusic /> Choose file
              <input type="file" accept=".mp3,.flac,.wav" onChange={handleFile} style={{ display: 'none' }} />
            </label>
            <div className="file-name">{file ? file.name : 'No file chosen'}</div>
            <button className="upload-submit" onClick={upload}><BsCloudUpload /> Upload</button>
            <div className="upload-status">{status}</div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}