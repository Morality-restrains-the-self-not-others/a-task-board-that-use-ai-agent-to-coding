#!/usr/bin/env python3
"""Platform-service watchdog. :9999 down does not auto-restart runAll (opt-in --restart-runall)."""

from __future__ import annotations

import argparse
import os
import socket
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

_SCRIPT_DIR = str(Path(__file__).resolve().parent)
if _SCRIPT_DIR not in sys.path:
    sys.path.insert(0, _SCRIPT_DIR)

from watchdog_notify import (  # noqa: E402
    NO_RESTART_RUNALL_FLAG,  # noqa: F401 — re-export for tests
    format_last_orchestrator_exit_summary,
    install_cron,
    runall_restart_inhibited,
    watchdog_cron_line,  # noqa: F401 — re-export for tests
    write_no_restart_runall_flag,
)
from conf_local import overlay_conf_file  # noqa: E402

INFRA_HOST_PLACEHOLDER = "${INFRA_HOST}"

# 运行 All-in 编排外的、由 watchdog 自愈的组。
# platform：核心 API 微服务；infrastructure 中跳过 docker-*（由 docker compose 自愈）。
WATCH_GROUPS = ("platform", "infrastructure")
SKIP_SERVICES = {"docker-mysql", "docker-redis", "docker-kafka", "docker-portainer"}
# domain-events-intents / container-stack / value-stream 由各自 run.sh 监督，默认不接管；
# 需要时 --groups 可覆盖。

# runAll UI 默认 :9999；精准编译重启 / 全部重启等 bulk 窗口内禁止抢拉服务，
# 否则会与 kill-first 构建窗口竞态（端口被 watchdog 抢占 → 新进程 bind EADDRINUSE）。
DEFAULT_RUNALL_STATUS_URL = "http://127.0.0.1:9999/api/status"
LIFECYCLE_BUSY_STATUSES = frozenset({
    "building",
    "restarting",
    "starting",
    "stopping",
    "pending",
})

# OPT-20260813-006 曾在 :9999 失联时自动重启 runAll + start-all。
# 现默认关闭（热替换窗口会与 shutdown-self 打架）；opt-in --restart-runall。
DEFAULT_RUNALL_UI_URL = "http://127.0.0.1:9999/"
RUNALL_RECOVER_COOLDOWN_SEC = 240
RUNALL_RECOVER_STARTALL_WAIT_SEC = 90


def monorepo_root() -> Path:
    here = Path(__file__).resolve()
    for parent in here.parents:
        if (parent / ".gitmodules").is_file() and (parent / "conf" / "runAll.yaml").is_file():
            return parent
    raise FileNotFoundError(".gitmodules + conf/runAll.yaml not found above watchdog")


def log_dir(root: Path) -> Path:
    return Path(os.environ.get("RUNALL_LOG_ROOT", str(root / "logs")))


def log(root: Path, msg: str) -> None:
    line = f"{time.strftime('%F %T')} {msg}"
    print(line)
    try:
        with open(log_dir(root) / "service-watchdog.log", "a", encoding="utf-8") as fh:
            fh.write(line + "\n")
    except OSError:
        pass


def alert(root: Path, msg: str) -> None:
    line = f"{time.strftime('%F %T')} ALERT {msg}"
    print(line, file=sys.stderr)
    try:
        alert_file = log_dir(root) / "service-watchdog-alert.log"
        with open(alert_file, "a", encoding="utf-8") as fh:
            fh.write(line + "\n")
    except OSError:
        pass
    webhook = os.environ.get("WATCHDOG_ALERT_WEBHOOK_URL", "")
    if webhook:
        try:
            req = urllib.request.Request(
                webhook,
                data=line.encode("utf-8"),
                headers={"Content-Type": "text/plain"},
                method="POST",
            )
            urllib.request.urlopen(req, timeout=3)
        except Exception:  # noqa: BLE001 — 告警通道失败不得影响主流程
            pass


