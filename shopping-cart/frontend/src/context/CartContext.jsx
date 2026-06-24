import { createContext, useContext, useState, useEffect, useCallback } from 'react';
import * as api from '../services/api';

const CartContext = createContext();

export function CartProvider({ children }) {
  const [cart, setCart] = useState({ items: [], itemCount: 0, subtotal: 0, tax: 0, total: 0 });
  const [loading, setLoading] = useState(false);

  const fetchCart = useCallback(async () => {
    try {
      const data = await api.getCart();
      setCart(data);
    } catch (err) {
      console.error('Failed to fetch cart:', err);
    }
  }, []);

  useEffect(() => {
    fetchCart();
  }, [fetchCart]);

  const addItem = async (productId, quantity = 1) => {
    setLoading(true);
    try {
      await api.addToCart(productId, quantity);
      await fetchCart();
    } finally {
      setLoading(false);
    }
  };

  const updateQuantity = async (productId, quantity) => {
    setLoading(true);
    try {
      await api.updateCartItem(productId, quantity);
      await fetchCart();
    } finally {
      setLoading(false);
    }
  };

  const removeItem = async (productId) => {
    setLoading(true);
    try {
      await api.removeFromCart(productId);
      await fetchCart();
    } finally {
      setLoading(false);
    }
  };

  const clearAll = async () => {
    setLoading(true);
    try {
      await api.clearCart();
      await fetchCart();
    } finally {
      setLoading(false);
    }
  };

  return (
    <CartContext.Provider value={{ cart, loading, addItem, updateQuantity, removeItem, clearAll, fetchCart }}>
      {children}
    </CartContext.Provider>
  );
}

export function useCart() {
  const context = useContext(CartContext);
  if (!context) throw new Error('useCart must be used within CartProvider');
  return context;
}
