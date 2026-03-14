"""Test fixture: Django views and DRF."""
from django.views import View
from django.http import JsonResponse
from django.views.decorators.http import require_http_methods
from rest_framework.views import APIView
from rest_framework.serializers import ModelSerializer
from rest_framework.permissions import BasePermission


class ProductView(View):
    def get(self, request, pk):
        """VIEW_MIDDLEWARE: class-based view method."""
        product = self.get_object(pk)
        return JsonResponse({"name": product.name})


class ProductAPIView(APIView):
    def post(self, request):
        """VIEW_MIDDLEWARE: DRF APIView method."""
        serializer = ProductSerializer(data=request.data)
        serializer.is_valid(raise_exception=True)
        serializer.save()
        return Response(serializer.data, status=201)


class ProductSerializer(ModelSerializer):
    class Meta:
        model = None
        fields = '__all__'

    def validate_price(self, value):
        """VIEW_MIDDLEWARE: serializer validation."""
        if value < 0:
            raise ValueError("negative")
        return value


class IsProductOwner(BasePermission):
    def has_object_permission(self, request, view, obj):
        """VIEW_MIDDLEWARE: permission class."""
        return obj.owner == request.user


@require_http_methods(["GET", "POST"])
def product_list(request):
    """VIEW_MIDDLEWARE: decorated function view."""
    return JsonResponse({"products": []})
