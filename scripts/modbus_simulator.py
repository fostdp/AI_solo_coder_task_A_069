import struct
import socket
import threading
import time
import random
import math

DEVICE_MAP = {
    1:  {"name": "CHU-001", "type": "chiller",       "count": 8,  "base_addr": 0},
    9:  {"name": "CT-001",  "type": "cooling_tower",  "count": 12, "base_addr": 100},
    21: {"name": "PAC-001", "type": "precision_ac",   "count": 80, "base_addr": 200},
    101:{"name": "CDU-001", "type": "cdu",            "count": 20, "base_addr": 400},
}

CHILLER_PARAMS = {
    "supply_temp":     {"offset": 0,  "min": 5.0,  "max": 12.0,  "nominal": 7.0},
    "return_temp":     {"offset": 2,  "min": 10.0, "max": 18.0,  "nominal": 12.0},
    "flow_rate":       {"offset": 4,  "min": 50.0, "max": 200.0, "nominal": 120.0},
    "power":           {"offset": 6,  "min": 200.0,"max": 600.0, "nominal": 350.0},
    "pressure":        {"offset": 8,  "min": 0.2,  "max": 0.8,   "nominal": 0.5},
    "cop":             {"offset": 10, "min": 2.0,  "max": 8.0,   "nominal": 5.5},
    "cooling_capacity":{"offset": 12, "min": 800.0,"max": 2500.0,"nominal": 1800.0},
}

COOLING_TOWER_PARAMS = {
    "supply_temp":     {"offset": 0,  "min": 25.0, "max": 35.0,  "nominal": 28.0},
    "return_temp":     {"offset": 2,  "min": 30.0, "max": 42.0,  "nominal": 35.0},
    "flow_rate":       {"offset": 4,  "min": 80.0, "max": 250.0, "nominal": 150.0},
    "power":           {"offset": 6,  "min": 30.0, "max": 100.0, "nominal": 60.0},
    "pressure":        {"offset": 8,  "min": 0.1,  "max": 0.5,   "nominal": 0.3},
    "cop":             {"offset": 10, "min": 3.0,  "max": 10.0,  "nominal": 7.0},
    "cooling_capacity":{"offset": 12, "min": 500.0,"max": 2000.0,"nominal": 1200.0},
}

PRECISION_AC_PARAMS = {
    "supply_temp":     {"offset": 0,  "min": 15.0, "max": 25.0,  "nominal": 20.0},
    "return_temp":     {"offset": 2,  "min": 25.0, "max": 40.0,  "nominal": 32.0},
    "flow_rate":       {"offset": 4,  "min": 5.0,  "max": 30.0,  "nominal": 15.0},
    "power":           {"offset": 6,  "min": 20.0, "max": 80.0,  "nominal": 45.0},
    "pressure":        {"offset": 8,  "min": 0.1,  "max": 0.4,   "nominal": 0.2},
    "cop":             {"offset": 10, "min": 1.5,  "max": 6.0,   "nominal": 3.5},
    "cooling_capacity":{"offset": 12, "min": 50.0, "max": 200.0, "nominal": 130.0},
}

CDU_PARAMS = {
    "supply_temp":     {"offset": 0,  "min": 14.0, "max": 22.0,  "nominal": 18.0},
    "return_temp":     {"offset": 2,  "min": 22.0, "max": 35.0,  "nominal": 28.0},
    "flow_rate":       {"offset": 4,  "min": 10.0, "max": 60.0,  "nominal": 30.0},
    "power":           {"offset": 6,  "min": 30.0, "max": 120.0, "nominal": 70.0},
    "pressure":        {"offset": 8,  "min": 0.2,  "max": 0.6,   "nominal": 0.4},
    "cop":             {"offset": 10, "min": 2.5,  "max": 8.0,   "nominal": 5.0},
    "cooling_capacity":{"offset": 12, "min": 200.0,"max": 800.0, "nominal": 450.0},
}

TYPE_PARAMS = {
    "chiller":       CHILLER_PARAMS,
    "cooling_tower": COOLING_TOWER_PARAMS,
    "precision_ac":  PRECISION_AC_PARAMS,
    "cdu":           CDU_PARAMS,
}

class ModbusTCPSimulator:
    def __init__(self, host="0.0.0.0", port=5020):
        self.host = host
        self.port = port
        self.registers = {}
        self.lock = threading.Lock()
        self._init_registers()
        self.running = False

    def _init_registers(self):
        for start_id, info in DEVICE_MAP.items():
            params = TYPE_PARAMS[info["type"]]
            for i in range(info["count"]):
                unit_id = start_id + i
                for param_name, p in params.items():
                    addr = info["base_addr"] + i * 20 + p["offset"]
                    val = p["nominal"] * 10
                    self.registers[(unit_id, addr)] = int(val)

    def _simulate_drift(self):
        for start_id, info in DEVICE_MAP.items():
            params = TYPE_PARAMS[info["type"]]
            for i in range(info["count"]):
                unit_id = start_id + i
                for param_name, p in params.items():
                    addr = info["base_addr"] + i * 20 + p["offset"]
                    current = self.registers.get((unit_id, addr), int(p["nominal"] * 10))
                    drift = random.gauss(0, 0.02) * p["nominal"] * 10
                    new_val = current + drift
                    new_val = max(int(p["min"] * 10), min(int(p["max"] * 10), int(new_val)))
                    if param_name == "cop" and random.random() < 0.005:
                        if random.random() < 0.5:
                            new_val = int(random.uniform(p["min"], 3.5) * 10)
                        else:
                            new_val = int(random.uniform(6.5, p["max"]) * 10)
                    self.registers[(unit_id, addr)] = new_val

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
            time.sleep(30)

    def start(self):
        self.running = True
        server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        server.bind((self.host, self.port))
        server.listen(50)
        server.settimeout(1.0)
        print(f"Modbus TCP Simulator listening on {self.host}:{self.port}")

        threading.Thread(target=self._drift_thread, daemon=True).start()

        try:
            while self.running:
                try:
                    conn, addr = server.accept()
                    threading.Thread(target=self._handle_connection, args=(conn, addr), daemon=True).start()
                except socket.timeout:
                    continue
        except KeyboardInterrupt:
            pass
        finally:
            self.running = False
            server.close()
            print("Simulator stopped")

if __name__ == "__main__":
    sim = ModbusTCPSimulator(host="0.0.0.0", port=5020)
    sim.start()
