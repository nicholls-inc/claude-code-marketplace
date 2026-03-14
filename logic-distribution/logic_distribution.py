#!/usr/bin/env python3
"""
Codebase Logic Distribution Analyzer

Static analysis tool that classifies every function/method in a Django codebase
by where its logic lives, to answer: "what percentage of code is reachable by
formal verification tools like Dafny or Lean?"

Usage:
    python logic_distribution.py /path/to/django/project [--app APP_NAME]
        [--verbose] [--spot-check N] [--output analysis_results.json]
"""

import argparse
import ast
import json
import os
import random
import re
import sys
import textwrap
from collections import defaultdict
from dataclasses import asdict, dataclass, field
from enum import Enum
from pathlib import Path
from typing import Optional


class Category(str, Enum):
    DATABASE_ORM = "DATABASE_ORM"
    MODEL_VALIDATION = "MODEL_VALIDATION"
    VIEW_MIDDLEWARE = "VIEW_MIDDLEWARE"
    PURE_FUNCTION = "PURE_FUNCTION"
    EXTERNAL_IO = "EXTERNAL_IO"
    TEST_CODE = "TEST_CODE"
    CONFIGURATION = "CONFIGURATION"


class Confidence(str, Enum):
    HIGH = "HIGH"
    MEDIUM = "MEDIUM"
    LOW = "LOW"


@dataclass
class FunctionInfo:
    file_path: str
    function_name: str
    line_number: int
    line_count: int
    category: Category
    confidence: Confidence
    rationale: str
    matched_categories: list = field(default_factory=list)
    is_borderline: bool = False
    borderline_reason: str = ""
    class_name: Optional[str] = None
    decorators: list = field(default_factory=list)


# ---------------------------------------------------------------------------
# Detection constants
# ---------------------------------------------------------------------------

ORM_ATTRIBUTES = {
    "objects", "filter", "exclude", "get", "aggregate", "annotate",
    "values", "values_list", "select_related", "prefetch_related",
    "bulk_create", "bulk_update", "update", "delete", "raw", "extra",
    "using", "order_by", "distinct", "count", "first", "last", "exists",
    "create", "get_or_create", "update_or_create", "in_bulk",
    "iterator", "only", "defer",
}

ORM_NAMES = {
    "Q", "F", "Value", "Subquery", "OuterRef", "Exists",
    "Count", "Sum", "Avg", "Max", "Min", "RawSQL",
    "Case", "When", "Coalesce", "Greatest", "Least",
    "Prefetch", "FilteredRelation",
}

ORM_TRANSACTION = {"atomic", "on_commit", "savepoint", "set_autocommit"}

SQL_KEYWORDS_PATTERN = re.compile(
    r"\b(SELECT\s+.+?\s+FROM|INSERT\s+INTO|UPDATE\s+.+?\s+SET|DELETE\s+FROM)\b",
    re.IGNORECASE,
)

EXTERNAL_IO_MODULES = {
    "requests", "httpx", "aiohttp", "urllib.request", "urllib3",
    "subprocess", "smtplib", "socket", "paramiko", "fabric",
    "boto3", "botocore", "google.cloud",
}

EXTERNAL_IO_CALLS = {
    "open", "urlopen", "os.system", "os.popen",
    "send_mail", "send_mass_mail", "mail_admins", "mail_managers",
}

EXTERNAL_IO_ATTRS = {
    "delay", "apply_async", "send",  # celery
    "get", "post", "put", "patch", "delete", "head", "options",  # HTTP
}

CELERY_DECORATORS = {"shared_task", "task"}

CACHE_ATTRS = {"cache.set", "cache.get", "cache.delete", "cache.clear"}

VIEW_BASE_CLASSES = {
    "View", "TemplateView", "ListView", "DetailView", "CreateView",
    "UpdateView", "DeleteView", "FormView", "RedirectView", "ArchiveIndexView",
    "APIView", "ViewSet", "ModelViewSet", "GenericViewSet",
    "GenericAPIView", "ReadOnlyModelViewSet", "ViewSetMixin",
    "ListAPIView", "RetrieveAPIView", "CreateAPIView", "DestroyAPIView",
    "UpdateAPIView", "ListCreateAPIView", "RetrieveUpdateAPIView",
    "RetrieveDestroyAPIView", "RetrieveUpdateDestroyAPIView",
    # GraphQL / Graphene / Strawberry / Ariadne
    "Mutation", "BaseMutation", "ModelMutation", "ClientIDMutation",
    "ObjectType", "InputObjectType", "DjangoObjectType",
    "DjangoMutation", "BaseBulkMutation", "ModelBulkDeleteMutation",
    "BaseReorderDiscount",
}

SERIALIZER_BASE_CLASSES = {
    "Serializer", "ModelSerializer", "ListSerializer",
    "HyperlinkedModelSerializer", "BaseSerializer",
}

PERMISSION_AUTH_CLASSES = {
    "BasePermission", "IsAuthenticated", "IsAdminUser",
    "IsAuthenticatedOrReadOnly", "DjangoModelPermissions",
    "BaseAuthentication", "BasicAuthentication",
    "SessionAuthentication", "TokenAuthentication",
}

FORM_BASE_CLASSES = {"Form", "ModelForm", "BaseForm", "BaseModelForm"}

