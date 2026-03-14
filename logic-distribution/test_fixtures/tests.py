"""Test fixture: Test file — all functions should be TEST_CODE."""
from django.test import TestCase


class ProductTests(TestCase):
    def test_create_product(self):
        pass

    def test_product_price(self):
        pass


def test_standalone_function():
    assert 1 + 1 == 2
