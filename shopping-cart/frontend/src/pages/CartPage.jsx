import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import CartItem from '../components/CartItem';
import { useCart } from '../context/CartContext';
import * as api from '../services/api';

function CartPage() {
  const { cart, clearAll } = useCart();
  const [address, setAddress] = useState('');
  const [ordering, setOrdering] = useState(false);
  const navigate = useNavigate();

  async function handleCheckout() {
    if (!address.trim()) {
      alert('Please enter a shipping address');
      return;
    }

    setOrdering(true);
    try {
      await api.createOrder(address);
      alert('Order placed successfully!');
      setAddress('');
      navigate('/orders');
    } catch (err) {
      alert(`Order failed: ${err.message}`);
    } finally {
      setOrdering(false);
    }
  }

  if (cart.items.length === 0) {
    return (
      <div className="empty-state">
        <div className="empty-state-emoji">🛒</div>
        <h2>Your cart is empty</h2>
        <p>Add some products to get started!</p>
      </div>
    );
  }

  return (
    <div className="cart-container">
      <h2>Shopping Cart</h2>
      <div style={{ marginTop: '1rem' }}>
        {cart.items.map((item) => (
          <CartItem key={item.productId} item={item} />
        ))}
      </div>

      <div className="cart-summary">
        <div className="cart-summary-row">
          <span>Subtotal ({cart.itemCount} items)</span>
          <span>${cart.subtotal.toFixed(2)}</span>
        </div>
        <div className="cart-summary-row">
          <span>Tax (8%)</span>
          <span>${cart.tax.toFixed(2)}</span>
        </div>
        <div className="cart-summary-row total">
          <span>Total</span>
          <span>${cart.total.toFixed(2)}</span>
        </div>

        <div style={{ marginTop: '1.5rem' }}>
          <input
            type="text"
            placeholder="Enter shipping address..."
            value={address}
            onChange={(e) => setAddress(e.target.value)}
            style={{ width: '100%', padding: '0.6rem', marginBottom: '1rem', borderRadius: '4px', border: '1px solid #ddd' }}
          />
          <div style={{ display: 'flex', gap: '1rem' }}>
            <button className="btn btn-primary" onClick={handleCheckout} disabled={ordering}>
              {ordering ? 'Placing Order...' : 'Place Order'}
            </button>
            <button className="btn btn-secondary" onClick={clearAll}>
              Clear Cart
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

export default CartPage;
