import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import styles from './LiveReminderModal.module.css';

export interface StreamInfo {
  id: string | number;
  name: string;
  avatarUrl: string;
  statusText?: string;
}

interface LiveReminderModalProps {
  isOpen: boolean;
  onClose: () => void;
  stream: StreamInfo | null;
}

const LiveReminderModal: React.FC<LiveReminderModalProps> = ({ isOpen, onClose, stream }) => {
  const navigate = useNavigate();
  const [shouldRender, setShouldRender] = useState(false);

  // 控制动画挂载/卸载
  useEffect(() => {
    if (isOpen) {
      setShouldRender(true);
      return;
    }

    if (!shouldRender) {
      return;
    }

    const timer = setTimeout(() => {
      setShouldRender(false);
    }, 200); // 对应动画的时间
    return () => clearTimeout(timer);
  }, [isOpen, shouldRender]);

  if (!shouldRender || !stream) return null;

  const handleJump = () => {
    onClose();
    // 假设跳转到直播间路由，具体路径根据实际情况调整
    navigate(`/live`);
  };

  return (
    <div 
      className={`${styles.overlay} ${!isOpen ? styles.fadeOut : ''}`} 
      onClick={onClose}
    >
      <div 
        className={`${styles.modal} ${!isOpen ? styles.slideDown : ''}`} 
        role="dialog"
        aria-modal="true"
        aria-labelledby="live-reminder-title"
        onClick={e => e.stopPropagation()}
      >
        <div className={styles.header}>
          <div className={styles.iconWrapper}>
            🎥
          </div>
          <h3 id="live-reminder-title" className={styles.title}>直播开播提醒</h3>
        </div>
        
        <div className={styles.content}>
          <p className={styles.message}>
            您收藏的直播间已经开始啦，快来参与竞拍吧！
          </p>
          
          <div className={styles.streamInfo}>
            <img 
              src={stream.avatarUrl} 
              alt={stream.name} 
              className={styles.streamAvatar} 
            />
            <div className={styles.streamDetails}>
              <h4 className={styles.streamName}>{stream.name}</h4>
              <div className={styles.streamStatus}>
                <span className={styles.liveDot}></span>
                {stream.statusText || '正在直播'}
              </div>
            </div>
          </div>
        </div>

        <div className={styles.footer}>
          <button 
            className={`${styles.button} ${styles.buttonCancel}`} 
            onClick={onClose}
          >
            稍后再看
          </button>
          <button 
            className={`${styles.button} ${styles.buttonConfirm}`} 
            onClick={handleJump}
          >
            立即前往
          </button>
        </div>
      </div>
    </div>
  );
};

export default LiveReminderModal;