MIDDLEWARE_METHODS = {
    "__call__", "process_request", "process_response",
    "process_view", "process_exception", "process_template_response",
}

VIEW_DECORATORS = {
    "api_view", "require_http_methods", "require_GET", "require_POST",
    "require_safe", "login_required", "permission_required",
    "user_passes_test", "csrf_exempt", "csrf_protect",
    "never_cache", "cache_page", "cache_control",
}

TEMPLATE_TAG_DECORATORS = {
    "register.filter", "register.simple_tag", "register.inclusion_tag",
    "register.tag", "register.assignment_tag",
}

MODEL_BASE_CLASSES = {"Model", "AbstractUser", "AbstractBaseUser"}
MANAGER_BASE_CLASSES = {"Manager", "BaseManager", "QuerySet"}

MODEL_SPECIAL_METHODS = {
    "clean", "full_clean", "save", "delete", "get_absolute_url",
    "get_FOO_display", "__str__", "__repr__",
}

SIGNAL_NAMES = {
    "pre_save", "post_save", "pre_delete", "post_delete",
    "m2m_changed", "pre_init", "post_init", "pre_migrate", "post_migrate",
    "request_started", "request_finished", "got_request_exception",
}

DJANGO_MODULES = {
    "django", "rest_framework", "drf_spectacular", "drf_yasg",
    "graphene", "graphene_django", "strawberry", "ariadne",
}

# GraphQL resolver/mutation method names
GRAPHQL_METHODS = {
    "resolve", "mutate", "perform_mutation", "get_queryset",
    "get_node", "get_type", "clean_input", "clean_instance",
    "post_save_action", "pre_save_action", "get_instance",
    "save", "perform_create", "perform_update", "perform_destroy",
}

# Framework-layer file path patterns (these files are framework glue, not pure logic)
FRAMEWORK_FILE_PATTERNS = {
    "mutations", "resolvers", "schema", "types", "enums",
    "filters", "sorters", "dataloaders", "permissions",
    "serializers", "validators", "managers", "signals",
    "webhooks", "tasks", "commands",
}

SIDE_EFFECT_CALLS = {"print"}
LOGGING_ATTRS = {"debug", "info", "warning", "error", "critical", "exception", "log"}

PURE_STDLIB_MODULES = {
    "math", "itertools", "functools", "collections", "dataclasses",
    "typing", "enum", "decimal", "datetime", "string", "re", "operator",
    "abc", "copy", "hashlib", "hmac", "json", "base64", "binascii",
    "struct", "textwrap", "unicodedata", "fractions", "statistics",
    "bisect", "heapq", "array", "contextlib", "inspect", "types",
    "numbers", "uuid",
}

EXCLUDED_DIRS = {
    "venv", ".venv", "env", ".env", "node_modules", ".tox",
    "__pycache__", ".git", ".hg", ".svn", "site-packages",
    ".mypy_cache", ".pytest_cache", ".eggs", "dist", "build",
}


# ---------------------------------------------------------------------------
# AST Helpers
# ---------------------------------------------------------------------------

def get_attribute_chain(node: ast.AST) -> list[str]:
    """Extract the full attribute chain from a node, e.g. foo.bar.baz -> ['foo','bar','baz']."""
    parts = []
    while isinstance(node, ast.Attribute):
        parts.append(node.attr)
        node = node.value
    if isinstance(node, ast.Name):
        parts.append(node.id)
    parts.reverse()
    return parts


def get_decorator_names(decorators: list[ast.expr]) -> list[str]:
    """Extract decorator names as strings."""
    names = []
    for dec in decorators:
        if isinstance(dec, ast.Name):
            names.append(dec.id)
        elif isinstance(dec, ast.Attribute):
            chain = get_attribute_chain(dec)
            names.append(".".join(chain))
        elif isinstance(dec, ast.Call):
            if isinstance(dec.func, ast.Name):
                names.append(dec.func.id)
            elif isinstance(dec.func, ast.Attribute):
                chain = get_attribute_chain(dec.func)
                names.append(".".join(chain))
    return names


def get_base_class_names(class_node: ast.ClassDef) -> list[str]:
    """Extract base class names from a class definition."""
    names = []
    for base in class_node.bases:
        if isinstance(base, ast.Name):
            names.append(base.id)
        elif isinstance(base, ast.Attribute):
            chain = get_attribute_chain(base)
            names.append(chain[-1])  # Just the final name
            names.append(".".join(chain))
    return names


def get_string_literals(node: ast.AST) -> list[str]:
    """Extract all string literals from a node tree."""
    strings = []
    for child in ast.walk(node):
        if isinstance(child, ast.Constant) and isinstance(child.value, str):
            strings.append(child.value)
    return strings


def get_all_calls(node: ast.AST) -> list[tuple[list[str], ast.AST]]:
    """Get all function/method calls as (attribute_chain, call_node) pairs."""
    calls = []
    for child in ast.walk(node):
        if isinstance(child, ast.Call):
            if isinstance(child.func, ast.Name):
                calls.append(([child.func.id], child))
            elif isinstance(child.func, ast.Attribute):
                chain = get_attribute_chain(child.func)
                calls.append((chain, child))
    return calls


