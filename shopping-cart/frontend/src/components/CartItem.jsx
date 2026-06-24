import { useCart } from '../context/CartContext';

function CartItem({ item }) {
  const { updateQuantity, removeItem, loading } = useCart();

  return (
    <div className="cart-item">
      <span className="cart-item-emoji">{item.image}</span>
      <div className="cart-item-details">
        <div className="cart-item-name">{item.name}</div>
        <div className="cart-item-price">${(item.price * item.quantity).toFixed(2)}</div>
      </div>
      <div className="quantity-controls">
        <button
          onClick={() => updateQuantity(item.productId, item.quantity - 1)}
          disabled={loading}
        >
          −
        </button>
        <span>{item.quantity}</span>
        <button
          onClick={() => updateQuantity(item.productId, item.quantity + 1)}
          disabled={loading}
        >
          +
        </button>
      </div>
      <button
        className="btn btn-danger"
        onClick={() => removeItem(item.productId)}
        disabled={loading}
      >
        Remove
      </button>
    </div>
  );
}

export default CartItem;
