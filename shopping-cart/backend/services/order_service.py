import uuid
from datetime import datetime, timezone
from models.order import OrderModel


class OrderService:

    def __init__(self):
        self.model = OrderModel()

    def create_order(self, session_id, items, subtotal, shipping_address):
        order_id = str(uuid.uuid4())[:8]
        tax = round(subtotal * 0.08, 2)
        total = round(subtotal + tax, 2)

        order = {
            "sessionId": session_id,
            "orderId": order_id,
            "items": items,
            "subtotal": str(subtotal),
            "tax": str(tax),
            "total": str(total),
            "shippingAddress": shipping_address,
            "status": "confirmed",
            "createdAt": datetime.now(timezone.utc).isoformat(),
        }

        self.model.put(order)
        return order

    def get_orders(self, session_id):
        return self.model.get_by_session(session_id)

    def get_order(self, session_id, order_id):
        return self.model.get(session_id, order_id)
