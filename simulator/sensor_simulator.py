#!/usr/bin/env python3
"""
三弓床弩传感器模拟器
模拟通过UDP上报弓弦拉力、弩臂变形、箭矢初速、穿甲深度
"""

import socket
import json
import time
import random
import math
import argparse
from datetime import datetime, timezone


class BallisticsSimulator:
    def __init__(self, host="127.0.0.1", port=8080, device_id="chuangnu-001", interval=60):
        self.host = host
        self.port = port
        self.device_id = device_id
        self.interval = interval
        self.sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)

        self.base_tension = 4500.0
        self.base_deformation = 8.0
        self.base_velocity = 120.0
        self.base_penetration = 0.002

        self.shot_count = 0

    def simulate_shot(self):
        self.shot_count += 1

        wear_factor = 1.0 + self.shot_count * 0.0005

        tension = self.base_tension + random.gauss(0, 300) * wear_factor
        deformation = self.base_deformation + abs(random.gauss(0, 1.5)) * wear_factor
        velocity = self.base_velocity + random.gauss(0, 8)
        penetration = self.base_penetration * (velocity / self.base_velocity) ** 2 + random.gauss(0, 0.0003)

        if self.shot_count % 50 == 0:
            deformation += random.uniform(5, 12)
            velocity *= random.uniform(0.6, 0.8)

        if self.shot_count % 100 == 0:
            velocity *= random.uniform(0.5, 0.7)

        if deformation < 0:
            deformation = 0.1
        if velocity < 10:
            velocity = 10
        if penetration < 0:
            penetration = 0.0001

        data = {
            "device_id": self.device_id,
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "bowstring_tension": round(tension, 2),
            "arm_deformation": round(deformation, 3),
            "arrow_initial_velocity": round(velocity, 2),
            "penetration_depth": round(penetration, 6),
            "temperature": round(20.0 + random.gauss(0, 3), 1),
            "humidity": round(50.0 + random.gauss(0, 10), 1)
        }
        return data

    def send_data(self, data):
        payload = json.dumps(data, ensure_ascii=False).encode("utf-8")
        try:
            self.sock.sendto(payload, (self.host, self.port))
            print(f"[{data['timestamp']}] Shot #{self.shot_count}: "
                  f"拉力={data['bowstring_tension']:.0f}N, "
                  f"变形={data['arm_deformation']:.2f}mm, "
                  f"初速={data['arrow_initial_velocity']:.1f}m/s, "
                  f"穿深={data['penetration_depth']*1000:.2f}mm")
            return True
        except Exception as e:
            print(f"发送失败: {e}")
            return False

    def run(self):
        print(f"三弓床弩传感器模拟器启动")
        print(f"目标: {self.host}:{self.port}")
        print(f"设备ID: {self.device_id}")
        print(f"上报间隔: {self.interval}秒")
        print("=" * 60)

        try:
            while True:
                data = self.simulate_shot()
                self.send_data(data)
                time.sleep(self.interval)
        except KeyboardInterrupt:
            print("\n模拟器已停止")
            self.sock.close()


def main():
    parser = argparse.ArgumentParser(description="三弓床弩传感器模拟器")
    parser.add_argument("--host", default="127.0.0.1", help="UDP目标主机")
    parser.add_argument("--port", type=int, default=8080, help="UDP目标端口")
    parser.add_argument("--device", default="chuangnu-001", help="设备ID")
    parser.add_argument("--interval", type=int, default=60, help="上报间隔(秒)")
    parser.add_argument("--once", action="store_true", help="只发送一次")
    parser.add_argument("--count", type=int, default=0, help="发送指定次数后停止")

    args = parser.parse_args()

    sim = BallisticsSimulator(args.host, args.port, args.device, args.interval)

    if args.once:
        data = sim.simulate_shot()
        sim.send_data(data)
        return

    if args.count > 0:
        for _ in range(args.count):
            data = sim.simulate_shot()
            sim.send_data(data)
            time.sleep(args.interval)
        return

    sim.run()


if __name__ == "__main__":
    main()