def get_all_attribute_accesses(node: ast.AST) -> list[list[str]]:
    """Get all attribute accesses as chains."""
    accesses = []
    for child in ast.walk(node):
        if isinstance(child, ast.Attribute):
            chain = get_attribute_chain(child)
            if chain:
                accesses.append(chain)
    return accesses


def get_all_names(node: ast.AST) -> set[str]:
    """Get all Name references in a node tree."""
    names = set()
    for child in ast.walk(node):
        if isinstance(child, ast.Name):
            names.add(child.id)
    return names


def function_line_count(func_node: ast.AST) -> int:
    """Calculate the number of lines a function spans."""
    if hasattr(func_node, "end_lineno") and func_node.end_lineno is not None:
        return func_node.end_lineno - func_node.lineno + 1
    # Fallback: walk all children and find max line number
    max_line = func_node.lineno
    for child in ast.walk(func_node):
        if hasattr(child, "lineno") and child.lineno is not None:
            max_line = max(max_line, child.lineno)
        if hasattr(child, "end_lineno") and child.end_lineno is not None:
            max_line = max(max_line, child.end_lineno)
    return max_line - func_node.lineno + 1


# ---------------------------------------------------------------------------
# File-level analysis
# ---------------------------------------------------------------------------

@dataclass
class FileContext:
    file_path: str
    imports: set = field(default_factory=set)  # module names imported
    import_names: set = field(default_factory=set)  # names imported (from X import Y)
    django_imported_names: set = field(default_factory=set)  # names from Django/framework modules
    is_test_file: bool = False
    is_migration_file: bool = False
    is_settings_file: bool = False
    is_config_file: bool = False
    class_bases: dict = field(default_factory=dict)  # class_name -> [base_names]


def analyze_file_context(file_path: str, tree: ast.Module) -> FileContext:
    """Analyze file-level context: imports, file type, class hierarchy."""
    ctx = FileContext(file_path=file_path)

    rel = file_path.replace("\\", "/")
    basename = os.path.basename(rel)
    parts = rel.split("/")

    # Detect file type
    ctx.is_test_file = (
        basename.startswith("test_")
        or basename.endswith("_test.py")
        or basename == "tests.py"
        or "tests" in parts
        or basename.startswith("conftest")
        or "/test/" in rel
    )

    ctx.is_migration_file = "migrations" in parts and basename != "__init__.py"
    ctx.is_settings_file = basename in (
        "settings.py", "local_settings.py", "production_settings.py",
        "base_settings.py",
    ) or "/settings/" in rel

    ctx.is_config_file = basename in (
        "apps.py", "admin.py", "urls.py", "wsgi.py", "asgi.py",
        "celery.py", "conftest.py",
    )

    # Analyze imports
    for node in ast.walk(tree):
        if isinstance(node, ast.Import):
            for alias in node.names:
                ctx.imports.add(alias.name.split(".")[0])
                ctx.imports.add(alias.name)
        elif isinstance(node, ast.ImportFrom):
            if node.module:
                ctx.imports.add(node.module.split(".")[0])
                ctx.imports.add(node.module)
                # Track names imported from Django/framework modules
                is_django_module = any(
                    node.module.startswith(dm) for dm in DJANGO_MODULES
                )
                for alias in node.names:
                    name = alias.asname or alias.name
                    ctx.import_names.add(alias.name)
                    if alias.asname:
                        ctx.import_names.add(alias.asname)
                    if is_django_module:
                        ctx.django_imported_names.add(name)
            else:
                for alias in node.names:
                    ctx.import_names.add(alias.name)
                    if alias.asname:
                        ctx.import_names.add(alias.asname)

    # Analyze class hierarchy
    for node in ast.walk(tree):
        if isinstance(node, ast.ClassDef):
            ctx.class_bases[node.name] = get_base_class_names(node)

    return ctx


# ---------------------------------------------------------------------------
# Function classifier
# ---------------------------------------------------------------------------