def load_services(root: Path, groups: tuple[str, ...]) -> list[dict]:
    """从 conf/runAll.yaml 读取服务拓扑（name / health url / start_command / working_dir）。"""
    cfg_path = root / "conf" / "runAll.yaml"
    cfg = overlay_conf_file(cfg_path)

    services: list[dict] = []
    for grp in cfg.get("groups", []):
        if grp.get("name") not in groups:
            continue
        for svc in grp.get("services", []):
            name = svc.get("name", "")
            if name in SKIP_SERVICES:
                continue
            hc = svc.get("health_check", {})
            url = hc.get("url") or ""
            if not url:
                health_path = hc.get("health_path") or "/api/health/"
                port = resolve_port(root, svc.get("conf_app") or "")
                if not port:
                    print(f"  skip {name}: cannot resolve port (conf_app={svc.get('conf_app')})")
                    continue
                url = f"http://127.0.0.1:{port}{health_path}"
            url = url.replace(INFRA_HOST_PLACEHOLDER, "127.0.0.1")
            services.append({
                "name": name,
                "url": url,
                "start_command": svc.get("start_command") or "",
                "working_dir": svc.get("working_dir") or ".",
                "timeout": min(int(hc.get("timeout") or 60), 180),
            })
    return services


def resolve_port(root: Path, conf_app: str) -> int | None:
    if not conf_app:
        return None
    cfg = root / "conf" / conf_app / "config.yaml"
    if not cfg.is_file():
        return None
    try:
        data = overlay_conf_file(cfg)
        return int(data.get("port") or data.get("httpPort") or 0) or None
    except Exception:  # noqa: BLE001
        return None


def url_port(url: str) -> int | None:
    from urllib.parse import urlparse

    try:
        return urlparse(url).port
    except ValueError:
        return None


def is_healthy(url: str) -> bool:
    try:
        with urllib.request.urlopen(url, timeout=5) as resp:
            return 200 <= resp.status < 400
    except (urllib.error.URLError, OSError, ValueError):
        return False


def runall_status_url() -> str:
    return (os.environ.get("RUNALL_STATUS_URL") or DEFAULT_RUNALL_STATUS_URL).strip()


def fetch_runall_status(status_url: str | None = None) -> dict | None:
    """读取 runAll /api/status；不可达时返回 None（fail-open：不阻断自愈）。"""
    url = (status_url or runall_status_url()).strip()
    if not url:
        return None
    try:
        with urllib.request.urlopen(url, timeout=2) as resp:
            import json

            payload = json.loads(resp.read().decode("utf-8"))
            return payload if isinstance(payload, dict) else None
    except (urllib.error.URLError, OSError, ValueError, TimeoutError):
        return None


BULK_LOCK_MAX_AGE_SEC = 2 * 3600  # 锁文件 >2h 视为过期（runAll 崩溃残留），忽略并告警


def bulk_op_lock_info(root: Path) -> dict | None:
    """读取 runAll bulk 操作本地锁（.runall/bulk_op.lock，OPT-20260810-042）。

    runAll 自身 hot-replace / 短暂宕机时 /api/status 不可达，门禁 fail-open 仍可能
    在窗口内抢拉端口。这里优先读锁文件：存在且未过期 → 视为 bulk 进行中；过期
    → 返回带 expired 标记（忽略并告警）。文件缺失/非法 → None（回退 API）。
    """
    lock_path = root / ".runall" / "bulk_op.lock"
    try:
        if not lock_path.is_file():
            return None
        import json

        data = json.loads(lock_path.read_text(encoding="utf-8"))
        if not isinstance(data, dict):
            return None
        kind = str(data.get("kind") or "").strip()
        ts = int(data.get("ts") or 0)
        if not kind or not ts:
            return None
        age = time.time() - ts
        if age > BULK_LOCK_MAX_AGE_SEC:
            return {"kind": kind, "expired": True, "age_sec": int(age)}
        return {"kind": kind, "expired": False, "age_sec": int(age)}
    except Exception:  # noqa: BLE001 — 锁文件损坏视为无锁，回退 API
        return None


