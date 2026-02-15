"""Ecommerce platform — simüle e-ticaret servisi.

Discit'in SimEcommerceProvider'ı şu endpointleri çağırır:
  POST /ecommerce/orders
  GET  /ecommerce/orders/{order_id}
  GET  /ecommerce/orders
  GET  /ecommerce/customers/{customer_id}
"""
from fastapi import APIRouter

router = APIRouter()

_orders: list[dict] = []
_order_counter = 0


@router.post("/orders")
async def create_order(order_data: dict = {}):
    global _order_counter
    _order_counter += 1
    order = {"id": _order_counter, "status": "created", **order_data}
    _orders.append(order)
    return order


@router.get("/orders/{order_id}")
async def get_order(order_id: int):
    for o in _orders:
        if o["id"] == order_id:
            return o
    return {"id": order_id, "status": "not_found"}


@router.get("/orders")
async def list_orders():
    return {"orders": _orders}


@router.get("/customers/{customer_id}")
async def get_customer(customer_id: int):
    return {"id": customer_id, "name": "sim-customer"}
