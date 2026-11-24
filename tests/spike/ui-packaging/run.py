"""Reproduksi fixture T05; seluruh output build berada di direktori sementara baru."""

import argparse
import base64
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import shutil
import subprocess
import tarfile
import tempfile
import time
import urllib.error
import urllib.request

SOURCE = Path(__file__).resolve().parent
REPO = SOURCE.parents[2]


def require(condition, category):
    if not condition:
        raise RuntimeError(category)


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def prepare(work):
    require(work.resolve().is_relative_to(Path(tempfile.gettempdir()).resolve()), "work_must_be_temporary")
    work.mkdir(mode=0o700)  # Fail jika sudah ada: jangan menimpa hasil atau secret run sebelumnya.
    context = work / "context"
    context.mkdir()
    for directory in ("backend", "web", "browser"):
        for source in (SOURCE / directory).rglob("*"):
            require(not source.is_symlink(), "symlink_source_rejected")
            relative = source.relative_to(SOURCE / directory)
            if source.is_file() and source.suffix in (".go", ".mod", ".html", ".ts", ".js", ".json", ".svelte"):
                require(not any(p in ("dist", "node_modules", "test-results") for p in relative.parts), "generated_source_rejected")
                target = context / ("web/browser" if directory == "browser" else directory) / relative
                target.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(source, target)
    for original, target in (("frontend-package.json", "package.json"), ("frontend-package-lock.json", "package-lock.json")):
        shutil.copyfile(REPO / "docs/m0/ep-01-t02" / original, context / "web" / target)
    shutil.copyfile(SOURCE / "Dockerfile.gateway", context / "Dockerfile.gateway")
    return context


class Run:
    def __init__(self, work):
        self.work = work
        self.records = []
        self.env = dict(os.environ, GOTOOLCHAIN="local", GOPROXY="off", GOSUMDB="off", GOENV="off", GOCACHE=str(work / "go-cache"))

    def command(self, name, command, cwd=None, expected=0, timeout=600):
        print(name, flush=True)
        result = subprocess.run(command, cwd=cwd, env=self.env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=timeout)
        (self.work / (name + ".log")).write_bytes(result.stdout)
        self.records.append({"id": name, "command": command, "exit": result.returncode, "expected_exit": expected})
        require(result.returncode == expected if expected is not None else result.returncode != 0, name + "_failed")
        return result.stdout


def request(origin, path, authorization=None, method="GET"):
    headers = {"Authorization": authorization} if authorization else {}
    req = urllib.request.Request(origin + path, headers=headers, method=method)
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    try:
        response = opener.open(req, timeout=5)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        return response.status, response.headers, response.read(2 * 1024 * 1024)


def finish(run, compose, image, secret_file, report, started):
    cleanup_error = None
    try:
        if started:
            run.command("compose-cleanup", compose + ["down", "--timeout", "5"])
    except (RuntimeError, OSError, subprocess.SubprocessError):
        cleanup_error = "compose_cleanup_failed"
    try:
        # Tag unik milik run ini; tidak menghapus base image, volume, atau resource existing.
        if any(c["id"] == "image-build" and c["exit"] == 0 for c in run.records):
            result = subprocess.run(["docker", "image", "rm", image], env=run.env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=30)
            if result.returncode != 0:
                cleanup_error = cleanup_error or "image_cleanup_failed"
    except (OSError, subprocess.SubprocessError):
        cleanup_error = cleanup_error or "image_cleanup_failed"
    finally:
        secret_file.unlink(missing_ok=True)
        if cleanup_error:
            report.update(status="failed", cleanup_error=cleanup_error)
        (run.work / "checks.json").write_text(json.dumps(report, indent=2) + "\n")
    if cleanup_error:
        raise RuntimeError(cleanup_error)


