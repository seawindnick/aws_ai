from models.product import ProductModel


class ProductService:

    def __init__(self):
        self.model = ProductModel()

    def get_all(self):
        products = self.model.scan_all()
        if not products:
            self._seed_products()
            products = self.model.scan_all()
        return products

    def get_by_id(self, product_id):
        return self.model.get(product_id)

    def get_by_category(self, category):
        all_products = self.get_all()
        return [p for p in all_products if p.get("category", "").lower() == category.lower()]

    def get_categories(self):
        products = self.get_all()
        categories = list(set(p.get("category", "Uncategorized") for p in products))
        categories.sort()
        return categories

    def search(self, query):
        all_products = self.get_all()
        query_lower = query.lower()
        return [p for p in all_products if query_lower in p.get("name", "").lower()]

    def _seed_products(self):
        default_products = [
            {"productId": "1", "name": "Wireless Headphones", "price": "49.99", "category": "Electronics", "image": "🎧", "description": "Premium wireless headphones with noise cancellation."},
            {"productId": "2", "name": "USB-C Cable", "price": "12.99", "category": "Electronics", "image": "🔌", "description": "Fast-charging USB-C cable, 2m braided nylon."},
            {"productId": "3", "name": "Laptop Stand", "price": "34.99", "category": "Accessories", "image": "💻", "description": "Ergonomic aluminum laptop stand with adjustable height."},
            {"productId": "4", "name": "Mechanical Keyboard", "price": "79.99", "category": "Electronics", "image": "⌨️", "description": "RGB mechanical keyboard with Cherry MX switches."},
            {"productId": "5", "name": "Mouse Pad XL", "price": "9.99", "category": "Accessories", "image": "🖱️", "description": "Extended mouse pad with stitched edges, 900x400mm."},
            {"productId": "6", "name": "Webcam HD", "price": "59.99", "category": "Electronics", "image": "📷", "description": "1080p webcam with built-in microphone and auto-focus."},
            {"productId": "7", "name": "Coffee Mug", "price": "14.99", "category": "Office", "image": "☕", "description": "Insulated stainless steel mug, keeps drinks hot for 6 hours."},
            {"productId": "8", "name": "Desk Lamp", "price": "29.99", "category": "Office", "image": "💡", "description": "LED desk lamp with 5 brightness levels and USB charging port."},
            {"productId": "9", "name": "Notebook Pack", "price": "7.99", "category": "Office", "image": "📓", "description": "3-pack dotted grid notebooks, A5 size, 120 pages each."},
            {"productId": "10", "name": "Backpack", "price": "44.99", "category": "Accessories", "image": "🎒", "description": "Water-resistant laptop backpack with USB charging port."},
        ]
        for product in default_products:
            self.model.put(product)
