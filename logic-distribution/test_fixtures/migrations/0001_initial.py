"""Test fixture: Migration file — should be CONFIGURATION."""
from django.db import migrations, models


class Migration(migrations.Migration):
    initial = True
    dependencies = []

    operations = [
        migrations.CreateModel(
            name='Product',
            fields=[
                ('id', models.AutoField(primary_key=True)),
                ('name', models.CharField(max_length=200)),
            ],
        ),
    ]
