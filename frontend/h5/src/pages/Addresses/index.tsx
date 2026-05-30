import React, { useEffect, useState } from 'react';
import { addressApi } from '../../services/api';
import PageHeader from '@/components/shared/PageHeader';
import styles from './Addresses.module.css';

interface Address {
  id: number;
  recipient_name: string;
  phone: string;
  province: string;
  city: string;
  district: string;
  detail: string;
  is_default: boolean;
}

function extractList<T>(response: any): T[] {
  if (Array.isArray(response)) return response;
  if (Array.isArray(response?.items)) return response.items;
  if (Array.isArray(response?.list)) return response.list;
  if (Array.isArray(response?.data?.items)) return response.data.items;
  if (Array.isArray(response?.data?.list)) return response.data.list;
  return [];
}

function formatAddress(addr: Address) {
  return [addr.province, addr.city, addr.district, addr.detail].filter(Boolean).join(' ');
}

const AddressesPage: React.FC = () => {
  const [addresses, setAddresses] = useState<Address[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [pendingId, setPendingId] = useState<number | null>(null);

  const loadAddresses = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await addressApi.list();
      setAddresses(extractList<Address>(response));
    } catch (err: any) {
      setError(err?.message || '加载收货地址失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadAddresses();
  }, []);

  const handleSetDefault = async (id: number) => {
    setPendingId(id);
    try {
      await addressApi.setDefault(id);
      await loadAddresses();
    } catch (err) {
      // request 内部已 toast，无需再次提示
    } finally {
      setPendingId(null);
    }
  };

  return (
    <div className={styles.page}>
      <PageHeader
        classes={{ header: styles.header, backButton: styles.backButton }}
        back={{ to: '/profile' }}
        title="收货地址"
      />

      <div className={styles.content}>
        {loading && <div className={styles.loading}>加载中…</div>}
        {error && !loading && <div className={styles.errorBanner}>{error}</div>}

        {!loading && !error && addresses.length === 0 && (
          <div className={styles.emptyState}>
            <div className={styles.emptyMark}>＋</div>
            <p>暂无收货地址</p>
            <span>地址新增功能即将开放</span>
          </div>
        )}

        {!loading && !error && addresses.length > 0 && (
          <div className={styles.list}>
            {addresses.map((addr) => (
              <div key={addr.id} className={styles.card}>
                <div className={styles.cardHead}>
                  <span className={styles.recipient}>{addr.recipient_name}</span>
                  <span className={styles.phone}>{addr.phone}</span>
                  {addr.is_default && <span className={styles.defaultBadge}>默认</span>}
                </div>
                <p className={styles.address}>{formatAddress(addr)}</p>
                {!addr.is_default && (
                  <div className={styles.actions}>
                    <button
                      type="button"
                      className={styles.setDefaultButton}
                      disabled={pendingId === addr.id}
                      onClick={() => handleSetDefault(addr.id)}
                    >
                      {pendingId === addr.id ? '设置中…' : '设为默认'}
                    </button>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default AddressesPage;
