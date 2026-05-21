// pages/Product/List.tsx

import React, { useState, useEffect } from 'react';
import { productApi } from '../../services/api';
import { Product, ProductStatus } from '../../types';

const ProductList: React.FC = () => {
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);

  useEffect(() => {
    fetchProducts();
  }, [page]);

  const fetchProducts = async () => {
    setLoading(true);
    try {
      const result = await productApi.list({ page, page_size: 10 });
      setProducts(result.items);
      setTotal(result.total);
    } catch (error) {
      console.error('获取商品列表失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id: number) => {
    if (!window.confirm('确定要删除这个商品吗？')) return;

    try {
      await productApi.delete(id);
      fetchProducts();
    } catch (error) {
      console.error('删除商品失败:', error);
    }
  };

  const getStatusText = (status: ProductStatus) => {
    return status === ProductStatus.Published ? '已发布' : '草稿';
  };

  const getStatusColor = (status: ProductStatus) => {
    return status === ProductStatus.Published ? 'green' : 'gray';
  };

  if (loading) {
    return <div>加载中...</div>;
  }

  return (
    <div style={{ padding: '20px' }}>
      <h1>商品列表</h1>

      <div style={{ marginBottom: '20px' }}>
        <a href="/products/create">
          <button style={{
            padding: '10px 20px',
            backgroundColor: '#1890ff',
            color: 'white',
            border: 'none',
            borderRadius: '4px',
            cursor: 'pointer'
          }}>
            新建商品
          </button>
        </a>
      </div>

      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead>
          <tr style={{ backgroundColor: '#f5f5f5' }}>
            <th style={thStyle}>ID</th>
            <th style={thStyle}>商品名称</th>
            <th style={thStyle}>描述</th>
            <th style={thStyle}>状态</th>
            <th style={thStyle}>创建时间</th>
            <th style={thStyle}>操作</th>
          </tr>
        </thead>
        <tbody>
          {products.map((product) => (
            <tr key={product.id} style={{ borderBottom: '1px solid #eee' }}>
              <td style={tdStyle}>{product.id}</td>
              <td style={tdStyle}>{product.name}</td>
              <td style={tdStyle}>{product.description}</td>
              <td style={tdStyle}>
                <span style={{
                  padding: '2px 8px',
                  borderRadius: '4px',
                  backgroundColor: getStatusColor(product.status) === 'green' ? '#f6ffed' : '#f5f5f5',
                  color: getStatusColor(product.status) === 'green' ? '#52c41a' : '#666'
                }}>
                  {getStatusText(product.status)}
                </span>
              </td>
              <td style={tdStyle}>{new Date(product.created_at).toLocaleString()}</td>
              <td style={tdStyle}>
                <a href={`/products/${product.id}/rules`}>
                  <button style={buttonStyle}>配置规则</button>
                </a>
                <button
                  style={{ ...buttonStyle, backgroundColor: '#ff4d4f', marginLeft: '8px' }}
                  onClick={() => handleDelete(product.id)}
                >
                  删除
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      <div style={{ marginTop: '20px', textAlign: 'center' }}>
        <button
          disabled={page <= 1}
          onClick={() => setPage(page - 1)}
          style={{ ...buttonStyle, opacity: page <= 1 ? 0.5 : 1 }}
        >
          上一页
        </button>
        <span style={{ margin: '0 20px' }}>
          第 {page} 页 / 共 {Math.ceil(total / 10)} 页
        </span>
        <button
          disabled={page * 10 >= total}
          onClick={() => setPage(page + 1)}
          style={{ ...buttonStyle, opacity: page * 10 >= total ? 0.5 : 1 }}
        >
          下一页
        </button>
      </div>
    </div>
  );
};

const thStyle: React.CSSProperties = {
  padding: '12px',
  textAlign: 'left',
  borderBottom: '1px solid #ddd',
};

const tdStyle: React.CSSProperties = {
  padding: '12px',
  borderBottom: '1px solid #eee',
};

const buttonStyle: React.CSSProperties = {
  padding: '4px 12px',
  backgroundColor: '#1890ff',
  color: 'white',
  border: 'none',
  borderRadius: '4px',
  cursor: 'pointer',
};

export default ProductList;
