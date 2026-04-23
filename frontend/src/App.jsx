import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { useEffect, useState } from 'react';
import Home from './pages/Home';
import './App.css';

function App() {
  const [sessionId, setSessionId] = useState('');

  useEffect(() => {
    let id = localStorage.getItem('sessionId');
    if (!id) {
      id = (typeof crypto !== 'undefined' && crypto.randomUUID) ? crypto.randomUUID() : Math.random().toString(36).substring(2, 15);
      localStorage.setItem('sessionId', id);
    }
    setSessionId(id.slice(0, 8) + '...');
  }, []);

  return (
    <BrowserRouter>
      <div className="app">
        <header className="header">
          <div className="logo">🎵 Waves</div>
          {/*<div className="session-badge">{sessionId}</div>*/}
        </header>
        <Routes>
          <Route path="/" element={<Home />} />
        </Routes>
      </div>
    </BrowserRouter>
  );
}

export default App;