def orchestration_blocks_restart(root: Path, status: dict | None, service_name: str) -> str:
    """若 runAll 正编排该服务（bulk / 生命周期态），返回跳过原因；否则空串。

    优先读本地锁文件（API 不可达时仍能感知 bulk）；过期锁忽略但告警。
    """
    lock = bulk_op_lock_info(root)
    if lock:
        if lock.get("expired"):
            alert(root, f"stale bulk_op.lock found (kind={lock.get('kind')}, age={lock.get('age_sec')}s) — ignoring")
        else:
            return f"runAll bulk op lock active ({lock.get('kind')})"
    if not status:
        return ""
    bulk = status.get("active_bulk_progress")
    if isinstance(bulk, dict) and bulk.get("kind"):
        return f"runAll bulk op active ({bulk.get('kind')})"
    for svc in status.get("services") or []:
        if not isinstance(svc, dict):
            continue
        if svc.get("name") != service_name:
            continue
        st = str(svc.get("status") or "").strip().lower()
        if st in LIFECYCLE_BUSY_STATUSES:
            return f"runAll service status={st}"
        return ""
    return ""


def port_listening(url: str) -> bool:
    """端口是否有监听（仅判断可连接，不判健康）。"""
    port = url_port(url)
    if not port:
        return False
    try:
        with socket.create_connection(("127.0.0.1", port), timeout=2):
            return True
    except OSError:
        return False


def binary_present(root: Path, svc: dict) -> bool:
    """start_command 的可执行文件（./bin/xxx 等相对路径）是否存在于 working_dir。"""
    cmd = (svc["start_command"] or "").strip()
    token = cmd.split()[0] if cmd else ""
    if not token or token in ("bash", "sh", "python3"):
        return True  # 包装脚本由 run.sh 体系负责，不预先校验
    exe = Path(token)
    if not exe.is_absolute():
        exe = root / svc["working_dir"] / token
    return exe.exists()


def start_service(root: Path, svc: dict) -> bool:
    """setsid nohup 拉起服务，输出到 logs/<name>.log。返回拉起是否成功。"""
    cmd = (svc["start_command"] or "").strip()
    if not cmd:
        return False
    workdir = root / svc["working_dir"]
    service_log = log_dir(root) / f"{svc['name']}.log"
    service_log.parent.mkdir(parents=True, exist_ok=True)
    try:
        with open(service_log, "a", encoding="utf-8") as fh:
            fh.write(f"\n[{time.strftime('%F %T')}] watchdog start: {cmd}\n")
            fh.flush()
            # setsid 脱离会话；nohup 防 SIGHUP；输出追加到服务日志
            subprocess.Popen(
                ["setsid", "nohup", *cmd.split()],
                cwd=str(workdir),
                stdout=fh,
                stderr=subprocess.STDOUT,
                stdin=subprocess.DEVNULL,
                start_new_session=True,
            )
        return True
    except OSError as exc:
        alert(root, f"service {svc['name']}: failed to spawn {cmd}: {exc}")
        return False


def wait_healthy(url: str, timeout: int) -> bool:
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        if is_healthy(url):
            return True
        time.sleep(4)
    return is_healthy(url)


def check_service(
    root: Path,
    svc: dict,
    verbose: bool,
    runall_status: dict | None = None,
) -> str:
    """单服务检查。返回描述行（供汇总）。"""
    url = svc["url"]
    if is_healthy(url):
        if verbose:
            return f"ok    {svc['name']:<28} {url}"
        return ""
    if not port_listening(url):
        if not binary_present(root, svc):
            alert(root, f"service {svc['name']}: unhealthy ({url}) and binary missing — run build first")
            return f"BUILD {svc['name']:<28} {url}"
        block_reason = orchestration_blocks_restart(root, runall_status, svc["name"])
        if block_reason:
            log(root, f"skip restart {svc['name']} ({url}) — {block_reason}")
            return f"SKIP  {svc['name']:<28} {block_reason}"
        log(root, f"unhealthy {svc['name']} ({url}) — no listener, restarting")
        if not start_service(root, svc):
            return f"FAIL  {svc['name']:<28} {url}"
        if wait_healthy(url, svc["timeout"]):
            log(root, f"recovered {svc['name']} ({url})")
            return f"RESTARTED ok {svc['name']:<22} {url}"
        alert(root, f"service {svc['name']}: restarted but still unhealthy ({url}) — check logs/{svc['name']}.log")
        return f"UNHEALTHY {svc['name']:<20} {url}"
    alert(root, f"service {svc['name']}: unhealthy but port listening ({url}) — manual attention")
    return f"LISTENING-UNHEALTHY {svc['name']:<8} {url}"