def classify_function(
    func_node: ast.FunctionDef | ast.AsyncFunctionDef,
    ctx: FileContext,
    enclosing_class: Optional[ast.ClassDef],
    verbose: bool = False,
) -> FunctionInfo:
    """Classify a single function/method."""

    func_name = func_node.name
    class_name = enclosing_class.name if enclosing_class else None
    full_name = f"{class_name}.{func_name}" if class_name else func_name
    line_num = func_node.lineno
    lines = function_line_count(func_node)
    decorators = get_decorator_names(func_node.decorator_list)

    matched = []  # (Category, rationale) tuples
    reasons = []

    # ---- Priority 1: TEST_CODE ----
    if ctx.is_test_file:
        return FunctionInfo(
            file_path=ctx.file_path, function_name=full_name,
            line_number=line_num, line_count=lines,
            category=Category.TEST_CODE, confidence=Confidence.HIGH,
            rationale="File is a test file",
            matched_categories=[Category.TEST_CODE.value],
            class_name=class_name, decorators=decorators,
        )

    # ---- Priority 2: CONFIGURATION ----
    if ctx.is_migration_file or ctx.is_settings_file:
        return FunctionInfo(
            file_path=ctx.file_path, function_name=full_name,
            line_number=line_num, line_count=lines,
            category=Category.CONFIGURATION, confidence=Confidence.HIGH,
            rationale="File is migration or settings",
            matched_categories=[Category.CONFIGURATION.value],
            class_name=class_name, decorators=decorators,
        )

    # Gather function body analysis
    calls = get_all_calls(func_node)
    attr_accesses = get_all_attribute_accesses(func_node)
    names_used = get_all_names(func_node)
    string_literals = get_string_literals(func_node)

    call_chains = [c[0] for c in calls]
    flat_attrs = {".".join(chain) for chain in attr_accesses}
    flat_attr_parts = set()
    for chain in attr_accesses:
        for part in chain:
            flat_attr_parts.add(part)

    # ---- Check DATABASE_ORM ----
    orm_reasons = []

    # ORM attribute accesses
    orm_attrs_found = ORM_ATTRIBUTES & flat_attr_parts
    if orm_attrs_found:
        # Check for actual ORM-like chains (not just any use of .get() etc.)
        for chain in attr_accesses:
            chain_str = ".".join(chain)
            for attr in orm_attrs_found:
                if attr in chain and (
                    "objects" in chain
                    or attr in {"filter", "exclude", "aggregate", "annotate",
                               "values", "values_list", "select_related",
                               "prefetch_related", "bulk_create", "bulk_update",
                               "raw", "extra"}
                    or any(c in chain for c in ["objects", "qs", "queryset"])
                ):
                    orm_reasons.append(f"ORM attribute: {chain_str}")
                    break
        # Check for .objects. specifically
        for chain in attr_accesses:
            if "objects" in chain:
                orm_reasons.append(f"QuerySet via .objects: {'.'.join(chain)}")

    # ORM names (Q, F, etc.)
    orm_names_found = ORM_NAMES & names_used
    if orm_names_found:
        # Verify they're actually imported from Django
        if orm_names_found & ctx.import_names or "django" in str(ctx.imports):
            orm_reasons.append(f"ORM expression names: {orm_names_found}")

    # cursor.execute
    if "cursor.execute" in flat_attrs or "execute" in flat_attr_parts:
        for chain in attr_accesses:
            if "cursor" in chain and "execute" in chain:
                orm_reasons.append("Raw SQL via cursor.execute")

    # SQL in string literals
    for s in string_literals:
        if SQL_KEYWORDS_PATTERN.search(s):
            orm_reasons.append(f"SQL keyword in string literal")
            break

    # transaction
    for chain in attr_accesses:
        if "transaction" in chain:
            txn_attr = set(chain) & ORM_TRANSACTION
            if txn_attr:
                orm_reasons.append(f"Transaction usage: {txn_attr}")

    if orm_reasons:
        matched.append((Category.DATABASE_ORM, "; ".join(orm_reasons[:3])))

    # ---- Check EXTERNAL_IO ----
    io_reasons = []

    # External HTTP modules
    for mod in EXTERNAL_IO_MODULES:
        if mod in ctx.imports or mod.split(".")[0] in ctx.imports:
            for chain in call_chains:
                chain_str = ".".join(chain)
                if mod.split(".")[0] in chain_str:
                    io_reasons.append(f"External call via {mod}: {chain_str}")

    # Direct IO calls
    for chain in call_chains:
        if chain and chain[0] in EXTERNAL_IO_CALLS:
            # open() is IO
            if chain[0] == "open":
                io_reasons.append("File IO via open()")
            else:
                io_reasons.append(f"IO call: {chain[0]}")

    # subprocess
    for chain in call_chains:
        chain_str = ".".join(chain)
        if "subprocess" in chain_str or "os.system" in chain_str or "os.popen" in chain_str:
            io_reasons.append(f"Subprocess: {chain_str}")

    # Celery decorators
    for dec in decorators:
        if any(cd in dec for cd in CELERY_DECORATORS):
            io_reasons.append(f"Celery task decorator: {dec}")

    # Celery calls (.delay(), .apply_async())
    for chain in attr_accesses:
        if chain and chain[-1] in ("delay", "apply_async"):
            io_reasons.append(f"Celery call: {'.'.join(chain)}")

    # Cache operations
    for chain_str in flat_attrs:
        if any(chain_str.endswith(ca.split(".")[-1]) and "cache" in chain_str
               for ca in CACHE_ATTRS):
            io_reasons.append(f"Cache operation: {chain_str}")

    # Email
    for chain in call_chains:
        chain_str = ".".join(chain)
        if any(em in chain_str for em in ("send_mail", "EmailMessage", "mail_admins")):
            io_reasons.append(f"Email: {chain_str}")

    # pathlib writes
    for chain in attr_accesses:
        chain_str = ".".join(chain)
        if "Path" in chain and any(w in chain_str for w in ("write_text", "write_bytes", "read_text", "read_bytes", "mkdir", "unlink")):
            io_reasons.append(f"Pathlib IO: {chain_str}")

    if io_reasons:
        matched.append((Category.EXTERNAL_IO, "; ".join(io_reasons[:3])))

    # ---- Check VIEW_MIDDLEWARE ----
    view_reasons = []

    if enclosing_class:
        bases = set(ctx.class_bases.get(enclosing_class.name, []))

        # View classes
        if bases & VIEW_BASE_CLASSES:
            view_reasons.append(f"Method on View subclass: {enclosing_class.name}")

        # Serializer classes
        if bases & SERIALIZER_BASE_CLASSES:
            view_reasons.append(f"Method on Serializer: {enclosing_class.name}")

        # Permission/Auth classes
        if bases & PERMISSION_AUTH_CLASSES:
            view_reasons.append(f"Method on Permission/Auth class: {enclosing_class.name}")

        # Form classes
        if bases & FORM_BASE_CLASSES:
            view_reasons.append(f"Method on Form: {enclosing_class.name}")

        # Middleware detection
        if func_name in MIDDLEWARE_METHODS:
            # Check if any base looks like middleware or if 'get_response' is in args
            arg_names = [a.arg for a in func_node.args.args if hasattr(a, 'arg')]
            if "get_response" in arg_names or func_name in ("process_request", "process_response", "process_view"):
                view_reasons.append(f"Middleware method: {func_name}")

    # View decorators
    view_decs = set(decorators) & VIEW_DECORATORS
    if view_decs:
        view_reasons.append(f"View decorator: {view_decs}")

    # Template tag decorators
    for dec in decorators:
        if any(tt in dec for tt in TEMPLATE_TAG_DECORATORS):
            view_reasons.append(f"Template tag: {dec}")

    # admin.py methods (if not caught by config)
    if ctx.is_config_file and os.path.basename(ctx.file_path) == "admin.py":
        view_reasons.append("Admin configuration method")

    # GraphQL resolver/mutation methods (by name pattern)
    if enclosing_class:
        bases = set(ctx.class_bases.get(enclosing_class.name, []))
        if bases & VIEW_BASE_CLASSES:
            # Already caught above
            pass
        elif func_name.startswith("resolve_") or func_name in GRAPHQL_METHODS:
            if any(m.startswith(dm) for m in ctx.imports for dm in ("graphene", "strawberry", "ariadne")):
                view_reasons.append(f"GraphQL method: {func_name} on {enclosing_class.name}")

    # Functions in framework-layer files (mutations/, resolvers/, etc.)
    file_basename = os.path.basename(ctx.file_path).replace(".py", "")
    file_parts = set(Path(ctx.file_path).parts)
    is_framework_file = (
        file_basename in FRAMEWORK_FILE_PATTERNS
        or bool(file_parts & FRAMEWORK_FILE_PATTERNS)
    )

    if is_framework_file and not view_reasons and not matched:
        # Functions in framework-layer files that use Django imports
        if any(m.startswith(dm) for m in ctx.imports for dm in DJANGO_MODULES):
            view_reasons.append(f"Function in framework file: {file_basename}.py")

    if view_reasons:
        matched.append((Category.VIEW_MIDDLEWARE, "; ".join(view_reasons[:3])))

    # ---- Check MODEL_VALIDATION ----
    model_reasons = []

    if enclosing_class:
        bases = set(ctx.class_bases.get(enclosing_class.name, []))

        is_model_class = bool(bases & MODEL_BASE_CLASSES)
        is_manager_class = bool(bases & MANAGER_BASE_CLASSES)

        if is_model_class or is_manager_class:
            if func_name in MODEL_SPECIAL_METHODS or func_name.startswith("validate_"):
                model_reasons.append(f"Model special method: {func_name}")
            elif func_name.startswith("get_") and func_name.endswith("_display"):
                model_reasons.append(f"Model display method: {func_name}")
            elif "property" in decorators:
                model_reasons.append(f"Model property: {func_name}")
            elif is_model_class:
                model_reasons.append(f"Custom model method: {func_name} on {enclosing_class.name}")
            elif is_manager_class:
                model_reasons.append(f"Manager/QuerySet method: {func_name}")

    # Signal handlers
    if "receiver" in decorators:
        model_reasons.append(f"Signal receiver: {func_name}")
    for chain in call_chains:
        chain_str = ".".join(chain)
        if "connect" in chain_str and any(s in chain_str for s in SIGNAL_NAMES):
            model_reasons.append(f"Signal connection: {chain_str}")

    if model_reasons:
        matched.append((Category.MODEL_VALIDATION, "; ".join(model_reasons[:3])))

    # ---- Check PURE_FUNCTION (by absence) ----
    pure_disqualifiers = []

    # Django/framework imports at file level
    django_imports = {m for m in ctx.imports if any(
        m.startswith(dm) for dm in DJANGO_MODULES
    )}

    # Check if function body references django-imported names
    uses_django_names = bool(ctx.django_imported_names & names_used)

    # Check for side effects
    has_logging = False
    has_print = False
    for chain in call_chains:
        chain_str = ".".join(chain)
        if chain[-1] in LOGGING_ATTRS and any(
            "log" in c.lower() or "logger" in c.lower() for c in chain
        ):
            has_logging = True
        if chain[0] == "print":
            has_print = True
        if chain[0] in ("global", ):
            pure_disqualifiers.append("Global state mutation")

    # Check for global/nonlocal
    for child in ast.walk(func_node):
        if isinstance(child, ast.Global):
            pure_disqualifiers.append("global statement")
        elif isinstance(child, ast.Nonlocal):
            pure_disqualifiers.append("nonlocal statement")

    # ---- Apply priority rules ----
    # Config file functions that didn't match view/middleware
    if ctx.is_config_file and not matched:
        return FunctionInfo(
            file_path=ctx.file_path, function_name=full_name,
            line_number=line_num, line_count=lines,
            category=Category.CONFIGURATION, confidence=Confidence.MEDIUM,
            rationale="Function in configuration file",
            matched_categories=[Category.CONFIGURATION.value],
            class_name=class_name, decorators=decorators,
        )

    all_matched_cats = [m[0].value for m in matched]

    if not matched:
        # If function uses Django-imported names, classify as VIEW_MIDDLEWARE
        # (framework glue) rather than pure
        if uses_django_names:
            return FunctionInfo(
                file_path=ctx.file_path, function_name=full_name,
                line_number=line_num, line_count=lines,
                category=Category.VIEW_MIDDLEWARE, confidence=Confidence.LOW,
                rationale=f"Uses Django-imported names: {ctx.django_imported_names & names_used}",
                matched_categories=[Category.VIEW_MIDDLEWARE.value],
                is_borderline=True,
                borderline_reason="Classified as framework code due to Django name usage",
                class_name=class_name, decorators=decorators,
            )

        # No specific category matched -> PURE_FUNCTION by default
        if pure_disqualifiers:
            confidence = Confidence.MEDIUM
            rationale = f"Default to pure, but has: {'; '.join(pure_disqualifiers)}"
        elif has_logging or has_print:
            confidence = Confidence.MEDIUM
            rationale = "Pure function (has logging/print side effects)"
        else:
            confidence = Confidence.LOW if django_imports else Confidence.HIGH
            rationale = "No Django/IO/ORM indicators detected"

        borderline = bool(has_logging or has_print or pure_disqualifiers)
        borderline_reason = ""
        if has_logging:
            borderline_reason = "Uses logging"
        elif has_print:
            borderline_reason = "Uses print()"
        elif pure_disqualifiers:
            borderline_reason = "; ".join(pure_disqualifiers)

        return FunctionInfo(
            file_path=ctx.file_path, function_name=full_name,
            line_number=line_num, line_count=lines,
            category=Category.PURE_FUNCTION, confidence=confidence,
            rationale=rationale,
            matched_categories=all_matched_cats or [Category.PURE_FUNCTION.value],
            is_borderline=borderline, borderline_reason=borderline_reason,
            class_name=class_name, decorators=decorators,
        )

    # Priority: DATABASE_ORM > EXTERNAL_IO > VIEW_MIDDLEWARE > MODEL_VALIDATION
    priority_order = [
        Category.DATABASE_ORM,
        Category.EXTERNAL_IO,
        Category.VIEW_MIDDLEWARE,
        Category.MODEL_VALIDATION,
    ]

    chosen_cat = None
    chosen_rationale = ""
    for cat in priority_order:
        for m_cat, m_rationale in matched:
            if m_cat == cat:
                chosen_cat = cat
                chosen_rationale = m_rationale
                break
        if chosen_cat:
            break

    if not chosen_cat:
        chosen_cat = matched[0][0]
        chosen_rationale = matched[0][1]

    confidence = Confidence.HIGH if len(matched) == 1 else Confidence.MEDIUM
    is_borderline = len(matched) > 1
    borderline_reason = ""
    if is_borderline:
        other_cats = [m[0].value for m in matched if m[0] != chosen_cat]
        borderline_reason = f"Also matched: {', '.join(other_cats)}"

    return FunctionInfo(
        file_path=ctx.file_path, function_name=full_name,
        line_number=line_num, line_count=lines,
        category=chosen_cat, confidence=confidence,
        rationale=chosen_rationale,
        matched_categories=all_matched_cats,
        is_borderline=is_borderline, borderline_reason=borderline_reason,
        class_name=class_name, decorators=decorators,
    )


