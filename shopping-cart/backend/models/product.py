import os
import boto3
from decimal import Decimal


class ProductModel:

    def __init__(self):
        dynamodb = boto3.resource("dynamodb")
        self.table = dynamodb.Table(os.environ["PRODUCTS_TABLE"])

    def scan_all(self):
        response = self.table.scan()
        items = response["Items"]
        return [self._serialize(item) for item in items]

    def get(self, product_id):
        response = self.table.get_item(Key={"productId": product_id})
        item = response.get("Item")
        return self._serialize(item) if item else None

    def put(self, product):
        item = {**product}
        if "price" in item:
            item["price"] = Decimal(str(item["price"]))
        self.table.put_item(Item=item)

    def _serialize(self, item):
        if not item:
            return None
        result = {**item}
        if "price" in result:
            result["price"] = float(result["price"])
        return result
