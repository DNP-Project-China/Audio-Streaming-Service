import { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { BsPlayFill, BsDownload, BsFire } from 'react-icons/bs';

export default function Top24({ onPlay, onDownload, onTrackClick }) {
  const [chart, setChart] = useState([]);

  // ---------- МОКОВЫЕ ДАННЫЕ ДЛЯ ТЕСТИРОВАНИЯ (удалить при подключении реального API) ----------
  const mockChart = [
    { id: '1', filename: 'Popular Track 1', play_count_24h: 42 },
    { id: '2', filename: 'Hit Song', play_count_24h: 37 },
    { id: '3', filename: 'Trendy Beat', play_count_24h: 25 },
    { id: '4', filename: 'Midnight Vibes', play_count_24h: 18 },
    { id: '5', filename: 'Neon Dreams', play_count_24h: 12 }
  ];

  useEffect(() => {
    // ВРЕМЕННО: моковые данные
    setChart(mockChart);
    // Реальный API закомментирован
    /*
    const load = () => {
      fetch('/stats/chart')
        .then(res => res.json())
        .then(setChart)
        .catch(() => setChart([]));
    };
    load();
    const interval = setInterval(load, 30000);
    return () => clearInterval(interval);
    */
  }, []);

  // Берем только первые 3 трека (топ-3 по прослушиваниям, предполагаем, что данные уже отсортированы)
  const top3 = chart.slice(0, 3);

  return (
    <div className="top24-card">
      <h2><BsFire /> Top 24 hours</h2>
      <div className="top24-list">
        {top3.map((track, idx) => (
          <motion.div
            key={track.id}
            whileHover={{ scale: 1.01 }}
            className="track-item"
            onClick={() => onTrackClick(track)}
            style={{ cursor: 'pointer' }}
          >
            <span className="track-name">{idx+1}. {track.filename}</span>
            <div className="track-actions" onClick={(e) => e.stopPropagation()}>
              <span className="fire-badge"><BsFire /> {track.play_count_24h || 0}</span>
              <button className="icon-btn" onClick={() => onPlay(track)}><BsPlayFill /></button>
              <button className="icon-btn" onClick={() => onDownload(track.id)}><BsDownload /></button>
            </div>
          </motion.div>
        ))}
        {top3.length === 0 && <div className="empty-message">No data for last 24h</div>}
      </div>
    </div>
  );
}