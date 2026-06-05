import { Routes, Route, Navigate } from 'react-router-dom';
import Layout from './components/Layout';
import Dashboard from './pages/Dashboard';
import Pressure from './pages/Pressure';
import E2E from './pages/E2E';
import UserJourney from './pages/UserJourney';
import AntiSnipe from './pages/AntiSnipe';
import Callback from './pages/Callback';
import Chaos from './pages/Chaos';
import Compare from './pages/Compare';
import Screen from './pages/Screen';
import History from './pages/History';
import Report from './pages/Report';

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/test" replace />} />
      {/* 大屏模式独立于 Layout，无侧栏 */}
      <Route path="/test/screen" element={<Screen />} />
      <Route path="/test" element={<Layout />}>
        <Route index element={<Dashboard />} />
        <Route path="pressure" element={<Pressure />} />
        <Route path="e2e" element={<E2E />} />
        <Route path="user-journey" element={<UserJourney />} />
        <Route path="antisnipe" element={<AntiSnipe />} />
        <Route path="callback" element={<Callback />} />
        <Route path="chaos" element={<Chaos />} />
        <Route path="compare" element={<Compare />} />
        <Route path="history" element={<History />} />
        <Route path="report/:id" element={<Report />} />
      </Route>
      <Route path="*" element={<Navigate to="/test" replace />} />
    </Routes>
  );
}
