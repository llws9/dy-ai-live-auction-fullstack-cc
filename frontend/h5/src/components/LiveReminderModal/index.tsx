import React, { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { trackBusinessEvent } from '../../utils/businessEvent';
import { trackEvent } from '../../utils/trackEvent';
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
  const actionTrackedRef = useRef(false);

  // 控制动画挂载/卸载
  useEffect(() => {
    if (isOpen) {
      actionTrackedRef.current = false;
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

  const trackActionOnce = (eventName: 'live_reminder_clicked' | 'live_reminder_dismissed', result: 'clicked' | 'dismissed') => {
    if (actionTrackedRef.current) {
      return false;
    }
    actionTrackedRef.current = true;
    trackEvent(eventName, {
      source: 'mobile_shell',
      entry: 'live_reminder_modal',
      type: 'live_start',
      result,
    });
    return true;
  };

  const trackDismiss = () => {
    if (!trackActionOnce('live_reminder_dismissed', 'dismissed')) {
      return;
    }
    onClose();
  };

  const handleJump = () => {
    if (!trackActionOnce('live_reminder_clicked', 'clicked')) {
      return;
    }
    trackBusinessEvent('reminder_click', {
      source: 'live_reminder',
      liveStreamId: Number(stream.id) || undefined,
    });
    onClose();
    navigate(`/live?id=${stream.id}`);
  };

  const avatarUrl = stream.avatarUrl.trim();
  const avatarInitial = stream.name.trim().slice(0, 1).toUpperCase() || '播';

  return (
    <div 
      className={`${styles.overlay} ${!isOpen ? styles.fadeOut : ''}`} 
      onClick={trackDismiss}
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
            {avatarUrl ? (
              <img
                src={avatarUrl}
                alt={stream.name}
                className={styles.streamAvatar}
              />
            ) : (
              <div className={styles.streamAvatarFallback} aria-hidden="true">
                {avatarInitial}
              </div>
            )}
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
            onClick={trackDismiss}
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
