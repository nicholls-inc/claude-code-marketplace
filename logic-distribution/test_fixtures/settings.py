"""Test fixture: Settings file — should be CONFIGURATION."""
DEBUG = True
DATABASES = {"default": {"ENGINE": "django.db.backends.sqlite3"}}


def get_env_variable(var_name):
    """CONFIGURATION: function in settings file."""
    import os
    return os.environ.get(var_name)
