// frontend/h5/src/pages/Auction/LiveVideo.tsx

import React from 'react';

interface LiveVideoProps {
  streamUrl?: string;
}

const LiveVideo: React.FC<LiveVideoProps> = ({ streamUrl }) => {
  return (
    <div style={{
      width: '100%',
      height: '300px',
      backgroundColor: '#000',
      borderRadius: '8px',
      overflow: 'hidden',
      position: 'relative',
    }}>
      {streamUrl ? (
        <video
          src={streamUrl}
          autoPlay
          muted
          playsInline
          style={{
            width: '100%',
            height: '100%',
            objectFit: 'cover',
          }}
        />
      ) : (
        <div style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          height: '100%',
          color: '#fff',
        }}>
          <div style={{ fontSize: '48px', marginBottom: '10px' }}>📺</div>
          <div>直播画面加载中...</div>
        </div>
      )}

      {/* 直播状态指示器 */}
      <div style={{
        position: 'absolute',
        top: '10px',
        left: '10px',
        padding: '4px 8px',
        backgroundColor: 'rgba(255, 0, 0, 0.8)',
        borderRadius: '4px',
        color: '#fff',
        fontSize: '12px',
        display: 'flex',
        alignItems: 'center',
        gap: '5px',
      }}>
        <span style={{
          width: '8px',
          height: '8px',
          borderRadius: '50%',
          backgroundColor: '#fff',
          animation: 'pulse 1s infinite',
        }} />
        直播中
      </div>
    </div>
  );
};

export default LiveVideo;
