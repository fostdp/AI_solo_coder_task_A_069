import struct
import socket
import threading
import time
import random
import math
import json
import argparse
from http.server import HTTPServer, BaseHTTPRequestHandler

DEVICE_CONFIGS = {
    "chiller": {
        "params": {
            "supply_temp":      {"offset": 0,  "min": 5.0,  "max": 12.0,  "nominal": 7.0},
            "return_temp":      {"offset": 2,  "min": 10.0, "max": 18.0,  "nominal": 12.0},
            "flow_rate":        {"offset": 4,  "min": 50.0, "max": 200.0, "nominal": 120.0},
            "power":            {"offset": 6,  "min": 200.0,"max": 600.0, "nominal": 350.0},
            "pressure":         {"offset": 8,  "min": 0.2,  "max": 0.8,   "nominal": 0.5},
            "cop":              {"offset": 10, "min": 2.0,  "max": 8.0,   "nominal": 5.5},
            "cooling_capacity": {"offset": 12, "min": 800.0,"max": 2500.0,"nominal": 1800.0},
        },
        "name_template": "CHU-{:03d}",
        "base_addr": 0,
    },
    "cooling_tower": {
        "params": {
            "supply_temp":      {"offset": 0,  "min": 25.0, "max": 35.0,  "nominal": 28.0},
            "return_temp":      {"offset": 2,  "min": 30.0, "max": 42.0,  "nominal": 35.0},
            "flow_rate":        {"offset": 4,  "min": 80.0, "max": 250.0, "nominal": 150.0},
            "power":            {"offset": 6,  "min": 30.0, "max": 100.0, "nominal": 60.0},
            "pressure":         {"offset": 8,  "min": 0.1,  "max": 0.5,   "nominal": 0.3},
            "cop":              {"offset": 10, "min": 3.0,  "max": 10.0,  "nominal": 7.0},
            "cooling_capacity": {"offset": 12, "min": 500.0,"max": 2000.0,"nominal": 1200.0},
        },
        "name_template": "CT-{:03d}",
        "base_addr": 100,
    },
    "precision_ac": {
        "params": {
            "supply_temp":      {"offset": 0,  "min": 15.0, "max": 25.0,  "nominal": 20.0},
            "return_temp":      {"offset": 2,  "min": 25.0, "max": 40.0,  "nominal": 32.0},
            "flow_rate":        {"offset": 4,  "min": 5.0,  "max": 30.0,  "nominal": 15.0},
            "power":            {"offset": 6,  "min": 20.0, "max": 80.0,  "nominal": 45.0},
            "pressure":         {"offset": 8,  "min": 0.1,  "max": 0.4,   "nominal": 0.2},
            "cop":              {"offset": 10, "min": 1.5,  "max": 6.0,   "nominal": 3.5},
            "cooling_capacity": {"offset": 12, "min": 50.0, "max": 200.0, "nominal": 130.0},
        },
        "name_template": "PAC-{:03d}",
        "base_addr": 200,
    },
    "cdu": {
        "params": {
            "supply_temp":      {"offset": 0,  "min": 14.0, "max": 22.0,  "nominal": 18.0},
            "return_temp":      {"offset": 2,  "min": 22.0, "max": 35.0,  "nominal": 28.0},
            "flow_rate":        {"offset": 4,  "min": 10.0, "max": 60.0,  "nominal": 30.0},
            "power":            {"offset": 6,  "min": 30.0, "max": 120.0, "nominal": 70.0},
            "pressure":         {"offset": 8,  "min": 0.2,  "max": 0.6,   "nominal": 0.4},
            "cop":              {"offset": 10, "min": 2.5,  "max": 8.0,   "nominal": 5.0},
            "cooling_capacity": {"offset": 12, "min": 200.0,"max": 800.0, "nominal": 450.0},
        },
        "name_template": "CDU-{:03d}",
        "base_addr": 400,
    },
}

