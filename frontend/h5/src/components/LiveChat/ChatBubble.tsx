import React from 'react';
import styles from './ChatPanel.module.css';
import type { ChatMessage } from '../../store/liveChatStore';

interface ChatBubbleProps {
  msg: ChatMessage;
  isSelf: boolean;
}

export const ChatBubble: React.FC<ChatBubbleProps> = ({ msg, isSelf }) => {
  return (
    <div className={`${styles.bubble} ${isSelf ? styles.bubbleSelf : ''}`}>
      <span className={styles.userName}>{msg.user_name}</span>
      <span>{msg.text}</span>
    </div>
  );
};
