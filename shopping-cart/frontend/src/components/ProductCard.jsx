import { useCart } from '../context/CartContext';

function ProductCard({ product }) {
  const { addItem, loading } = useCart();

  return (
    <div className="product-card">
      <div className="product-emoji">{product.image}</div>
      <div className="product-category">{product.category}</div>
      <div className="product-name">{product.name}</div>
      <div className="product-description">{product.description}</div>
      <div className="product-price">${product.price.toFixed(2)}</div>
      <button
        className="btn btn-primary"
        onClick={() => addItem(product.productId)}
        disabled={loading}
      >
        Add to Cart
      </button>
    </div>
  );
}

export default ProductCard;