class ModbusTCPSimulator:
    def __init__(self, host="0.0.0.0", port=5020, drift_interval=30,
                 num_chillers=8, num_cooling_towers=12,
                 num_precision_ac=80, num_cdu=20,
                 anomaly_rate=0.005):
        self.host = host
        self.port = port
        self.drift_interval = drift_interval
        self.anomaly_rate = anomaly_rate
        self.registers = {}
        self.device_info = {}
        self.anomaly_overrides = {}
        self.lock = threading.Lock()
        self.running = False

        self.device_counts = {
            "chiller": num_chillers,
            "cooling_tower": num_cooling_towers,
            "precision_ac": num_precision_ac,
            "cdu": num_cdu,
        }
        self._init_registers()

    def _init_registers(self):
        unit_id = 1
        for dtype, count in self.device_counts.items():
            cfg = DEVICE_CONFIGS[dtype]
            for i in range(count):
                uid = unit_id + i
                self.device_info[uid] = {
                    "name": cfg["name_template"].format(i + 1),
                    "type": dtype,
                    "base_addr": cfg["base_addr"] + i * 20,
                }
                for param_name, p in cfg["params"].items():
                    addr = cfg["base_addr"] + i * 20 + p["offset"]
                    self.registers[(uid, addr)] = int(p["nominal"] * 10)
            unit_id += count

    def _simulate_drift(self):
        for uid, info in self.device_info.items():
            dtype = info["type"]
            cfg = DEVICE_CONFIGS[dtype]
            for param_name, p in cfg["params"].items():
                addr = info["base_addr"] + p["offset"]
                current = self.registers.get((uid, addr), int(p["nominal"] * 10))
                drift = random.gauss(0, 0.02) * p["nominal"] * 10
                new_val = current + drift
                new_val = max(int(p["min"] * 10), min(int(p["max"] * 10), int(new_val)))
                if param_name == "cop" and random.random() < self.anomaly_rate:
                    if random.random() < 0.5:
                        new_val = int(random.uniform(p["min"], 3.5) * 10)
                    else:
                        new_val = int(random.uniform(6.5, p["max"]) * 10)
                self.registers[(uid, addr)] = new_val

        expired = [k for k, v in self.anomaly_overrides.items() if v["until"] and time.time() > v["until"]]
        for k in expired:
            del self.anomaly_overrides[k]
        for (uid, param), override in list(self.anomaly_overrides.items()):
            if override["until"] is None or time.time() <= override["until"]:
                info = self.device_info.get(uid)
                if info:
                    cfg = DEVICE_CONFIGS[info["type"]]
                    p = cfg["params"].get(param)
                    if p:
                        addr = info["base_addr"] + p["offset"]
                        self.registers[(uid, addr)] = int(override["value"] * 10)

    def inject_anomaly(self, device_type=None, unit_id=None, param="cop",
                       value=None, duration=None):
        targets = []
        if unit_id is not None:
            if unit_id in self.device_info:
                targets = [unit_id]
        elif device_type is not None:
            targets = [uid for uid, info in self.device_info.items()
                       if info["type"] == device_type]
        else:
            targets = list(self.device_info.keys())

        if not targets:
            return {"injected": 0}

        count = 0
        for uid in targets:
            info = self.device_info[uid]
            cfg = DEVICE_CONFIGS[info["type"]]
            if param not in cfg["params"]:
                continue
            p = cfg["params"][param]
            if value is None:
                if param == "cop":
                    value = random.uniform(p["min"], 3.0)
                else:
                    value = p["min"]
            until = time.time() + duration if duration else None
            self.anomaly_overrides[(uid, param)] = {"value": value, "until": until}
            addr = info["base_addr"] + p["offset"]
            self.registers[(uid, addr)] = int(value * 10)
            count += 1

        return {"injected": count, "param": param, "value": value,
                "duration": duration}

    def clear_anomalies(self):
        n = len(self.anomaly_overrides)
        self.anomaly_overrides.clear()
        self._init_registers()
        return {"cleared": n}

    def get_status(self):
        devices = []
        for uid, info in sorted(self.device_info.items()):
            cfg = DEVICE_CONFIGS[info["type"]]
            d = {"unit_id": uid, "name": info["name"], "type": info["type"]}
            for param_name, p in cfg["params"].items():
                addr = info["base_addr"] + p["offset"]
                d[param_name] = self.registers.get((uid, addr), 0) / 10.0
            devices.append(d)
        return {
            "total_devices": len(self.device_info),
            "device_counts": dict(self.device_counts),
            "drift_interval": self.drift_interval,
            "anomaly_rate": self.anomaly_rate,
            "active_anomalies": len(self.anomaly_overrides),
            "devices": devices,
        }

    def _read_holding_registers(self, unit_id, start_addr, count):
        results = []
        with self.lock:
            for i in range(count):
                addr = start_addr + i
                val = self.registers.get((unit_id, addr), 0)
                results.append(val)
        return results

    def _handle_connection(self, conn, addr):
        try:
            while self.running:
                data = conn.recv(1024)
                if not data:
                    break
                if len(data) < 7:
                    continue
                tx_id = struct.unpack(">H", data[0:2])[0]
                unit_id = data[6]
                func_code = data[7]
                if func_code == 0x03:
                    start_addr = struct.unpack(">H", data[8:10])[0]
                    count = struct.unpack(">H", data[10:12])[0]
                    values = self._read_holding_registers(unit_id, start_addr, count)
                    byte_count = count * 2
                    resp = struct.pack(">H", tx_id)
                    resp += struct.pack(">H", 0)
                    resp += struct.pack(">H", 3 + byte_count)
                    resp += struct.pack("B", unit_id)
                    resp += struct.pack("B", func_code)
                    resp += struct.pack("B", byte_count)
                    for v in values:
                        resp += struct.pack(">H", v & 0xFFFF)
                    conn.sendall(resp)
        except Exception:
            pass
        finally:
            conn.close()

    def _drift_thread(self):
        while self.running:
            with self.lock:
                self._simulate_drift()
            time.sleep(self.drift_interval)

    def start(self):
        self.running = True
        server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        server.bind((self.host, self.port))
        server.listen(50)
        server.settimeout(1.0)
        print(f"Modbus TCP Simulator listening on {self.host}:{self.port}")
        print(f"  Devices: {dict(self.device_counts)}")
        print(f"  Drift interval: {self.drift_interval}s")
        print(f"  Anomaly rate: {self.anomaly_rate}")

        threading.Thread(target=self._drift_thread, daemon=True).start()

        try:
            while self.running:
                try:
                    conn, addr = server.accept()
                    threading.Thread(target=self._handle_connection,
                                    args=(conn, addr), daemon=True).start()
                except socket.timeout:
                    continue
        except KeyboardInterrupt:
            pass
        finally:
            self.running = False
            server.close()
            print("Simulator stopped")


