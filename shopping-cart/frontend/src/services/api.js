/**
 * API service - handles all HTTP requests to the backend.
 * Uses the API Gateway URL configured at build time.
 */
const API_BASE = import.meta.env.VITE_API_URL || '/api';

async function request(path, options = {}) {
  const url = `${API_BASE}${path}`;
  const response = await fetch(url, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Request failed' }));
    throw new Error(error.error || `HTTP ${response.status}`);
  }

  return response.json();
}

// Product endpoints
export const getProducts = (category) => {
  const params = category ? `?category=${encodeURIComponent(category)}` : '';
  return request(`/products${params}`);
};

export const getProduct = (id) => request(`/products/${id}`);
export const getCategories = () => request('/products/categories');
export const searchProducts = (query) => request(`/products/search?q=${encodeURIComponent(query)}`);

// Cart endpoints
export const getCart = () => request('/cart');

export const addToCart = (productId, quantity = 1) =>
  request('/cart/items', {
    method: 'POST',
    body: JSON.stringify({ productId, quantity }),
  });

export const updateCartItem = (productId, quantity) =>
  request(`/cart/items/${productId}`, {
    method: 'PUT',
    body: JSON.stringify({ quantity }),
  });

export const removeFromCart = (productId) =>
  request(`/cart/items/${productId}`, { method: 'DELETE' });

export const clearCart = () => request('/cart', { method: 'DELETE' });

// Order endpoints
export const createOrder = (shippingAddress) =>
  request('/orders', {
    method: 'POST',
    body: JSON.stringify({ shippingAddress }),
  });

export const getOrders = () => request('/orders');
export const getOrder = (orderId) => request(`/orders/${orderId}`);
