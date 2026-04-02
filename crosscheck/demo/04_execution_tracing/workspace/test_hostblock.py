"""
Tests for host-based content blocker.

Run: pytest test_hostblock.py -v
"""

import pytest
from hostblock import BlockerConfig, HostBlocker, widened_hostnames


class TestBasicBlocking:
    def test_blocked_host(self):
        blocker = HostBlocker(BlockerConfig())
        blocker.update_blocklist({"ads.example.com"})
        assert blocker.filter_request("https://ads.example.com/tracker.js")

    def test_unblocked_host(self):
        blocker = HostBlocker(BlockerConfig())
        blocker.update_blocklist({"ads.example.com"})
        assert not blocker.filter_request("https://safe.example.com/page")

    def test_config_blocked_host(self):
        blocker = HostBlocker(BlockerConfig())
        blocker.update_config_hosts({"malware.net"})
        assert blocker.filter_request("https://malware.net/payload")

    def test_blocking_disabled(self):
        config = BlockerConfig(blocking_enabled=False)
        blocker = HostBlocker(config)
        blocker.update_blocklist({"ads.example.com"})
        assert not blocker.filter_request("https://ads.example.com/ad")

    def test_whitelisted_host(self):
        config = BlockerConfig(whitelisted_hosts={"cdn.example.com"})
        blocker = HostBlocker(config)
        blocker.update_blocklist({"cdn.example.com"})
        assert not blocker.filter_request("https://cdn.example.com/script.js")


class TestWidenedHostnames:
    @pytest.mark.parametrize(
        "hostname, expected",
        [
            ("a.b.c", ["a.b.c", "b.c", "c"]),
            ("foobarbaz", ["foobarbaz"]),
            ("", []),
            (".c", [".c", "c"]),
            ("c.", ["c."]),
            (None, []),
        ],
    )
    def test_widen_hostnames(self, hostname, expected):
        assert list(widened_hostnames(hostname)) == expected


class TestSubdomainBlocking:
    """Blocking a parent domain should also block its subdomains."""

    def test_subdomain_of_blocked_host(self):
        blocker = HostBlocker(BlockerConfig())
        blocker.update_blocklist({"example.com"})
        assert blocker._is_blocked("https://sub.example.com/tracker")

    def test_deep_subdomain_of_blocked_host(self):
        """Blocking 'tracker.net' should block 'a.b.c.tracker.net'."""
        blocker = HostBlocker(BlockerConfig())
        blocker.update_blocklist({"tracker.net"})
        assert blocker._is_blocked("https://a.b.c.tracker.net/pixel")

    def test_config_blocked_subdomain(self):
        """User-configured blocks should also apply to subdomains."""
        blocker = HostBlocker(BlockerConfig())
        blocker.update_config_hosts({"adserver.com"})
        assert blocker._is_blocked("https://cdn.adserver.com/banner.jpg")


class TestWhitelistPrecedence:
    """Whitelist should be checked BEFORE block decision, not after."""

    def test_whitelisted_subdomain_of_blocked_parent(self):
        config = BlockerConfig(whitelisted_hosts={"cdn.example.com"})
        blocker = HostBlocker(config)
        blocker.update_blocklist({"example.com"})

        # cdn.example.com is whitelisted — should NOT be blocked
        assert not blocker._is_blocked("https://cdn.example.com/needed.js")

        # ads.example.com is NOT whitelisted — SHOULD be blocked via parent
        assert blocker._is_blocked("https://ads.example.com/tracker")
