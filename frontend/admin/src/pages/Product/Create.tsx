// pages/Product/Create.tsx

import React, { useState } from 'react';
import { productApi } from '../../services/api';

const ProductCreate: React.FC = () => {
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    images: [] as string[],
  });
  const [imageUrl, setImageUrl] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!formData.name.trim()) {
      alert('请输入商品名称');
      return;
    }

    setLoading(true);
    try {
      const product = await productApi.create(formData);
      alert('商品创建成功！');
      window.location.href = `/products/${product.id}/rules`;
    } catch (error) {
      console.error('创建商品失败:', error);
      alert('创建商品失败');
    } finally {
      setLoading(false);
    }
  };

  const handleAddImage = () => {
    if (imageUrl.trim()) {
      setFormData({
        ...formData,
        images: [...formData.images, imageUrl.trim()],
      });
      setImageUrl('');
    }
  };

  const handleRemoveImage = (index: number) => {
    setFormData({
      ...formData,
      images: formData.images.filter((_, i) => i !== index),
    });
  };

  return (
    <div style={{ padding: '20px', maxWidth: '800px', margin: '0 auto' }}>
      <h1>新建商品</h1>

      <form onSubmit={handleSubmit}>
        <div style={formItemStyle}>
          <label style={labelStyle}>
            商品名称 <span style={{ color: 'red' }}>*</span>
          </label>
          <input
            type="text"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            placeholder="请输入商品名称"
            style={inputStyle}
            maxLength={128}
          />
        </div>

        <div style={formItemStyle}>
          <label style={labelStyle}>商品描述</label>
          <textarea
            value={formData.description}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
            placeholder="请输入商品描述"
            style={{ ...inputStyle, height: '100px', resize: 'vertical' }}
          />
        </div>

        <div style={formItemStyle}>
          <label style={labelStyle}>商品图片</label>
          <div style={{ display: 'flex', gap: '10px', marginBottom: '10px' }}>
            <input
              type="text"
              value={imageUrl}
              onChange={(e) => setImageUrl(e.target.value)}
              placeholder="请输入图片URL"
              style={{ ...inputStyle, flex: 1 }}
            />
            <button
              type="button"
              onClick={handleAddImage}
              style={{ ...buttonStyle, padding: '8px 16px' }}
            >
              添加
            </button>
          </div>
          {formData.images.length > 0 && (
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '10px' }}>
              {formData.images.map((img, index) => (
                <div
                  key={index}
                  style={{
                    position: 'relative',
                    width: '100px',
                    height: '100px',
                    border: '1px solid #ddd',
                    borderRadius: '4px',
                    overflow: 'hidden',
                  }}
                >
                  <img
                    src={img}
                    alt={`商品图片${index + 1}`}
                    style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                    onError={(e) => {
                      (e.target as HTMLImageElement).src = 'https://via.placeholder.com/100?text=Error';
                    }}
                  />
                  <button
                    type="button"
                    onClick={() => handleRemoveImage(index)}
                    style={{
                      position: 'absolute',
                      top: '2px',
                      right: '2px',
                      width: '20px',
                      height: '20px',
                      borderRadius: '50%',
                      border: 'none',
                      backgroundColor: 'rgba(0,0,0,0.5)',
                      color: 'white',
                      cursor: 'pointer',
                    }}
                  >
                    ×
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        <div style={{ marginTop: '30px', display: 'flex', gap: '10px' }}>
          <button
            type="submit"
            disabled={loading}
            style={{
              ...buttonStyle,
              padding: '10px 30px',
              opacity: loading ? 0.5 : 1,
            }}
          >
            {loading ? '提交中...' : '提交'}
          </button>
          <button
            type="button"
            onClick={() => window.location.href = '/products'}
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

export default ProductCreate;