# ---------------------------------------------------------------------------
# File + project analysis
# ---------------------------------------------------------------------------

def analyze_file(file_path: str, verbose: bool = False) -> list[FunctionInfo]:
    """Analyze all functions in a single Python file."""
    try:
        with open(file_path, "r", encoding="utf-8", errors="replace") as f:
            source = f.read()
    except (OSError, IOError) as e:
        if verbose:
            print(f"  [SKIP] Cannot read {file_path}: {e}", file=sys.stderr)
        return []

    try:
        tree = ast.parse(source, filename=file_path)
    except SyntaxError as e:
        if verbose:
            print(f"  [SKIP] Syntax error in {file_path}: {e}", file=sys.stderr)
        return []

    ctx = analyze_file_context(file_path, tree)
    results = []
    for node in tree.body:
        if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
            info = classify_function(node, ctx, enclosing_class=None, verbose=verbose)
            results.append(info)
            if verbose:
                print(f"  {info.category.value:20s} {info.function_name} "
                      f"(L{info.line_number}, {info.line_count} lines) "
                      f"[{info.confidence.value}] {info.rationale}")
        elif isinstance(node, ast.ClassDef):
            for item in node.body:
                if isinstance(item, (ast.FunctionDef, ast.AsyncFunctionDef)):
                    info = classify_function(item, ctx, enclosing_class=node, verbose=verbose)
                    results.append(info)
                    if verbose:
                        print(f"  {info.category.value:20s} {info.function_name} "
                              f"(L{info.line_number}, {info.line_count} lines) "
                              f"[{info.confidence.value}] {info.rationale}")

    return results


