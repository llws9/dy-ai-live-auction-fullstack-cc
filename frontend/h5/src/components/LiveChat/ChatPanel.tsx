import React, { useEffect, useRef, useState } from 'react';
import styles from './ChatPanel.module.css';
import { ChatBubble } from './ChatBubble';
import { useLiveChatStore } from '../../store/liveChatStore';

const MAX_LEN = 50;

interface ChatPanelProps {
  currentUserId: number;
  onSend: (text: string, clientMsgId: string) => void;
}

export const ChatPanel: React.FC<ChatPanelProps> = ({ currentUserId, onSend }) => {
  const history = useLiveChatStore((s) => s.history);
  const markSent = useLiveChatStore((s) => s.markSent);
  const isCoolingDown = useLiveChatStore((s) => s.isCoolingDown);

  const [text, setText] = useState('');
  const [tick, setTick] = useState(0); // 强制刷新 cooldown 状态
  const listRef = useRef<HTMLDivElement>(null);

  // cooldown 倒计时刷新
  useEffect(() => {
    const id = setInterval(() => setTick((t) => t + 1), 200);
    return () => clearInterval(id);
  }, []);

  // 自动滚动到底部
  useEffect(() => {
    if (listRef.current) {
      listRef.current.scrollTop = listRef.current.scrollHeight;
    }
  }, [history.length]);

  const trimmed = text.trim();
  const tooLong = [...trimmed].length > MAX_LEN;
  const canSend = trimmed.length > 0 && !tooLong && !isCoolingDown();

  const handleSend = () => {
    if (!canSend) return;
    const clientMsgId = `${currentUserId}-${Date.now()}`;
    onSend(trimmed, clientMsgId);
    markSent();
    setText('');
  };

  return (
    <div className={styles.panel}>
      <div className={styles.list} ref={listRef} data-testid="chat-list">
        {history.map((m, i) => (
          <ChatBubble key={`${m.sent_at}-${i}`} msg={m} isSelf={m.user_id === currentUserId} />
        ))}
      </div>
      <div className={styles.inputBar}>
        <input
          className={styles.input}
          placeholder="说点什么..."
          value={text}
          maxLength={MAX_LEN * 4 /* 给中文留空间，业务上仍按字符数限制 */}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') handleSend();
          }}
        />
        <button
          type="button"
          className={styles.sendBtn}
          disabled={!canSend}
          onClick={handleSend}
          data-tick={tick}
        >
          发送
        </button>
      </div>
    </div>
  );
};
