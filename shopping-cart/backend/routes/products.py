from flask import Blueprint, jsonify, request
from services.product_service import ProductService

products_bp = Blueprint("products", __name__)
product_service = ProductService()


@products_bp.route("", methods=["GET"])
def list_products():
    category = request.args.get("category")
    if category:
        products = product_service.get_by_category(category)
    else:
        products = product_service.get_all()
    return jsonify({"products": products, "count": len(products)})


@products_bp.route("/<product_id>", methods=["GET"])
def get_product(product_id):
    product = product_service.get_by_id(product_id)
    if not product:
        return jsonify({"error": "Product not found"}), 404
    return jsonify(product)


@products_bp.route("/categories", methods=["GET"])
def list_categories():
    categories = product_service.get_categories()
    return jsonify({"categories": categories})


@products_bp.route("/search", methods=["GET"])
def search_products():
    query = request.args.get("q", "")
    if not query:
        return jsonify({"error": "Search query 'q' is required"}), 400
    results = product_service.search(query)
    return jsonify({"products": results, "count": len(results)})
