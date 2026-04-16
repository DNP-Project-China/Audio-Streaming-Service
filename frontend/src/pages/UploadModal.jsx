import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { BsCloudUpload, BsFileMusic, BsX } from 'react-icons/bs';

export default function UploadModal({ isOpen, onClose }) {
  const [file, setFile] = useState(null);
  const [status, setStatus] = useState('');

  const handleFile = (e) => setFile(e.target.files[0]);

  const upload = async () => {
    if (!file) { setStatus('Select a file'); return; }
    const formData = new FormData();
    formData.append('file', file);
    setStatus('Uploading...');
    try {
      const res = await fetch('/api/upload', { method: 'POST', body: formData });
      if (res.ok) {
        setStatus('✅ Uploaded! Processing...');
        setFile(null);
        setTimeout(() => {
          onClose();
          setStatus('');
        }, 2000);
      } else setStatus('❌ Upload failed');
    } catch (err) {
      setStatus('❌ API unavailable');
    }
  };

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
            className="modal-content"
            initial={{ scale: 0.9, y: 20 }}
            animate={{ scale: 1, y: 0 }}
            exit={{ scale: 0.9, y: 20 }}
            onClick={(e) => e.stopPropagation()}
          >
            <button className="modal-close" onClick={onClose}><BsX /></button>
            <h2><BsCloudUpload /> Upload Music</h2>
            <p>MP3, FLAC, WAV</p>
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