RUNALL_RECOVER_STAMP = ".runall/runall_recover.last"


def runall_ui_up(url: str | None = None) -> bool:
    """runAll UI 是否响应（GET :9999/，2s 超时）。"""
    target = (url or DEFAULT_RUNALL_UI_URL).strip()
    try:
        with urllib.request.urlopen(target, timeout=2):
            return True
    except (urllib.error.URLError, OSError, ValueError):
        return False


def runall_recovery_allowed(root: Path, now: int | None = None) -> bool:
    """冷却门禁：距上次自动恢复 ≥ RUNALL_RECOVER_COOLDOWN_SEC 才允许再次拉起。"""
    now = now if now is not None else int(time.time())
    stamp = root / RUNALL_RECOVER_STAMP
    try:
        prev = int((stamp.read_text(encoding="utf-8").strip() or "0"))
    except (OSError, ValueError):
        prev = 0
    return (now - prev) >= RUNALL_RECOVER_COOLDOWN_SEC


def mark_runall_recovery(root: Path, now: int | None = None) -> None:
    now = now if now is not None else int(time.time())
    (root / ".runall").mkdir(parents=True, exist_ok=True)
    (root / RUNALL_RECOVER_STAMP).write_text(str(now), encoding="utf-8")


def spawn_runall(root: Path) -> bool:
    """setsid nohup 拉起 runAll UI（与 run.sh 同款：cwd=runAll, config=../conf/runAll.yaml）。"""
    bin_path = root / "runAll" / "bin" / "runAll"
    if not bin_path.is_file():
        return False
    console_log = log_dir(root) / "runall-console.log"
    console_log.parent.mkdir(parents=True, exist_ok=True)
    env = dict(os.environ)
    env.setdefault("INFRA_HOST", "10.2.150.68")
    try:
        with open(console_log, "a", encoding="utf-8") as fh:
            fh.write(f"\n[{time.strftime('%F %T')}] watchdog runAll restart\n")
            fh.flush()
            subprocess.Popen(
                ["setsid", "nohup", str(bin_path), "--config", "../conf/runAll.yaml"],
                cwd=str(root / "runAll"),
                stdout=fh,
                stderr=subprocess.STDOUT,
                stdin=subprocess.DEVNULL,
                start_new_session=True,
                env=env,
            )
        return True
    except OSError as exc:
        alert(root, f"runAll respawn failed: {exc}")
        return False