def find_python_files(project_root: str, app_name: Optional[str] = None) -> list[str]:
    """Find all Python files in the project, excluding standard non-source dirs."""
    python_files = []
    root = Path(project_root).resolve()

    for dirpath, dirnames, filenames in os.walk(root):
        # Prune excluded directories
        dirnames[:] = [
            d for d in dirnames
            if d not in EXCLUDED_DIRS and not d.startswith(".")
        ]

        rel_dir = os.path.relpath(dirpath, root)

        # If app_name specified, only include files in that app
        if app_name:
            parts = Path(rel_dir).parts
            if parts and parts[0] != app_name and rel_dir != ".":
                continue

        for fname in filenames:
            if fname.endswith(".py"):
                python_files.append(os.path.join(dirpath, fname))

    return sorted(python_files)


def analyze_project(
    project_root: str,
    app_name: Optional[str] = None,
    verbose: bool = False,
) -> list[FunctionInfo]:
    """Analyze an entire Django project."""
    files = find_python_files(project_root, app_name)
    if verbose:
        print(f"Found {len(files)} Python files to analyze", file=sys.stderr)

    all_results = []
    for f in files:
        if verbose:
            rel = os.path.relpath(f, project_root)
            print(f"\nAnalyzing: {rel}", file=sys.stderr)
        results = analyze_file(f, verbose=verbose)
        all_results.extend(results)

    return all_results


