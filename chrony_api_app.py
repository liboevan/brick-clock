from flask import Flask, request, jsonify
import subprocess
import os
import re

app = Flask(__name__)

# Default NTP servers
DEFAULT_SERVERS = ["pool.ntp.org"]
CHRONY_CONF_PATH = "/etc/chrony/chrony.conf"

# Helper to run chronyc commands
def run_chronyc(args):
    try:
        result = subprocess.run(["chronyc"] + args, capture_output=True, text=True, check=True)
        return result.stdout.strip(), None
    except subprocess.CalledProcessError as e:
        return e.stdout.strip() if e.stdout else '', e.stderr.strip() if e.stderr else 'Error'

# Helper to read/write allow directive in chrony.conf
def get_server_mode_status():
    try:
        with open(CHRONY_CONF_PATH) as f:
            conf = f.read()
        return any(line.strip().startswith("allow") for line in conf.splitlines())
    except Exception:
        return False

def set_server_mode_status(enabled):
    try:
        with open(CHRONY_CONF_PATH) as f:
            lines = f.readlines()
        new_lines = []
        found = False
        for line in lines:
            if line.strip().startswith("allow"):
                found = True
                if enabled:
                    new_lines.append(line)
                # else: skip the line to disable
            else:
                new_lines.append(line)
        if enabled and not found:
            new_lines.append("allow 0.0.0.0/0\n")
        with open(CHRONY_CONF_PATH, "w") as f:
            f.writelines(new_lines)
        return True
    except Exception as e:
        return False

def parse_sources_output(output):
    # Parse chronyc sources output into a list of dicts
    lines = output.splitlines()
    servers = []
    header_found = False
    for line in lines:
        if not header_found:
            if line.strip().startswith("="):
                header_found = True
            continue
        if line.strip() == '' or line.strip().startswith('='):
            continue
        # Example line: ^? 198.18.5.209 0 7 0 - +0ns[   +0ns] +/- 0ns
        parts = re.split(r'\s+', line.strip())
        if len(parts) >= 2:
            servers.append({
                "name": parts[1],
                "raw": line.strip()
            })
    return servers

def parse_tracking_output(output):
    # Parse chronyc tracking output into a dict
    result = {}
    for line in output.splitlines():
        if ':' in line:
            key, value = line.split(':', 1)
            result[key.strip()] = value.strip()
    return result

# RESTful endpoints
@app.route("/chrony/servers", methods=["GET"])
def list_servers():
    out, err = run_chronyc(["sources"])
    return jsonify({"servers": out, "error": err})

@app.route("/chrony/servers", methods=["PUT"])
def set_servers():
    data = request.get_json()
    servers = data.get("servers", [])
    if not servers or not isinstance(servers, list):
        return jsonify({"error": "servers must be a non-empty list"}), 400
    # Delete all current servers
    run_chronyc(["delete", "sources"])
    # Add new servers
    responses = []
    for server in servers:
        out, err = run_chronyc(["add", "server", server])
        responses.append({"server": server, "output": out, "error": err})
    return jsonify({"result": responses})

@app.route("/chrony/servers", methods=["DELETE"])
def reset_servers():
    # Delete all current servers
    out, err = run_chronyc(["delete", "sources"])
    return jsonify({"output": out, "error": err})

@app.route("/chrony/servers/default", methods=["PUT"])
def set_default_servers():
    # Delete all current servers
    run_chronyc(["delete", "sources"])
    # Add default servers
    responses = []
    for server in DEFAULT_SERVERS:
        out, err = run_chronyc(["add", "server", server])
        responses.append({"server": server, "output": out, "error": err})
    return jsonify({"result": responses})

@app.route("/chrony/status", methods=["GET"])
def chrony_status():
    # 1. Server mode
    server_mode_enabled = get_server_mode_status()
    # 2. chrony tracking (chronyc tracking)
    tracking_raw, tracking_err = run_chronyc(["tracking"])
    formatted_tracking = parse_tracking_output(tracking_raw) if tracking_raw else {}
    # 3. chrony sources (chronyc sources)
    sources_raw, sources_err = run_chronyc(["sources"])
    formatted_sources = parse_sources_output(sources_raw) if sources_raw else []
    return jsonify({
        "server_mode_enabled": server_mode_enabled,
        "tracking": formatted_tracking,
        "tracking_error": tracking_err,
        "sources": formatted_sources,
        "sources_error": sources_err
    })

@app.route("/chrony/version", methods=["GET"])
def chrony_version():
    out, err = run_chronyc(["--version"])
    return jsonify({"version": out, "error": err})

@app.route("/chrony/server-mode", methods=["GET"])
def get_server_mode():
    enabled = get_server_mode_status()
    return jsonify({"server_mode_enabled": enabled})

@app.route("/chrony/server-mode", methods=["PUT"])
def set_server_mode():
    data = request.get_json()
    enabled = data.get("enabled", None)
    if enabled is None or not isinstance(enabled, bool):
        return jsonify({"error": "'enabled' must be a boolean"}), 400
    success = set_server_mode_status(enabled)
    return jsonify({"success": success, "server_mode_enabled": enabled})

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8291) 