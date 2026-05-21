// pages/Product/RuleConfig.tsx

import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ruleApi, productApi } from '../../services/api';
import { Product, AuctionRule } from '../../types';

const RuleConfig: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [product, setProduct] = useState<Product | null>(null);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [formData, setFormData] = useState({
    start_price: 0,
    increment: 10,
    cap_price: 0,
    duration: 300,
    delay_duration: 30,
    max_delay_time: 180,
    trigger_delay_before: 30,
  });

  useEffect(() => {
    if (id) {
      fetchProduct();
    }
  }, [id]);

  const fetchProduct = async () => {
    try {
      const productData = await productApi.get(Number(id));
      setProduct(productData);
    } catch (error) {
      console.error('获取商品失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (formData.increment <= 0) {
      alert('加价幅度必须大于0');
      return;
    }

    if (formData.duration <= 0) {
      alert('竞拍时长必须大于0');
      return;
    }

    setSubmitting(true);
    try {
      await ruleApi.create(Number(id), {
        ...formData,
        cap_price: formData.cap_price > 0 ? formData.cap_price : undefined,
      });
      alert('竞拍规则配置成功！');
      navigate('/products');
    } catch (error) {
      console.error('配置竞拍规则失败:', error);
      alert('配置竞拍规则失败');
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return <div>加载中...</div>;
  }

  if (!product) {
    return <div>商品不存在</div>;
  }

  return (
    <div style={{ padding: '20px', maxWidth: '800px', margin: '0 auto' }}>
      <h1>配置竞拍规则</h1>

      <div style={{
        padding: '15px',
        backgroundColor: '#f5f5f5',
        borderRadius: '4px',
        marginBottom: '20px'
      }}>
        <h3 style={{ margin: '0 0 10px 0' }}>商品信息</h3>
        <p style={{ margin: '5px 0' }}><strong>商品名称：</strong>{product.name}</p>
        <p style={{ margin: '5px 0' }}><strong>商品描述：</strong>{product.description || '无'}</p>
      </div>

      <form onSubmit={handleSubmit}>
        <div style={formItemStyle}>
          <label style={labelStyle}>
            起拍价（元）
          </label>
          <input
            type="number"
            value={formData.start_price}
            onChange={(e) => setFormData({ ...formData, start_price: Number(e.target.value) })}
            min="0"
            step="0.01"
            style={inputStyle}
          />
          <small style={{ color: '#888' }}>默认 0 元起拍，任何人都可以参与竞拍</small>
        </div>

        <div style={formItemStyle}>
          <label style={labelStyle}>
            加价幅度（元） <span style={{ color: 'red' }}>*</span>
          </label>
          <input
            type="number"
            value={formData.increment}
            onChange={(e) => setFormData({ ...formData, increment: Number(e.target.value) })}
            min="0.01"
            step="0.01"
            style={inputStyle}
            required
          />
          <small style={{ color: '#888' }}>每次出价必须按此幅度递增</small>
        </div>

        <div style={formItemStyle}>
          <label style={labelStyle}>封顶价（元）</label>
          <input
            type="number"
            value={formData.cap_price}
            onChange={(e) => setFormData({ ...formData, cap_price: Number(e.target.value) })}
            min="0"
            step="0.01"
            style={inputStyle}
          />
          <small style={{ color: '#888' }}>达到封顶价自动成交，0表示无封顶</small>
        </div>

        <div style={formItemStyle}>
          <label style={labelStyle}>
            竞拍时长（秒） <span style={{ color: 'red' }}>*</span>
          </label>
          <input
            type="number"
            value={formData.duration}
            onChange={(e) => setFormData({ ...formData, duration: Number(e.target.value) })}
            min="60"
            style={inputStyle}
            required
          />
          <small style={{ color: '#888' }}>建议 300 秒（5分钟）</small>
        </div>

        <div style={formItemStyle}>
          <label style={labelStyle}>单次延时时长（秒）</label>
          <input
            type="number"
            value={formData.delay_duration}
            onChange={(e) => setFormData({ ...formData, delay_duration: Number(e.target.value) })}
            min="10"
            max="60"
            style={inputStyle}
          />
          <small style={{ color: '#888' }}>结束前出价触发延时，默认 30 秒</small>
        </div>

        <div style={formItemStyle}>
          <label style={labelStyle}>最大延时时长（秒）</label>
          <input
            type="number"
            value={formData.max_delay_time}
            onChange={(e) => setFormData({ ...formData, max_delay_time: Number(e.target.value) })}
            min="60"
            max="600"
            style={inputStyle}
          />
          <small style={{ color: '#888' }}>总延时不超过此限制，默认 180 秒（3分钟）</small>
        </div>

        <div style={formItemStyle}>
          <label style={labelStyle}>延时触发时间（秒）</label>
          <input
            type="number"
            value={formData.trigger_delay_before}
            onChange={(e) => setFormData({ ...formData, trigger_delay_before: Number(e.target.value) })}
            min="10"
            max="60"
            style={inputStyle}
          />
          <small style={{ color: '#888' }}>结束前多少秒内的出价触发延时，默认 30 秒</small>
        </div>

        <div style={{ marginTop: '30px', display: 'flex', gap: '10px' }}>
          <button
            type="submit"
            disabled={submitting}
            style={{
              ...buttonStyle,
              padding: '10px 30px',
              opacity: submitting ? 0.5 : 1,
            }}
          >
            {submitting ? '提交中...' : '保存配置'}
          </button>
          <button
            type="button"
            onClick={() => navigate('/products')}
            style={{
              ...buttonStyle,
              padding: '10px 30px',
              backgroundColor: '#fff',
              color: '#666',
              border: '1px solid #ddd',
            }}
          >
            取消
          </button>
        </div>
      </form>
    </div>
  );
};

const formItemStyle: React.CSSProperties = {
  marginBottom: '20px',
};

const labelStyle: React.CSSProperties = {
  display: 'block',
  marginBottom: '8px',
  fontWeight: 'bold',
};

const inputStyle: React.CSSProperties = {
  width: '100%',
  padding: '10px',
  border: '1px solid #ddd',
  borderRadius: '4px',
  fontSize: '14px',
  boxSizing: 'border-box',
};

const buttonStyle: React.CSSProperties = {
  backgroundColor: '#1890ff',
  color: 'white',
  border: 'none',
  borderRadius: '4px',
  cursor: 'pointer',
};

export default RuleConfig;
