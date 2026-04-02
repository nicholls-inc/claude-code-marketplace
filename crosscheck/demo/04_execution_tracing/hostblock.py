"""
Host-based content blocker for a web browser.

Blocks requests to known ad/tracker domains using a set-based lookup.
Supports both dynamic blocklists (downloaded from remote sources) and
user-configured blocked hosts.

SWE-bench_Pro: qutebrowser__qutebrowser-c580ebf0801e5a3ecabc54f327498bb753c6d5f2
"""

from __future__ import annotations

from dataclasses import dataclass, field
from urllib.parse import urlparse


@dataclass
class BlockerConfig:
    """Configuration for the host blocker."""
    blocking_enabled: bool = True
    whitelisted_hosts: set[str] = field(default_factory=set)


def widened_hostnames(hostname: str):
    """Generate progressively wider hostname matches.

    For 'a.b.example.com', yields:
        'a.b.example.com', 'b.example.com', 'example.com', 'com'

    Used elsewhere in the browser for URL pattern matching in
    per-site configuration lookups.
    """
    if hostname is None:
        return
    while hostname:
        yield hostname
        hostname = hostname.partition(".")[-1]


class HostBlocker:
    """Block requests to known ad/tracker hostnames.

    Maintains two hostname sets:
    - _blocked_hosts: downloaded from remote blocklists
    - _config_blocked_hosts: user-configured via settings

    A request is blocked if its hostname appears in either set,
    UNLESS the URL is whitelisted.
    """

    def __init__(self, config: BlockerConfig):
        self._config = config
        self._blocked_hosts: set[str] = set()
        self._config_blocked_hosts: set[str] = set()

    def update_blocklist(self, hosts: set[str]):
        """Update the dynamic blocklist from downloaded sources."""
        self._blocked_hosts = hosts

    def update_config_hosts(self, hosts: set[str]):
        """Update the user-configured blocked hosts."""
        self._config_blocked_hosts = hosts

    def is_whitelisted(self, url: str) -> bool:
        """Check if a URL's host is in the whitelist."""
        host = urlparse(url).hostname
        if host is None:
            return False
        return host in self._config.whitelisted_hosts

    def _is_blocked(self, request_url: str, first_party_url: str | None = None) -> bool:
        """Determine whether a request URL should be blocked.

        Args:
            request_url: The URL of the resource being requested
            first_party_url: The URL of the page making the request (for
                per-site blocking config). Currently unused but reserved
                for future per-site blocking toggle.

        Returns:
            True if the request should be blocked, False otherwise
        """
        if not self._config.blocking_enabled:
            return False

        host = urlparse(request_url).hostname
        if host is None:
            return False

        return (
            host in self._blocked_hosts or host in self._config_blocked_hosts
        ) and not self.is_whitelisted(request_url)

    def filter_request(self, request_url: str, first_party_url: str | None = None) -> bool:
        """Filter an incoming request. Returns True if the request was blocked."""
        if self._is_blocked(request_url, first_party_url):
            return True  # Block the request
        return False


# --- Diagnostic helpers for the demo ---

def diagnose_blocking(blocker: HostBlocker, url: str) -> dict:
    """Run diagnostic checks on why a URL is/isn't blocked."""
    host = urlparse(url).hostname
    return {
        'url': url,
        'hostname': host,
        'in_blocked_hosts': host in blocker._blocked_hosts,
        'in_config_blocked': host in blocker._config_blocked_hosts,
        'is_whitelisted': blocker.is_whitelisted(url),
        'is_blocked': blocker._is_blocked(url),
        'widened_hostnames': list(widened_hostnames(host)) if host else [],
    }
