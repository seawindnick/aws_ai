from decimal import Decimal
from models.cart import CartModel
from services.product_service import ProductService


class CartService:

    def __init__(self):
        self.model = CartModel()
        self.product_service = ProductService()

    def get_cart(self, session_id):
        items = self.model.get_items(session_id)
        subtotal = sum(
            Decimal(str(item["price"])) * int(item["quantity"]) for item in items
        )
        item_count = sum(int(item["quantity"]) for item in items)

        return {
            "items": items,
            "itemCount": item_count,
            "subtotal": float(subtotal),
            "tax": float(subtotal * Decimal("0.08")),
            "total": float(subtotal * Decimal("1.08")),
        }

    def add_item(self, session_id, product_id, quantity=1):
        product = self.product_service.get_by_id(product_id)
        if not product:
            return {"error": "Product not found"}

        existing = self.model.get_item(session_id, product_id)
        if existing:
            new_quantity = int(existing["quantity"]) + quantity
        else:
            new_quantity = quantity

        cart_item = {
            "sessionId": session_id,
            "productId": product_id,
            "name": product["name"],
            "price": product["price"],
            "image": product["image"],
            "quantity": new_quantity,
        }
        self.model.put_item(cart_item)

        return {
            "message": f"Added {product['name']} to cart",
            "item": cart_item,
        }

    def update_quantity(self, session_id, product_id, quantity):
        existing = self.model.get_item(session_id, product_id)
        if not existing:
            return {"error": "Item not in cart"}

        existing["quantity"] = quantity
        self.model.put_item(existing)
        return {"message": "Quantity updated", "item": existing}

    def remove_item(self, session_id, product_id):
        existing = self.model.get_item(session_id, product_id)
        if not existing:
            return {"error": "Item not in cart"}

        self.model.delete_item(session_id, product_id)
        return {"message": f"Removed {existing['name']} from cart"}

    def clear_cart(self, session_id):
        self.model.clear(session_id)
