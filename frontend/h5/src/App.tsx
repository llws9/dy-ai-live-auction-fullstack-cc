import { Routes, Route } from 'react-router-dom'
import Home from './pages/Home'
import Auction from './pages/Auction'
import Result from './pages/Result'
import History from './pages/History'
import { AuctionProvider } from './store/auctionContext'

function App() {
  return (
    <AuctionProvider>
      <div className="app">
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/auction/:id" element={<Auction />} />
          <Route path="/result/:id" element={<Result />} />
          <Route path="/history" element={<History />} />
        </Routes>
      </div>
    </AuctionProvider>
  )
}

export default App
