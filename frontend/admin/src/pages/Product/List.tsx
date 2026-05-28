// pages/Product/List.tsx

import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
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
      setProducts(result.items || []);
      setTotal(result.total || 0);
    } catch (error) {
      console.error('获取商品列表失败:', error);
      // 模拟数据
      setProducts([
        { id: 1, name: '稀有珠宝', description: '限量版珠宝，全球仅发售10件', status: ProductStatus.Published, images: [], created_at: new Date().toISOString() },
        { id: 2, name: '签名版限量球鞋', description: '球星亲笔签名，收藏价值极高', status: ProductStatus.Draft, images: [], created_at: new Date().toISOString() },
        { id: 3, name: '古董怀表收藏品', description: '19世纪瑞士制造，品相完美', status: ProductStatus.Draft, images: [], created_at: new Date().toISOString() },
        { id: 4, name: '限定款奢侈品包包', description: '2024限量款，全新未拆封', status: ProductStatus.Draft, images: [], created_at: new Date().toISOString() },
        { id: 5, name: '艺术画作原稿', description: '知名艺术家原创作品', status: ProductStatus.Published, images: [], created_at: new Date().toISOString() },
      ]);
      setTotal(5);
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

  const handlePublish = async (id: number) => {
    if (!window.confirm('确定要发布这个商品吗？发布后将创建竞拍记录。')) return;
    try {
      const token = localStorage.getItem('token');
      const response = await fetch(`/api/v1/products/${id}/publish`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({}),
      });

      if (!response.ok) {
        throw new Error('发布失败');
      }

      alert('发布成功！');
      fetchProducts();
    } catch (error) {
      console.error('发布商品失败:', error);
      alert('发布失败，请重试');
    }
  };

  const handleUnpublish = async (id: number) => {
    const reason = prompt('请输入下架原因（可选）：');
    if (!window.confirm('确定要下架这个商品吗？这将中断正在进行的拍卖。')) return;

    try {
      const token = localStorage.getItem('token');
      const response = await fetch(`/api/v1/products/${id}/unpublish`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ reason: reason || '' }),
      });

      if (!response.ok) {
        throw new Error('下架失败');
      }

      alert('下架成功！');
      fetchProducts();
    } catch (error) {
      console.error('下架商品失败:', error);
      alert('下架失败，请重试');
    }
  };

  const getStatusConfig = (status: ProductStatus) => {
    switch (status) {
      case ProductStatus.Published:
        return { text: '已发布', class: 'success' };
      case ProductStatus.Unpublished:
        return { text: '已下架', class: 'warning' };
      default:
        return { text: '草稿', class: 'default' };
    }
  };

  const totalPages = Math.ceil(total / 10);

  if (loading) {
    return (
      <div className="empty-state">
        <div className="loading-spinner"></div>
        <p style={{ marginTop: '16px' }}>加载中...</p>
      </div>
    );
  }

  return (
    <div>
      {/* 页面标题 */}
      <div className="page-header">
        <div className="page-title-wrapper">
          <h1 className="page-title">📦 商品管理</h1>
          <p className="page-subtitle">管理所有竞拍商品，配置竞拍规则</p>
        </div>
        <Link to="/products/create">
          <button className="btn btn-primary">
            <span>＋</span>
            新建商品
          </button>
        </Link>
      </div>

      {/* 统计卡片 */}
      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-card-header">
            <div className="stat-card-icon blue">📦</div>
          </div>
          <div className="stat-card-value">{total}</div>
          <div className="stat-card-label">商品总数</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-header">
            <div className="stat-card-icon green">✓</div>
          </div>
          <div className="stat-card-value">
            {products.filter(p => p.status === ProductStatus.Published).length}
          </div>
          <div className="stat-card-label">已发布</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-header">
            <div className="stat-card-icon gold">📝</div>
          </div>
          <div className="stat-card-value">
            {products.filter(p => p.status === ProductStatus.Draft).length}
          </div>
          <div className="stat-card-label">草稿</div>
        </div>
      </div>

      {/* 数据表格 */}
      <div className="data-table-wrapper">
        <div className="data-table-header">
          <h3 className="data-table-title">商品列表</h3>
          <div className="data-table-actions">
            <button className="btn btn-secondary btn-sm">导出数据</button>
          </div>
        </div>
        <table className="data-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>商品名称</th>
              <th>描述</th>
              <th>状态</th>
              <th>创建时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {products.map((product) => {
              const statusConfig = getStatusConfig(product.status);
              return (
                <tr key={product.id}>
                  <td style={{ color: 'var(--accent-primary)', fontWeight: 600 }}>
                    #{product.id}
                  </td>
                  <td style={{ color: 'var(--text-primary)', fontWeight: 500 }}>
                    {product.name}
                  </td>
                  <td style={{ maxWidth: '200px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {product.description}
                  </td>
                  <td>
                    <span className={`status-badge ${statusConfig.class}`}>
                      {statusConfig.text}
                    </span>
                  </td>
                  <td>{new Date(product.created_at).toLocaleString('zh-CN')}</td>
                  <td>
                    <div className="action-buttons">
                      {product.status === ProductStatus.Draft && (
                        <button
                          className="btn btn-primary btn-sm"
                          onClick={() => handlePublish(product.id)}
                        >
                          发布
                        </button>
                      )}
                      {product.status === ProductStatus.Published && (
                        <button
                          className="btn btn-warning btn-sm"
                          onClick={() => handleUnpublish(product.id)}
                        >
                          下架
                        </button>
                      )}
                      <Link to={`/products/${product.id}/edit`}>
                        <button className="btn btn-secondary btn-sm">编辑</button>
                      </Link>
                      <Link to={`/products/${product.id}/rules`}>
                        <button className="btn btn-secondary btn-sm">配置规则</button>
                      </Link>
                      <button
                        className="btn btn-danger btn-sm"
                        onClick={() => handleDelete(product.id)}
                      >
                        删除
                      </button>
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>

        {/* 分页 */}
        <div className="pagination">
          <button
            className="pagination-btn"
            disabled={page <= 1}
            onClick={() => setPage(page - 1)}
          >
            ← 上一页
          </button>
          <span className="pagination-info">
            第 {page} 页 / 共 {totalPages || 1} 页
          </span>
          <button
            className="pagination-btn"
            disabled={page >= totalPages}
            onClick={() => setPage(page + 1)}
          >
            下一页 →
          </button>
        </div>
      </div>
    </div>
  );
};

export default ProductList;
