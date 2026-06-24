import os
import boto3
from decimal import Decimal


class CartModel:

    def __init__(self):
        dynamodb = boto3.resource("dynamodb")
        self.table = dynamodb.Table(os.environ["CART_TABLE"])

    def get_items(self, session_id):
        response = self.table.query(
            KeyConditionExpression=boto3.dynamodb.conditions.Key("sessionId").eq(session_id)
        )
        return [self._serialize(item) for item in response["Items"]]

    def get_item(self, session_id, product_id):
        response = self.table.get_item(
            Key={"sessionId": session_id, "productId": product_id}
        )
        item = response.get("Item")
        return self._serialize(item) if item else None

    def put_item(self, item):
        db_item = {}
        for key, val in item.items():
            if isinstance(val, float):
                db_item[key] = Decimal(str(val))
            elif isinstance(val, int) and not isinstance(val, bool):
                db_item[key] = val
            else:
                db_item[key] = val
        self.table.put_item(Item=db_item)

    def delete_item(self, session_id, product_id):
        self.table.delete_item(
            Key={"sessionId": session_id, "productId": product_id}
        )

    def clear(self, session_id):
        items = self.get_items(session_id)
        for item in items:
            self.delete_item(session_id, item["productId"])

    def _serialize(self, item):
        if not item:
            return None
        result = {**item}
        if "price" in result:
            result["price"] = float(result["price"])
        if "quantity" in result:
            result["quantity"] = int(result["quantity"])
        return result
