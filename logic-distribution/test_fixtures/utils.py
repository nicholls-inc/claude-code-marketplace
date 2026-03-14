"""Test fixture: Pure functions and external IO."""
import math
import re
from decimal import Decimal
from collections import defaultdict


def calculate_tax(price, tax_rate):
    """PURE_FUNCTION: simple arithmetic."""
    return price * (1 + tax_rate / 100)


def format_currency(amount, currency="USD"):
    """PURE_FUNCTION: string formatting."""
    symbols = {"USD": "$", "EUR": "€", "GBP": "£"}
    symbol = symbols.get(currency, currency)
    return f"{symbol}{amount:,.2f}"


def parse_sku(sku_string):
    """PURE_FUNCTION: regex parsing."""
    match = re.match(r"^([A-Z]{2,4})-(\d{4,8})(-[A-Z])?$", sku_string)
    if not match:
        return None
    return {
        "prefix": match.group(1),
        "number": int(match.group(2)),
        "variant": match.group(3),
    }


def group_by_category(items):
    """PURE_FUNCTION: data transformation."""
    groups = defaultdict(list)
    for item in items:
        groups[item["category"]].append(item)
    return dict(groups)


def haversine_distance(lat1, lon1, lat2, lon2):
    """PURE_FUNCTION: math computation."""
    R = 6371
    dlat = math.radians(lat2 - lat1)
    dlon = math.radians(lon2 - lon1)
    a = (math.sin(dlat / 2) ** 2
         + math.cos(math.radians(lat1))
         * math.cos(math.radians(lat2))
         * math.sin(dlon / 2) ** 2)
    return R * 2 * math.atan2(math.sqrt(a), math.sqrt(1 - a))


def compute_with_logging(x, y):
    """BORDERLINE: pure math but uses print."""
    result = x ** 2 + y ** 2
    print(f"Result: {result}")
    return result
