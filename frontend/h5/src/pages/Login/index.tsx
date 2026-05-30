import React, { useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useAuth } from '../../store/authContext';
import { authService } from '../../services/auth';
import styles from './Login.module.css';

const LoginPage: React.FC = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { setAuth } = useAuth();
  const redirectUrl = searchParams.get('redirect') || '/';

  const [loading, setLoading] = useState(false);
  const [phone, setPhone] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');

  const formatPhone = (value: string) => {
    const cleaned = value.replace(/\D/g, '').slice(0, 11);
    if (cleaned.length <= 3) return cleaned;
    if (cleaned.length <= 7) return `${cleaned.slice(0, 3)} ${cleaned.slice(3)}`;
    return `${cleaned.slice(0, 3)} ${cleaned.slice(3, 7)} ${cleaned.slice(7)}`;
  };

  const normalizePhone = (value: string) => value.replace(/\D/g, '');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const normalizedPhone = normalizePhone(phone);
    if (!normalizedPhone) {
      setError('请输入手机号');
      return;
    }

    if (!password.trim()) {
      setError('请输入密码');
      return;
    }

    setLoading(true);
    setError('');

    try {
      // 通过 authService 走统一 api.ts，token 与 user 由其内部持久化
      const data = await authService.login({ phone: normalizedPhone, password });
      setAuth(data.token, data.user);
      window.dispatchEvent(new CustomEvent('login-success'));
      navigate(redirectUrl);
    } catch (err: any) {
      setError(err?.message || '登录失败，请重试');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <button
          type="button"
          className={styles.backButton}
          aria-label="返回"
          onClick={() => navigate(-1)}
        >
          ‹
        </button>
        <div>
          <p className={styles.eyebrow}>Luxury Auction</p>
          <h1>登录</h1>
        </div>
      </header>

      <main className={styles.content}>
        <section className={styles.brandCard} aria-label="奢华竞拍">
          <div className={styles.logoMark}>LA</div>
          <h2>奢华竞拍</h2>
          <p>尊享品质，实时竞拍</p>
        </section>

        <form onSubmit={handleSubmit} className={styles.form}>
          <label className={styles.field} htmlFor="login-phone">
            <span>手机号</span>
            <input
              id="login-phone"
              type="tel"
              inputMode="numeric"
              placeholder="请输入手机号"
              value={phone}
              onChange={(e) => setPhone(formatPhone(e.target.value))}
              autoComplete="tel"
              maxLength={13}
            />
          </label>

          <label className={styles.field} htmlFor="login-password">
            <span>密码</span>
            <input
              id="login-password"
              type="password"
              placeholder="请输入密码"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
            />
          </label>

          {error && <div className={styles.error}>{error}</div>}

          <button type="submit" disabled={loading} className={styles.submitButton}>
            {loading ? '登录中...' : '登录'}
          </button>
        </form>

        <p className={styles.agreement}>登录即表示同意《用户协议》和《隐私政策》</p>
      </main>
    </div>
  );
};

export default LoginPage;
