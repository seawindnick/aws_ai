import { useState, useEffect } from 'react';
import * as api from '../services/api';

function OrdersPage() {
  const [orders, setOrders] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadOrders();
  }, []);

  async function loadOrders() {
    try {
      const data = await api.getOrders();
      setOrders(data.orders);
    } catch (err) {
      console.error('Failed to load orders:', err);
    } finally {
      setLoading(false);
    }
  }

  if (loading) return <div className="loading">Loading orders...</div>;

  if (orders.length === 0) {
    return (
      <div className="empty-state">
        <div className="empty-state-emoji">📦</div>
        <h2>No orders yet</h2>
        <p>Place an order from your cart to see it here.</p>
      </div>
    );
  }

  return (
    <div>
      <h2>Order History</h2>
      <div style={{ marginTop: '1rem' }}>
        {orders.map((order) => (
          <div key={order.orderId} className="order-card">
            <div className="order-header">
              <div>
                <strong>Order #{order.orderId}</strong>
                <div style={{ fontSize: '0.85rem', color: '#666' }}>{order.createdAt}</div>
              </div>
              <span className="order-status">{order.status}</span>
            </div>
            <div>
              {order.items.map((item, i) => (
                <div key={i} style={{ display: 'flex', gap: '0.5rem', padding: '0.3rem 0' }}>
                  <span>{item.image}</span>
                  <span>{item.name} × {item.quantity}</span>
                  <span style={{ marginLeft: 'auto', color: '#b12704' }}>
                    ${(item.price * item.quantity).toFixed(2)}
                  </span>
                </div>
              ))}
            </div>
            <div style={{ marginTop: '0.8rem', textAlign: 'right', fontWeight: 'bold' }}>
              Total: ${order.total.toFixed(2)}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

export default OrdersPage;
