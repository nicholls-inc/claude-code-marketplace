"""Test fixture: Django model methods."""
from django.db import models
from django.db.models import Q, F, Sum
from django.dispatch import receiver
from django.db.models.signals import pre_save


class Product(models.Model):
    name = models.CharField(max_length=200)
    price = models.DecimalField(max_digits=10, decimal_places=2)
    stock = models.IntegerField(default=0)

    def clean(self):
        """MODEL_VALIDATION: model clean method."""
        if self.price < 0:
            raise ValueError("Price cannot be negative")

    def save(self, *args, **kwargs):
        """MODEL_VALIDATION: save override."""
        self.full_clean()
        super().save(*args, **kwargs)

    @property
    def is_available(self):
        """MODEL_VALIDATION: model property."""
        return self.stock > 0

    def get_discounted_price(self, discount_pct):
        """MODEL_VALIDATION: custom model method (no ORM)."""
        return self.price * (1 - discount_pct / 100)

    def get_related_products(self):
        """DATABASE_ORM: model method WITH ORM query."""
        return Product.objects.filter(
            category=self.category
        ).exclude(pk=self.pk)[:10]

    def total_revenue(self):
        """DATABASE_ORM: model method with aggregate."""
        return self.orderitem_set.aggregate(
            total=Sum(F('quantity') * F('unit_price'))
        )['total']


@receiver(pre_save, sender=Product)
def validate_product(sender, instance, **kwargs):
    """MODEL_VALIDATION: signal handler."""
    if instance.price and instance.price < 0:
        raise ValueError("Negative price")
