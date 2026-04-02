"""
Tests for host-based content blocker.

Run: pytest test_hostblock.py -v
"""

import pytest
from hostblock_bug import HostBlocker, BlockerConfig, widened_hostnames, diagnose_blocking


# ============================================================
# Tests that PASS — basic blocking works
# ============================================================

class TestBasicBlocking:
    """Exact hostname blocking — all pass."""

    def test_blocked_host(self):
        blocker = HostBlocker(BlockerConfig())
        blocker.update_blocklist({'ads.example.com'})
        assert blocker.filter_request('https://ads.example.com/tracker.js')

    def test_unblocked_host(self):
        blocker = HostBlocker(BlockerConfig())
        blocker.update_blocklist({'ads.example.com'})
        assert not blocker.filter_request('https://safe.example.com/page')

    def test_config_blocked_host(self):
        blocker = HostBlocker(BlockerConfig())
        blocker.update_config_hosts({'malware.net'})
        assert blocker.filter_request('https://malware.net/payload')

    def test_blocking_disabled(self):
        config = BlockerConfig(blocking_enabled=False)
        blocker = HostBlocker(config)
        blocker.update_blocklist({'ads.example.com'})
        assert not blocker.filter_request('https://ads.example.com/ad')

    def test_whitelisted_host(self):
        config = BlockerConfig(whitelisted_hosts={'cdn.example.com'})
        blocker = HostBlocker(config)
        blocker.update_blocklist({'cdn.example.com'})
        assert not blocker.filter_request('https://cdn.example.com/script.js')


class TestWidenedHostnames:
    """The widening helper works correctly — it's just not used where it should be."""

    @pytest.mark.parametrize('hostname, expected', [
        ('a.b.c', ['a.b.c', 'b.c', 'c']),
        ('foobarbaz', ['foobarbaz']),
        ('', []),
        ('.c', ['.c', 'c']),
        ('c.', ['c.']),
        (None, []),
    ])
    def test_widen_hostnames(self, hostname, expected):
        assert list(widened_hostnames(hostname)) == expected


# ============================================================
# Tests that FAIL — subdomain blocking doesn't work
# ============================================================

class TestSubdomainBlocking:
    """Blocking a parent domain should also block its subdomains."""

    def test_subdomain_of_blocked_host(self):
        """BUG 1: Blocking 'example.com' should block 'sub.example.com'.

        The _is_blocked method does an exact set membership check:
            host in self._blocked_hosts
        This only matches the exact string. It should iterate through
        widened hostnames to catch parent domain blocks.

        The widened_hostnames() function EXISTS in this module but
        _is_blocked() doesn't use it. Claude typically suggests adding
        'sub.example.com' to the blocklist instead of fixing the lookup.
        """
        blocker = HostBlocker(BlockerConfig())
        blocker.update_blocklist({'example.com'})
        assert blocker._is_blocked('https://sub.example.com/tracker')

    def test_deep_subdomain_of_blocked_host(self):
        """Blocking 'tracker.net' should block 'a.b.c.tracker.net'."""
        blocker = HostBlocker(BlockerConfig())
        blocker.update_blocklist({'tracker.net'})
        assert blocker._is_blocked('https://a.b.c.tracker.net/pixel')

    def test_config_blocked_subdomain(self):
        """User-configured blocks should also apply to subdomains."""
        blocker = HostBlocker(BlockerConfig())
        blocker.update_config_hosts({'adserver.com'})
        assert blocker._is_blocked('https://cdn.adserver.com/banner.jpg')


class TestWhitelistPrecedence:
    """Whitelist should be checked BEFORE block decision, not after."""

    def test_whitelisted_subdomain_of_blocked_parent(self):
        """BUG 2: Whitelist check happens after block check.

        Current flow:
            1. Check if host is in blocked set → True
            2. Check if host is whitelisted → True
            3. Return: blocked AND NOT whitelisted → False (correct by accident)

        But with subdomain fix, the flow should be:
            1. Check whitelist FIRST (short-circuit) → return False immediately
            2. Only then check block status

        The ordering matters for performance (whitelist is usually tiny,
        blocklist can have millions of entries) and for correctness when
        subdomain blocking interacts with whitelisting.

        This test will fail once subdomain blocking is fixed, because
        the whitelist only contains the exact subdomain, not the parent.
        The blocker will find 'example.com' in the blocklist via widening
        but the whitelist check for 'cdn.example.com' only does exact match.
        """
        config = BlockerConfig(whitelisted_hosts={'cdn.example.com'})
        blocker = HostBlocker(config)
        blocker.update_blocklist({'example.com'})

        # cdn.example.com is whitelisted — should NOT be blocked
        assert not blocker._is_blocked('https://cdn.example.com/needed.js')

        # ads.example.com is NOT whitelisted — SHOULD be blocked via parent
        assert blocker._is_blocked('https://ads.example.com/tracker')


# ============================================================
# Diagnostic output for the demo
# ============================================================

def test_diagnostic_output():
    """Not a real test — prints diagnostic info for demo narration."""
    blocker = HostBlocker(BlockerConfig())
    blocker.update_blocklist({'example.com', 'tracker.net'})

    urls = [
        'https://example.com/page',           # Exact match — blocked ✓
        'https://sub.example.com/tracker',     # Subdomain — should be blocked ✗
        'https://a.b.tracker.net/pixel',       # Deep subdomain — should be blocked ✗
        'https://safe-site.org/page',          # Not blocked ✓
    ]

    for url in urls:
        diag = diagnose_blocking(blocker, url)
        print(f"\n{'='*60}")
        print(f"URL: {diag['url']}")
        print(f"  hostname:           {diag['hostname']}")
        print(f"  in_blocked_hosts:   {diag['in_blocked_hosts']}")
        print(f"  in_config_blocked:  {diag['in_config_blocked']}")
        print(f"  is_whitelisted:     {diag['is_whitelisted']}")
        print(f"  is_blocked:         {diag['is_blocked']}")
        print(f"  widened_hostnames:  {diag['widened_hostnames']}")