# ---------------------------------------------------------------------------
# Reporting
# ---------------------------------------------------------------------------

MAIN_CATEGORIES = [
    Category.DATABASE_ORM,
    Category.MODEL_VALIDATION,
    Category.VIEW_MIDDLEWARE,
    Category.PURE_FUNCTION,
    Category.EXTERNAL_IO,
]


def print_summary(results: list[FunctionInfo], project_root: str):
    """Print the summary table."""
    main = [r for r in results if r.category in MAIN_CATEGORIES]
    tests = [r for r in results if r.category == Category.TEST_CODE]
    config = [r for r in results if r.category == Category.CONFIGURATION]

    cat_funcs = defaultdict(int)
    cat_lines = defaultdict(int)
    for r in main:
        cat_funcs[r.category] += 1
        cat_lines[r.category] += r.line_count

    total_funcs = sum(cat_funcs.values())
    total_lines = sum(cat_lines.values())

    print("\n" + "=" * 80)
    print("LOGIC DISTRIBUTION ANALYSIS")
    print("=" * 80)
    print(f"Project: {project_root}")
    print(f"Total functions analyzed: {len(results)}")
    print()

    header = f"{'Category':<22s} {'Functions':>10s} {'Lines of Code':>15s} {'% Functions':>12s} {'% Lines':>10s}"
    print(header)
    print("-" * len(header))

    for cat in MAIN_CATEGORIES:
        funcs = cat_funcs.get(cat, 0)
        lines = cat_lines.get(cat, 0)
        pct_funcs = (funcs / total_funcs * 100) if total_funcs else 0
        pct_lines = (lines / total_lines * 100) if total_lines else 0
        print(f"{cat.value:<22s} {funcs:>10,d} {lines:>15,d} {pct_funcs:>11.1f}% {pct_lines:>9.1f}%")

    print("-" * len(header))
    print(f"{'TOTAL':<22s} {total_funcs:>10,d} {total_lines:>15,d} {'100.0%':>12s} {'100.0%':>10s}")
    print()
    print(f"Excluded: {len(tests)} test functions, {len(config)} configuration functions")


def print_pure_functions(results: list[FunctionInfo], project_root: str):
    """Print detail for pure functions."""
    pure = [r for r in results if r.category == Category.PURE_FUNCTION]
    if not pure:
        print("\nNo pure functions found.")
        return

    print("\n" + "=" * 80)
    print(f"PURE FUNCTIONS (formally verifiable) — {len(pure)} functions")
    print("=" * 80)

    pure.sort(key=lambda r: (r.file_path, r.line_number))
    for r in pure:
        rel = os.path.relpath(r.file_path, project_root)
        conf = f"[{r.confidence.value}]"
        borderline = " *BORDERLINE*" if r.is_borderline else ""
        print(f"  {rel}:{r.line_number}  {r.function_name} "
              f"({r.line_count} lines) {conf}{borderline}")
        if r.borderline_reason:
            print(f"    Note: {r.borderline_reason}")


def print_borderline_cases(results: list[FunctionInfo], project_root: str):
    """Print borderline classification cases."""
    borderline = [r for r in results if r.is_borderline and r.category in MAIN_CATEGORIES]
    if not borderline:
        print("\nNo borderline cases found.")
        return

    print("\n" + "=" * 80)
    print(f"BORDERLINE CASES — {len(borderline)} functions")
    print("=" * 80)

    borderline.sort(key=lambda r: (r.file_path, r.line_number))
    for r in borderline:
        rel = os.path.relpath(r.file_path, project_root)
        print(f"  {rel}:{r.line_number}  {r.function_name} → {r.category.value}")
        print(f"    Reason: {r.borderline_reason}")
        print(f"    Rationale: {r.rationale}")


