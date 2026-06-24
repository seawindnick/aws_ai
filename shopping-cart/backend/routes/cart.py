from flask import Blueprint, jsonify, request
from services.cart_service import CartService

cart_bp = Blueprint("cart", __name__)
cart_service = CartService()

SESSION_ID = "workshop-user"


@cart_bp.route("", methods=["GET"])
def get_cart():
    cart = cart_service.get_cart(SESSION_ID)
    return jsonify(cart)


@cart_bp.route("/items", methods=["POST"])
def add_item():
    body = request.get_json()
    if not body or "productId" not in body:
        return jsonify({"error": "productId is required"}), 400

    quantity = body.get("quantity", 1)
    if quantity < 1:
        return jsonify({"error": "Quantity must be at least 1"}), 400

    result = cart_service.add_item(SESSION_ID, body["productId"], quantity)
    if "error" in result:
        return jsonify(result), 404
    return jsonify(result), 201


@cart_bp.route("/items/<product_id>", methods=["PUT"])
def update_item(product_id):
    body = request.get_json()
    if not body or "quantity" not in body:
        return jsonify({"error": "quantity is required"}), 400

    quantity = body["quantity"]
    if quantity < 0:
        return jsonify({"error": "Quantity cannot be negative"}), 400

    if quantity == 0:
        result = cart_service.remove_item(SESSION_ID, product_id)
    else:
        result = cart_service.update_quantity(SESSION_ID, product_id, quantity)

    if "error" in result:
        return jsonify(result), 404
    return jsonify(result)


@cart_bp.route("/items/<product_id>", methods=["DELETE"])
def remove_item(product_id):
    result = cart_service.remove_item(SESSION_ID, product_id)
    if "error" in result:
        return jsonify(result), 404
    return jsonify(result)


@cart_bp.route("", methods=["DELETE"])
def clear_cart():
    cart_service.clear_cart(SESSION_ID)
    return jsonify({"message": "Cart cleared"})
