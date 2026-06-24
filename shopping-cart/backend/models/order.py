import os
import boto3
from decimal import Decimal
from boto3.dynamodb.conditions import Key


class OrderModel:

    def __init__(self):
        dynamodb = boto3.resource("dynamodb")
        self.table = dynamodb.Table(os.environ["ORDERS_TABLE"])

    def put(self, order):
        db_item = self._deserialize(order)
        self.table.put_item(Item=db_item)

    def get(self, session_id, order_id):
        response = self.table.get_item(
            Key={"sessionId": session_id, "orderId": order_id}
        )
        item = response.get("Item")
        return self._serialize(item) if item else None

    def get_by_session(self, session_id):
        response = self.table.query(
            KeyConditionExpression=Key("sessionId").eq(session_id)
        )
        return [self._serialize(item) for item in response["Items"]]

    def _deserialize(self, order):
        db_item = {**order}
        for key in ["subtotal", "tax", "total"]:
            if key in db_item:
                db_item[key] = Decimal(str(db_item[key]))
        if "items" in db_item:
            db_item["items"] = [
                {**item, "price": Decimal(str(item["price"]))} for item in db_item["items"]
            ]
        return db_item

    def _serialize(self, item):
        if not item:
            return None
        result = {**item}
        for key in ["subtotal", "tax", "total"]:
            if key in result:
                result[key] = float(result[key])
        if "items" in result:
            result["items"] = [
                {**i, "price": float(i["price"])} for i in result["items"]
            ]
        return result
