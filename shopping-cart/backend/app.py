from flask import Flask
from flask_cors import CORS
from routes.products import products_bp
from routes.cart import cart_bp
from routes.orders import orders_bp

app = Flask(__name__)
CORS(app)

app.register_blueprint(products_bp, url_prefix="/api/products")
app.register_blueprint(cart_bp, url_prefix="/api/cart")
app.register_blueprint(orders_bp, url_prefix="/api/orders")


@app.route("/api/health")
def health_check():
    return {"status": "healthy", "service": "shopping-cart-api"}


if __name__ == "__main__":
    app.run(debug=True, port=5000)
