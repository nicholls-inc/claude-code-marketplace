"""Test fixture: External IO and mixed functions."""
import requests
from django.core.mail import send_mail
from django.db import transaction
from celery import shared_task


def fetch_exchange_rate(currency_pair):
    """EXTERNAL_IO: HTTP request."""
    response = requests.get(f"https://api.example.com/rates/{currency_pair}")
    return response.json()["rate"]


def export_report(data, filename):
    """EXTERNAL_IO: file IO."""
    with open(filename, "w") as f:
        f.write(str(data))


@shared_task
def send_order_confirmation(order_id):
    """EXTERNAL_IO: celery task + email."""
    send_mail(
        "Order Confirmation",
        f"Your order {order_id} has been confirmed.",
        "noreply@example.com",
        ["customer@example.com"],
    )


def process_payment_with_query(order_id):
    """DATABASE_ORM: has both ORM and external IO, ORM wins by priority."""
    from .models import Order
    order = Order.objects.get(pk=order_id)
    response = requests.post("https://payments.example.com/charge", json={
        "amount": str(order.total),
    })
    return response.json()