def print_per_file_breakdown(results: list[FunctionInfo], project_root: str):
    """Print per-file category distribution."""
    print("\n" + "=" * 80)
    print("PER-FILE BREAKDOWN")
    print("=" * 80)

    by_file = defaultdict(list)
    for r in results:
        by_file[r.file_path].append(r)

    for fpath in sorted(by_file.keys()):
        funcs = by_file[fpath]
        rel = os.path.relpath(fpath, project_root)
        cat_counts = defaultdict(int)
        for f in funcs:
            cat_counts[f.category] += 1

        parts = " | ".join(
            f"{cat.value}: {cat_counts[cat]}"
            for cat in list(Category)
            if cat_counts[cat] > 0
        )
        print(f"  {rel} ({len(funcs)} funcs): {parts}")


def print_spot_check(results: list[FunctionInfo], n: int, project_root: str):
    """Print a random sample of N functions for manual review."""
    main = [r for r in results if r.category in MAIN_CATEGORIES]
    sample = random.sample(main, min(n, len(main)))

    print("\n" + "=" * 80)
    print(f"SPOT CHECK — {len(sample)} randomly sampled functions")
    print("=" * 80)

    for r in sample:
        rel = os.path.relpath(r.file_path, project_root)
        print(f"\n  {rel}:{r.line_number}")
        print(f"  Function: {r.function_name}")
        print(f"  Category: {r.category.value} [{r.confidence.value}]")
        print(f"  Lines: {r.line_count}")
        print(f"  Rationale: {r.rationale}")
        if r.is_borderline:
            print(f"  Borderline: {r.borderline_reason}")
        if r.decorators:
            print(f"  Decorators: {r.decorators}")


def write_json_output(results: list[FunctionInfo], output_path: str, project_root: str):
    """Write machine-readable JSON output."""
    data = {
        "project_root": project_root,
        "total_functions": len(results),
        "functions": [],
        "summary": {},
    }

    # Summary
    main = [r for r in results if r.category in MAIN_CATEGORIES]
    cat_funcs = defaultdict(int)
    cat_lines = defaultdict(int)
    for r in main:
        cat_funcs[r.category.value] += 1
        cat_lines[r.category.value] += r.line_count

    total_funcs = sum(cat_funcs.values())
    total_lines = sum(cat_lines.values())

    data["summary"] = {
        "main_functions": total_funcs,
        "main_lines": total_lines,
        "test_functions": sum(1 for r in results if r.category == Category.TEST_CODE),
        "config_functions": sum(1 for r in results if r.category == Category.CONFIGURATION),
        "categories": {
            cat: {
                "functions": cat_funcs.get(cat, 0),
                "lines": cat_lines.get(cat, 0),
                "pct_functions": round(cat_funcs.get(cat, 0) / total_funcs * 100, 1) if total_funcs else 0,
                "pct_lines": round(cat_lines.get(cat, 0) / total_lines * 100, 1) if total_lines else 0,
            }
            for cat in [c.value for c in MAIN_CATEGORIES]
        },
    }

    # All functions
    for r in results:
        entry = {
            "file_path": os.path.relpath(r.file_path, project_root),
            "function_name": r.function_name,
            "line_number": r.line_number,
            "line_count": r.line_count,
            "category": r.category.value,
            "confidence": r.confidence.value,
            "rationale": r.rationale,
            "matched_categories": r.matched_categories,
            "is_borderline": r.is_borderline,
            "borderline_reason": r.borderline_reason,
            "class_name": r.class_name,
            "decorators": r.decorators,
        }
        data["functions"].append(entry)

    with open(output_path, "w", encoding="utf-8") as f:
        json.dump(data, f, indent=2, ensure_ascii=False)

    print(f"\nJSON output written to: {output_path}")


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(
        description="Analyze logic distribution in a Django codebase",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=textwrap.dedent("""\
            Examples:
              python logic_distribution.py /path/to/saleor
              python logic_distribution.py /path/to/zulip --app zerver --verbose
              python logic_distribution.py /path/to/project --spot-check 20
              python logic_distribution.py /path/to/project --output results.json
        """),
    )
    parser.add_argument("project_root", help="Path to the Django project root")
    parser.add_argument("--app", help="Limit analysis to a specific Django app")
    parser.add_argument("--verbose", action="store_true",
                        help="Print each classification decision")
    parser.add_argument("--spot-check", type=int, metavar="N",
                        help="Randomly sample N functions for manual review")
    parser.add_argument("--output", default="analysis_results.json",
                        help="Path for JSON output (default: analysis_results.json)")
    parser.add_argument("--no-json", action="store_true",
                        help="Skip JSON output")
    parser.add_argument("--summary-only", action="store_true",
                        help="Only print the summary table")

    args = parser.parse_args()

    project_root = os.path.abspath(args.project_root)
    if not os.path.isdir(project_root):
        print(f"Error: {project_root} is not a directory", file=sys.stderr)
        sys.exit(1)

    print(f"Analyzing: {project_root}", file=sys.stderr)
    if args.app:
        print(f"Limiting to app: {args.app}", file=sys.stderr)

    results = analyze_project(project_root, app_name=args.app, verbose=args.verbose)

    if not results:
        print("No functions found to analyze.", file=sys.stderr)
        sys.exit(0)

    # Reports
    print_summary(results, project_root)

    if not args.summary_only:
        print_pure_functions(results, project_root)
        print_borderline_cases(results, project_root)
        print_per_file_breakdown(results, project_root)

    if args.spot_check:
        print_spot_check(results, args.spot_check, project_root)

    if not args.no_json:
        write_json_output(results, args.output, project_root)


if __name__ == "__main__":
    main()
