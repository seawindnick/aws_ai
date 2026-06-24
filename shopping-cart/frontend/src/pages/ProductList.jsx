import { useState, useEffect } from 'react';
import ProductCard from '../components/ProductCard';
import * as api from '../services/api';

function ProductList() {
  const [products, setProducts] = useState([]);
  const [categories, setCategories] = useState([]);
  const [activeCategory, setActiveCategory] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadProducts();
    loadCategories();
  }, []);

  async function loadProducts(category = null) {
    setLoading(true);
    try {
      const data = await api.getProducts(category);
      setProducts(data.products);
    } catch (err) {
      console.error('Failed to load products:', err);
    } finally {
      setLoading(false);
    }
  }

  async function loadCategories() {
    try {
      const data = await api.getCategories();
      setCategories(data.categories);
    } catch (err) {
      console.error('Failed to load categories:', err);
    }
  }

  function handleCategoryFilter(category) {
    if (activeCategory === category) {
      setActiveCategory(null);
      loadProducts(null);
    } else {
      setActiveCategory(category);
      loadProducts(category);
    }
  }

  if (loading && products.length === 0) {
    return <div className="loading">Loading products...</div>;
  }

  return (
    <div>
      <div className="category-filter">
        <button
          className={`category-btn ${!activeCategory ? 'active' : ''}`}
          onClick={() => handleCategoryFilter(null)}
        >
          All
        </button>
        {categories.map((cat) => (
          <button
            key={cat}
            className={`category-btn ${activeCategory === cat ? 'active' : ''}`}
            onClick={() => handleCategoryFilter(cat)}
          >
            {cat}
          </button>
        ))}
      </div>

      <div className="product-grid">
        {products.map((product) => (
          <ProductCard key={product.productId} product={product} />
        ))}
      </div>
    </div>
  );
}

export default ProductList;
