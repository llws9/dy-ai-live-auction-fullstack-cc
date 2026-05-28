// pages/Product/RuleConfig.tsx

import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ruleApi, productApi } from '../../services/api';
import { Product } from '../../types';

const RuleConfig: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [product, setProduct] = useState<Product | null>(null);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
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

  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (formData.increment <= 0) {
      newErrors.increment = '加价幅度必须大于0';
    }

    if (formData.duration < 60) {
      newErrors.duration = '竞拍时长不能少于60秒';
    }

    if (formData.duration > 3600) {
      newErrors.duration = '竞拍时长不能超过3600秒（1小时）';
    }

    if (formData.delay_duration < 10 || formData.delay_duration > 60) {
      newErrors.delay_duration = '单次延时时长必须在10-60秒之间';
    }

    if (formData.max_delay_time < 60 || formData.max_delay_time > 600) {
      newErrors.max_delay_time = '最大延时时长必须在60-600秒之间';
    }

    if (formData.trigger_delay_before < 10 || formData.trigger_delay_before > 60) {
      newErrors.trigger_delay_before = '延时触发时间必须在10-60秒之间';
    }

    if (formData.cap_price < 0) {
      newErrors.cap_price = '封顶价不能为负数';
    }

    if (formData.start_price < 0) {
      newErrors.start_price = '起拍价不能为负数';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
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
    return (
      <div className="empty-state">
        <div className="loading-spinner"></div>
        <p>加载中...</p>
      </div>
    );
  }

  if (!product) {
    return (
      <div className="empty-state">
        <div className="empty-state-icon">❌</div>
        <div className="empty-state-text">商品不存在</div>
      </div>
    );
  }

  return (
    <div>
      {/* 页面标题 */}
      <div className="page-header">
        <div className="page-title-wrapper">
          <h1 className="page-title">⚙️ 配置竞拍规则</h1>
          <p className="page-subtitle">为商品设置竞拍参数，包括起拍价、时长、延时规则等</p>
        </div>
      </div>

      {/* 商品信息卡片 */}
      <div className="data-table-wrapper" style={{ marginBottom: '24px' }}>
        <div className="data-table-header">
          <h3 className="data-table-title">商品信息</h3>
        </div>
        <div style={{ padding: '20px' }}>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '16px' }}>
            <div>
              <div style={{ color: 'var(--text-muted)', fontSize: '14px', marginBottom: '4px' }}>商品名称</div>
              <div style={{ fontWeight: 600, color: 'var(--text-primary)' }}>{product.name}</div>
            </div>
            <div>
              <div style={{ color: 'var(--text-muted)', fontSize: '14px', marginBottom: '4px' }}>商品描述</div>
              <div style={{ color: 'var(--text-secondary)' }}>{product.description || '暂无描述'}</div>
            </div>
            <div>
              <div style={{ color: 'var(--text-muted)', fontSize: '14px', marginBottom: '4px' }}>商品状态</div>
              <div>
                <span className="status-badge default">草稿</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* 配置表单 */}
      <div className="data-table-wrapper">
        <div className="data-table-header">
          <h3 className="data-table-title">竞拍规则</h3>
        </div>
        <form onSubmit={handleSubmit} style={{ padding: '20px' }}>
          {/* 价格设置 */}
          <div style={{ marginBottom: '32px' }}>
            <h4 style={{ marginBottom: '16px', color: 'var(--text-primary)', borderBottom: '1px solid var(--border-color)', paddingBottom: '8px' }}>
              💰 价格设置
            </h4>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))', gap: '20px' }}>
              <div>
                <label className="form-label">
                  起拍价（元）
                </label>
                <input
                  type="number"
                  value={formData.start_price}
                  onChange={(e) => setFormData({ ...formData, start_price: Number(e.target.value) })}
                  min="0"
                  step="0.01"
                  className="form-input"
                />
                <small className="form-hint">默认 0 元起拍，任何人都可以参与竞拍</small>
                {errors.start_price && <div className="form-error">{errors.start_price}</div>}
              </div>

              <div>
                <label className="form-label">
                  加价幅度（元） <span style={{ color: 'red' }}>*</span>
                </label>
                <input
                  type="number"
                  value={formData.increment}
                  onChange={(e) => setFormData({ ...formData, increment: Number(e.target.value) })}
                  min="0.01"
                  step="0.01"
                  className="form-input"
                  required
                />
                <small className="form-hint">每次出价必须按此幅度递增</small>
                {errors.increment && <div className="form-error">{errors.increment}</div>}
              </div>

              <div>
                <label className="form-label">封顶价（元）</label>
                <input
                  type="number"
                  value={formData.cap_price}
                  onChange={(e) => setFormData({ ...formData, cap_price: Number(e.target.value) })}
                  min="0"
                  step="0.01"
                  className="form-input"
                />
                <small className="form-hint">达到封顶价自动成交，0表示无封顶</small>
                {errors.cap_price && <div className="form-error">{errors.cap_price}</div>}
              </div>
            </div>
          </div>

          {/* 时间设置 */}
          <div style={{ marginBottom: '32px' }}>
            <h4 style={{ marginBottom: '16px', color: 'var(--text-primary)', borderBottom: '1px solid var(--border-color)', paddingBottom: '8px' }}>
              ⏰ 时间设置
            </h4>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))', gap: '20px' }}>
              <div>
                <label className="form-label">
                  竞拍时长（秒） <span style={{ color: 'red' }}>*</span>
                </label>
                <input
                  type="number"
                  value={formData.duration}
                  onChange={(e) => setFormData({ ...formData, duration: Number(e.target.value) })}
                  min="60"
                  max="3600"
                  className="form-input"
                  required
                />
                <small className="form-hint">建议 300 秒（5分钟），最长 3600 秒（1小时）</small>
                {errors.duration && <div className="form-error">{errors.duration}</div>}
              </div>

              <div>
                <label className="form-label">单次延时时长（秒）</label>
                <input
                  type="number"
                  value={formData.delay_duration}
                  onChange={(e) => setFormData({ ...formData, delay_duration: Number(e.target.value) })}
                  min="10"
                  max="60"
                  className="form-input"
                />
                <small className="form-hint">结束前出价触发延时，默认 30 秒，范围 10-60 秒</small>
                {errors.delay_duration && <div className="form-error">{errors.delay_duration}</div>}
              </div>

              <div>
                <label className="form-label">最大延时时长（秒）</label>
                <input
                  type="number"
                  value={formData.max_delay_time}
                  onChange={(e) => setFormData({ ...formData, max_delay_time: Number(e.target.value) })}
                  min="60"
                  max="600"
                  className="form-input"
                />
                <small className="form-hint">总延时不超过此限制，默认 180 秒（3分钟）</small>
                {errors.max_delay_time && <div className="form-error">{errors.max_delay_time}</div>}
              </div>

              <div>
                <label className="form-label">延时触发时间（秒）</label>
                <input
                  type="number"
                  value={formData.trigger_delay_before}
                  onChange={(e) => setFormData({ ...formData, trigger_delay_before: Number(e.target.value) })}
                  min="10"
                  max="60"
                  className="form-input"
                />
                <small className="form-hint">结束前多少秒内的出价触发延时，默认 30 秒</small>
                {errors.trigger_delay_before && <div className="form-error">{errors.trigger_delay_before}</div>}
              </div>
            </div>
          </div>

          {/* 提交按钮 */}
          <div style={{ display: 'flex', gap: '12px', marginTop: '24px' }}>
            <button
              type="submit"
              disabled={submitting}
              className="btn btn-primary"
            >
              {submitting ? '保存中...' : '保存配置'}
            </button>
            <button
              type="button"
              onClick={() => navigate('/products')}
              className="btn btn-secondary"
            >
              取消
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default RuleConfig;