def trigger_start_all(root: Path, wait_sec: int | None = None) -> bool:
    """等待 :9999 起来后 POST /api/start-all（带 watchdog session_id）。"""
    deadline = time.monotonic() + (wait_sec or RUNALL_RECOVER_STARTALL_WAIT_SEC)
    session_id = f"watchdog-{int(time.time())}"
    endpoint = DEFAULT_RUNALL_UI_URL.rstrip("/") + "/api/start-all"
    import json as _json

    while time.monotonic() < deadline:
        if not runall_ui_up():
            time.sleep(2)
            continue
        try:
            req = urllib.request.Request(
                endpoint,
                data=_json.dumps({"session_id": session_id}).encode("utf-8"),
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            with urllib.request.urlopen(req, timeout=5) as resp:
                payload = _json.loads(resp.read().decode("utf-8") or "{}")
                if isinstance(payload, dict) and (payload.get("run_id") or payload.get("status") == "accepted"):
                    return True
        except (urllib.error.URLError, OSError, ValueError, TimeoutError):
            time.sleep(2)
    return False


def recover_runall(root: Path, restart: bool = False, verbose: bool = False) -> str:
    """:9999 失联时默认只告警；--restart-runall 且无禁止文件才拉起。"""
    url = DEFAULT_RUNALL_UI_URL
    if runall_ui_up(url):
        return "ok    runall-ui (:9999)" if verbose else ""
    exit_summary = format_last_orchestrator_exit_summary(root)
    exit_clause = f"; {exit_summary}" if exit_summary else ""
    if runall_restart_inhibited(root) or not restart:
        why = ".runall/no_restart_runall" if runall_restart_inhibited(root) else "--no-restart-runall"
        alert(root, f"runAll UI (:9999) down — auto-restart disabled ({why}){exit_clause}")
        return "DOWN  runall-ui (:9999) (restart disabled)"
    if not runall_recovery_allowed(root):
        log(root, f"runAll UI (:9999) down but within cooldown — skip auto-restart{exit_clause}")
        return "COOLDOWN runall-ui (:9999)"
    alert(root, f"runAll UI (:9999) down — restarting + start-all{exit_clause}")
    if not spawn_runall(root):
        alert(root, "runAll UI down and runAll/bin/runAll missing — run build first")
        return "BUILD  runall-ui (:9999)"
    mark_runall_recovery(root)
    log(root, "runAll respawned, waiting for UI + start-all...")
    if not trigger_start_all(root):
        alert(root, "runAll respawned but /api/start-all not accepted within "
                    f"{RUNALL_RECOVER_STARTALL_WAIT_SEC}s")
        return "STARTALL-FAIL runall-ui (:9999)"
    log(root, "runAll restart + start-all accepted")
    return "RESTARTED runall-ui (:9999)"


def main() -> int:
    parser = argparse.ArgumentParser(description="服务进程级自愈 watchdog")
    parser.add_argument("--verbose", action="store_true", help="健康服务也输出")
    parser.add_argument("--install-cron", action="store_true", help="幂等安装/更新 crontab 条目并退出")
    parser.add_argument("--groups", default=",".join(WATCH_GROUPS),
                        help="watch 的 runAll 组（逗号分隔，默认 platform,infrastructure）")
    parser.add_argument("--root", default="", help="monorepo 根（默认自动探测）")
    parser.add_argument("--no-restart-runall", action="store_true",
                        help=":9999 失联时只告警，不自动重启 runAll（默认 cron 行为）")
    parser.add_argument("--restart-runall", action="store_true",
                        help="opt-in：:9999 失联时自动重启 runAll 并 start-all（禁止文件仍优先生效）")
    parser.add_argument("--notify-no-restart-runall", action="store_true",
                        help="写入 .runall/no_restart_runall 并更新 crontab 后退出")
    args = parser.parse_args()

    root = Path(args.root).resolve() if args.root else monorepo_root()
    if args.notify_no_restart_runall:
        path = write_no_restart_runall_flag(root)
        ok = install_cron(root)
        print(f"notified watchdog: {path}")
        return 0 if ok else 1
    if args.install_cron:
        return 0 if install_cron(root) else 1

    groups = tuple(g.strip() for g in args.groups.split(",") if g.strip())
    services = load_services(root, groups)
    if not services:
        print(f"ERROR: no watchable services in groups {groups}", file=sys.stderr)
        return 2

    # 一次拉取 runAll 状态，供 bulk / 生命周期门禁复用（避免每服务打一次 API）。
    runall_status = fetch_runall_status()
    results = [check_service(root, svc, args.verbose, runall_status) for svc in services]
    lines = [r for r in results if r]

    # :9999 失联默认不拉起 runAll；--restart-runall 可 opt-in，禁止文件仍阻断。
    restart_runall = args.restart_runall and not args.no_restart_runall
    runall_line = recover_runall(root, restart=restart_runall, verbose=args.verbose)
    if runall_line:
        lines.append(runall_line)

    if lines:
        print("\n".join(lines))
    else:
        print(f"all {len(services)} services healthy")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