def smoke(work):
    context = prepare(work)
    run = Run(work)
    token = secrets.token_hex(24)
    gateway_token = secrets.token_hex(24)
    secret_file = work / "admin"
    secret_file.write_text(token)
    # Parent host 0700; file sintetis 0444 dibaca UID65532 via mount RO Docker.
    secret_file.chmod(0o444)
    project = "opencode-infinity-t05-" + secrets.token_hex(6)
    image = project + ":fixture"
    run.env.update(T05_IMAGE=image, T05_ADMIN_SECRET_FILE=str(secret_file), PLAYWRIGHT_BROWSERS_PATH=os.environ.get("PLAYWRIGHT_BROWSERS_PATH", str(work / "browsers")))
    compose = ["docker", "compose", "-p", project, "-f", str(SOURCE / "compose.yaml")]
    started = False
    report = {"task": "EP-01-T05", "status": "failed", "scope": "independent synthetic UI packaging; not dashboard AC-20–AC-22 or M0 acceptance", "checks": run.records}
    try:
        require(run.command("node-version", ["node", "--version"]).strip() == b"v26.13.1", "node_pin_mismatch")
        require(run.command("npm-version", ["npm", "--version"]).strip() == b"12.2.2", "npm_pin_mismatch")
        report["go_host"] = run.command("go-version", ["go", "version"]).decode().strip()
        report["docker"] = run.command("docker-version", ["docker", "version", "--format", "{{.Server.Version}} {{.Server.Os}}/{{.Server.Arch}}"] ).decode().strip()
        report["compose"] = run.command("compose-version", ["docker", "compose", "version", "--short"]).decode().strip()
        run.command("npm-ci", ["npm", "ci", "--ignore-scripts", "--strict-peer-deps", "--no-audit", "--no-fund"], context / "web")
        run.command("frontend-check", ["npm", "run", "check"], context / "web")
        run.command("frontend-build", ["npm", "run", "build"], context / "web")
        built = context / "web/dist"
        dist = context / "backend/internal/webui/dist"
        shutil.copytree(built, dist)
        for name, removed in (("no-assets", "dist"), ("no-index", "index.html"), ("no-js", "js"), ("no-css", "css")):
            negative = work / name
            shutil.copytree(context / "backend", negative)
            target = negative / "internal/webui/dist"
            if removed == "dist": shutil.rmtree(target)
            elif removed == "index.html": (target / removed).unlink()
            else:
                for path in target.glob("assets/*." + removed): path.unlink()
            result = run.command(name, ["go", "build", "-buildvcs=false", "./cmd/gateway"], negative, expected=None)
            require(b"no matching files found" in result, "negative_build_wrong_reason")
        run.command("backend-test", ["go", "test", "-count=1", "./..."], context / "backend")
        run.command("backend-race", ["go", "test", "-race", "-count=1", "./..."], context / "backend")
        run.command("backend-vet", ["go", "vet", "./..."], context / "backend")
        require(not run.command("backend-format", ["gofmt", "-l", "cmd", "internal"], context / "backend").strip(), "gofmt_required")
        # Staging build hanya source/manifest: dependency dan dist host tidak masuk context Docker.
        build_context = work / "image-source"
        shutil.copytree(context, build_context, ignore=shutil.ignore_patterns("node_modules", "dist", "test-results"))
        run.command("image-build", ["docker", "build", "--platform", "linux/amd64", "--label", "opencode-infinity.task=EP-01-T05", "-t", image, "-f", "Dockerfile.gateway", "."], build_context, timeout=1800)
        run.command("compose-config", compose + ["config", "--quiet"])
        started = True
        run.command("compose-up", compose + ["up", "-d", "--no-build", "--pull", "never"])
        container = run.command("container-id", compose + ["ps", "-q", "gateway"]).decode().strip()
        require(bool(container) and "\n" not in container, "unexpected_service_count")
        info = json.loads(run.command("runtime-inspect", ["docker", "inspect", container]))[0]
        image_info = json.loads(run.command("image-inspect", ["docker", "image", "inspect", image]))[0]
        require(image_info["Architecture"] == "amd64" and image_info["Os"] == "linux", "architecture_mismatch")
        require(info["Config"]["User"] == "65532:65532" and info["Config"]["Entrypoint"] == ["/gateway"], "runtime_identity_mismatch")
        host = info["HostConfig"]
        require(host["ReadonlyRootfs"] and host["CapDrop"] == ["ALL"] and not host["Privileged"], "runtime_privilege_mismatch")
        require("no-new-privileges:true" in host["SecurityOpt"] and not host.get("CapAdd"), "unexpected_capability")
        require(host["PidsLimit"] == 64 and host["Memory"] == 128 * 1024 * 1024, "runtime_limit_mismatch")
        require(len(info["Mounts"]) == 1 and info["Mounts"][0]["Destination"] == "/run/secrets/admin" and not info["Mounts"][0]["RW"], "unexpected_runtime_mount")
        ports = info["NetworkSettings"]["Ports"]
        require(list(ports) == ["8788/tcp"] and isinstance(ports["8788/tcp"], list) and len(ports["8788/tcp"]) == 1, "unexpected_ports")
        binding = ports["8788/tcp"][0]
        require(binding["HostIp"] == "127.0.0.1", "non_loopback_publish")
        origin = "http://127.0.0.1:" + binding["HostPort"]
        auth = "Basic " + base64.b64encode(("admin:" + token).encode()).decode()
        for attempt in range(40):
            try:
                status, _, body = request(origin, "/", auth)
                if status == 200: break
            except OSError: pass
            time.sleep(0.1)
        else: raise RuntimeError("runtime_unavailable")
        assets = re.findall(rb'(?:src|href)="(/assets/[^" ]+)"', body)
        require(any(a.endswith(b".js") for a in assets) and any(a.endswith(b".css") for a in assets), "asset_references_missing")
        for path in ["/"] + [a.decode() for a in assets]:
            status, headers, body = request(origin, path, auth)
            require(status == 200, "asset_http_failed")
            require(body == (built / ("index.html" if path == "/" else path[1:])).read_bytes(), "embedded_bytes_mismatch")
            for denied in (None, "Basic " + base64.b64encode(b"admin:wrong").decode(), "Bearer " + gateway_token, "Basic !!!"):
                denied_status, _, denied_body = request(origin, path, denied)
                require(denied_status == 401 and token.encode() not in denied_body and gateway_token.encode() not in denied_body, "runtime_auth_failed")
        run.env["T05_ORIGIN"] = origin
        playwright = context / "web/node_modules/playwright/cli.js"
        run.command("browser-install", ["node", str(playwright), "install", "chromium"], timeout=600)
        run.command("browser-test", ["node", str(playwright), "test", "--config", "browser/playwright.config.ts"], context / "web")
        top = run.command("runtime-processes", ["docker", "top", container, "-eo", "pid,comm"]).decode().splitlines()
        require(len(top) == 2 and top[1].split()[1:] == ["gateway"], "unexpected_runtime_process")
        run.command("runtime-export", ["docker", "export", "-o", str(work / "runtime.tar"), container])
        banned = {"node", "nodejs", "npm", "npx", "go", "gcc", "cc", "make", "chrome", "chromium", "chrome-headless-shell", "ffmpeg", "sh", "bash", "dash", "busybox"}
        with tarfile.open(work / "runtime.tar") as archive:
            members = archive.getmembers()
            require(not any(Path(m.name).name in banned or "node_modules" in Path(m.name).parts for m in members), "runtime_contains_toolchain")
            binary = archive.extractfile("gateway").read()
            require(binary.startswith(b"\x7fELF"), "runtime_binary_not_elf")
            require(token.encode() not in binary and gateway_token.encode() not in binary, "runtime_secret_leak")
        logs = run.command("runtime-logs", ["docker", "logs", container])
        for path in list(built.rglob("*")) + list(work.glob("*.log")):
            if path.is_file():
                data = path.read_bytes()
                require(token.encode() not in data and gateway_token.encode() not in data, "artifact_secret_leak")
        require(token.encode() not in logs, "log_secret_leak")
        report.update(status="passed", image_id=image_info["Id"], binary_sha256=hashlib.sha256(binary).hexdigest(), runtime_files=len(members), runtime_processes=1, origin=origin, architecture="linux/amd64", uid=65532, assets=[a.decode() for a in assets], asset_sha256={str(p.relative_to(built)): digest(p) for p in built.rglob("*") if p.is_file()}, loopback_only=True, toolchain_absent=True, auth_negative_passed=True, secret_absent=True)
    finally:
        finish(run, compose, image, secret_file, report, started)
    print("T05 passed; evidence: " + str(work / "checks.json"), flush=True)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--work-dir", required=True, type=Path)
    args = parser.parse_args()
    try:
        smoke(args.work_dir.absolute())
    except (RuntimeError, OSError, subprocess.SubprocessError) as error:
        # Jangan mencetak isi stdout/config/header atau nilai credential pada error.
        print("T05 failed: " + (str(error) if isinstance(error, RuntimeError) else type(error).__name__), flush=True)
        raise SystemExit(1)
