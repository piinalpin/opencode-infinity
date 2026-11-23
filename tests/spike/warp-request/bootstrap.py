"""Bootstrap WARP fixture eksplisit; tidak pernah dipanggil oleh startup relay."""

import base64
import ipaddress
import re
import uuid
from datetime import datetime, timezone


class BootstrapError(Exception):
    """Kode kategori aman tanpa response/header/profile upstream."""


def make_profile(private_key, response, resolve):
    try:
        for key in [private_key, response["config"]["peers"][0]["public_key"]]:
            if len(base64.b64decode(key, validate=True)) != 32:
                raise ValueError()
        uuid.UUID(response["id"])
        if not re.fullmatch(r"[A-Za-z0-9._~+/=-]{10,2048}", response["token"]):
            raise ValueError()
        address = ipaddress.IPv4Address(response["config"]["interface"]["addresses"]["v4"])
        if not address.is_private or address.is_loopback or address.is_unspecified:
            raise ValueError()
        endpoint = response["config"]["peers"][0]["endpoint"]["host"]
        host, port = endpoint.rsplit(":", 1)
        if not port.isdecimal() or not 1 <= int(port) <= 65535:
            raise ValueError()
        if host == "engage.cloudflareclient.com":
            endpoint_ip = ipaddress.IPv4Address(resolve(host))
        else:
            endpoint_ip = ipaddress.IPv4Address(host)
        if not endpoint_ip.is_global:
            raise ValueError()
        peer = response["config"]["peers"][0]["public_key"]
    except (ValueError, KeyError, TypeError, IndexError, AttributeError):
        raise BootstrapError("invalid_registration") from None
    return ("[Interface]\nPrivateKey = " + private_key + "\nAddress = " + str(address)
            + "/32\nMTU = 1280\nTable = off\n\n[Peer]\nPublicKey = " + peer
            + "\nAllowedIPs = 0.0.0.0/0\nEndpoint = " + str(endpoint_ip) + ":" + str(int(port))
            + "\nPersistentKeepalive = 15\n")


def register_profile(private_key, public_key, send, resolve):
    try:
        if len(base64.b64decode(public_key, validate=True)) != 32:
            raise BootstrapError("invalid_public_key")
        endpoint = "https://api.cloudflareclient.com/v0a884/reg"
        payload = {"key": public_key, "install_id": "", "tos": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S.000Z"),
                   "type": "Android", "model": "PC", "locale": "en_US"}
        response = send("POST", endpoint, payload, {})
        profile = make_profile(private_key, response, resolve)
        send("PATCH", endpoint + "/" + str(uuid.UUID(response["id"])), {"warp_enabled": True},
             {"Authorization": "Bearer " + response["token"]})
        return profile
    except BootstrapError:
        raise
    except Exception:
        raise BootstrapError("bootstrap_request_failed") from None


def api_send(method, url, data, headers):
    import json
    import ssl
    import urllib.request
    import urllib.error

    class NoRedirect(urllib.request.HTTPRedirectHandler):
        def redirect_request(self, req, fp, code, msg, hdrs, newurl):
            return None

    # No implicit environment proxy, redirect, credential store or retry.
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect(),
                                        urllib.request.HTTPSHandler(context=ssl.create_default_context()))
    request = urllib.request.Request(url, data=json.dumps(data).encode(), method=method,
                                    headers={"Content-Type": "application/json", "User-Agent": "okhttp/3.12.1", **headers})
    try:
        with opener.open(request, timeout=20) as response:
            payload = response.read(64 * 1024 + 1)
            if response.status != 200 or len(payload) > 64 * 1024:
                raise BootstrapError("cloudflare_response_limit_or_status")
            return json.loads(payload) if payload else {}
    except urllib.error.HTTPError as error:
        raise BootstrapError("cloudflare_http_" + str(error.code)) from None
    except BootstrapError:
        raise
    except Exception:
        raise BootstrapError("cloudflare_transport_or_json") from None


def main():
    import argparse
    import json
    import os
    import socket
    import subprocess
    from pathlib import Path

    parser = argparse.ArgumentParser(description="Explicit bootstrap of a NEW T04 private WARP profile")
    parser.add_argument("--project", required=True)
    parser.add_argument("--compose-file", required=True)
    parser.add_argument("--state-dir", required=True)
    args = parser.parse_args()
    state = Path(args.state_dir)
    if not re.fullmatch(r"opencode-infinity-t04-[a-f0-9]{8}", args.project):
        raise BootstrapError("invalid_fixture_project")
    if not state.is_dir() or state.stat().st_uid != os.getuid() or state.stat().st_mode & 0o077:
        raise BootstrapError("state_directory_not_private")
    # A pre-existing profile volume is never overwritten or rekeyed.
    existing = subprocess.run(["docker", "volume", "inspect", args.project + "_profile"], capture_output=True)
    if existing.returncode == 0:
        raise BootstrapError("profile_volume_already_exists")
    try:
        fd = os.open(state / "bootstrap.used", os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        os.close(fd)
    except OSError:
        raise BootstrapError("bootstrap_budget_used") from None
    key = subprocess.run(["wg", "genkey"], capture_output=True, text=True, timeout=5)
    if key.returncode:
        raise BootstrapError("key_generation")
    private = key.stdout.strip()
    pub = subprocess.run(["wg", "pubkey"], input=private + "\n", capture_output=True, text=True, timeout=5)
    if pub.returncode:
        raise BootstrapError("public_key_generation")
    attempts = []
    def send(method, url, data, headers):
        attempts.append({"method": method, "phase": "registration" if method == "POST" else "enable"})
        (state / "bootstrap-attempts.json").write_text(json.dumps(attempts) + "\n")
        return api_send(method, url, data, headers)
    def resolve(host):
        return socket.getaddrinfo(host, None, socket.AF_INET, socket.SOCK_DGRAM)[0][4][0]
    profile = register_profile(private, pub.stdout.strip(), send, resolve)
    # Secret is carried on stdin; argv/output/config interpolation contain no key.
    write = subprocess.run(["docker", "compose", "-p", args.project, "-f", args.compose_file,
                            "run", "--rm", "--no-deps", "-T", "relay", "write-profile"],
                           input=profile.encode(), capture_output=True, timeout=45)
    if write.returncode:
        raise BootstrapError("private_volume_write")
    print(json.dumps({"phase": "bootstrap", "outcome": "profile_saved", "attempts": attempts}))


if __name__ == "__main__":
    import json
    try:
        main()
    except BootstrapError as error:
        print(json.dumps({"phase": "bootstrap", "outcome": "failed", "category": str(error)}))
        raise SystemExit(1)
    except Exception:
        print(json.dumps({"phase": "bootstrap", "outcome": "failed", "category": "bootstrap_local_failure"}))
        raise SystemExit(1)