sim_ref = None

class ControlHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/status":
            self._json_response(sim_ref.get_status())
        else:
            self._json_response({"error": "not found"}, 404)

    def do_POST(self):
        content_length = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(content_length) if content_length > 0 else b"{}"
        try:
            data = json.loads(body)
        except json.JSONDecodeError:
            self._json_response({"error": "invalid json"}, 400)
            return

        if self.path == "/inject":
            result = sim_ref.inject_anomaly(
                device_type=data.get("device_type"),
                unit_id=data.get("unit_id"),
                param=data.get("param", "cop"),
                value=data.get("value"),
                duration=data.get("duration"),
            )
            self._json_response(result)
        elif self.path == "/clear":
            result = sim_ref.clear_anomalies()
            self._json_response(result)
        else:
            self._json_response({"error": "not found"}, 404)

    def _json_response(self, data, code=200):
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(data, default=str).encode())

    def log_message(self, format, *args):
        pass


def main():
    parser = argparse.ArgumentParser(description="Modbus TCP Simulator for DC Cooling")
    parser.add_argument("--host", default="0.0.0.0", help="Modbus listen host")
    parser.add_argument("--port", type=int, default=5020, help="Modbus listen port")
    parser.add_argument("--control-port", type=int, default=8081,
                        help="HTTP control API port")
    parser.add_argument("--drift-interval", type=int, default=30,
                        help="Data drift interval in seconds")
    parser.add_argument("--chillers", type=int, default=8,
                        help="Number of chiller units")
    parser.add_argument("--cooling-towers", type=int, default=12,
                        help="Number of cooling towers")
    parser.add_argument("--precision-ac", type=int, default=80,
                        help="Number of precision AC units")
    parser.add_argument("--cdu", type=int, default=20,
                        help="Number of CDU units")
    parser.add_argument("--anomaly-rate", type=float, default=0.005,
                        help="Probability of random COP anomaly per drift cycle")
    args = parser.parse_args()

    global sim_ref
    sim_ref = ModbusTCPSimulator(
        host=args.host,
        port=args.port,
        drift_interval=args.drift_interval,
        num_chillers=args.chillers,
        num_cooling_towers=args.cooling_towers,
        num_precision_ac=args.precision_ac,
        num_cdu=args.cdu,
        anomaly_rate=args.anomaly_rate,
    )

    sim_thread = threading.Thread(target=sim_ref.start, daemon=True)
    sim_thread.start()
    time.sleep(0.5)

    control_server = HTTPServer((args.host, args.control_port), ControlHandler)
    print(f"Control API listening on {args.host}:{args.control_port}")
    print(f"  GET  /status           - Get all device status")
    print(f"  POST /inject           - Inject anomaly")
    print(f"  POST /clear            - Clear all anomalies")
    print(f"  Inject example: curl -X POST http://localhost:{args.control_port}/inject "
          f"-H 'Content-Type: application/json' -d "
          f"'{{\"device_type\":\"chiller\",\"param\":\"cop\",\"value\":2.5,\"duration\":300}}'")

    try:
        control_server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        sim_ref.running = False
        control_server.server_close()
        print("Simulator stopped")


if __name__ == "__main__":
    main()
