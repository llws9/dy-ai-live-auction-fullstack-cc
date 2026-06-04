import React from 'react';
import styles from './ChatPanel.module.css';
import type { ChatMessage } from '../../store/liveChatStore';
import { repairUtf8Mojibake } from '../../utils/textEncoding';

interface ChatBubbleProps {
  msg: ChatMessage;
  isSelf: boolean;
}

export const ChatBubble: React.FC<ChatBubbleProps> = ({ msg, isSelf }) => {
  const userName = repairUtf8Mojibake(msg.user_name) || '用户';

  return (
    <div className={`${styles.bubble} ${isSelf ? styles.bubbleSelf : ''}`}>
      <span className={styles.userName}>{userName}</span>
      <span>{msg.text}</span>
    </div>
  );
};
