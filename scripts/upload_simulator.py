import argparse
import csv
import io
import os
import random
import tempfile
import uuid

from PIL import Image
import requests

SCENE_TYPES = ["highway", "urban", "parking", "suburban", "rural"]

SCENE_NAMES = {
    "highway": "Highway Driving Scenario",
    "urban": "Urban Intersection Scenario",
    "parking": "Parking Lot Navigation Scenario",
    "suburban": "Suburban Road Scenario",
    "rural": "Rural Road Scenario",
}

SCENE_DESCRIPTIONS = {
    "highway": "High-speed highway driving with lane changes and merging traffic",
    "urban": "Complex urban intersection with pedestrians and traffic signals",
    "parking": "Low-speed parking lot navigation around static and dynamic obstacles",
    "suburban": "Residential area driving with moderate traffic and speed bumps",
    "rural": "Narrow rural road with limited visibility and occasional oncoming traffic",
}

FRAME_COLORS = {
    "highway": (100, 150, 200),
    "urban": (180, 140, 100),
    "parking": (140, 160, 120),
    "suburban": (160, 180, 130),
    "rural": (120, 140, 90),
}

FRAME_RATE = 10
DURATION = 3.0
FRAME_COUNT = 30
CAN_RECORD_COUNT = 30


def generate_frame_image(scene_type, frame_index, output_dir):
    base_color = FRAME_COLORS.get(scene_type, (128, 128, 128))
    variation = int(frame_index * 2)
    color = tuple(min(255, max(0, c + variation - 30)) for c in base_color)
    img = Image.new("RGB", (640, 480), color)
    road_color = (80, 80, 80)
    img.paste(road_color, [160, 0, 320, 480])
    line_color = (255, 255, 255)
    y_pos = 240 + (frame_index * 5) % 100 - 50
    img.paste(line_color, [280, y_pos, 282, y_pos + 30])
    img.paste(line_color, [358, y_pos, 360, y_pos + 30])
    filename = f"frame_{frame_index:04d}.png"
    filepath = os.path.join(output_dir, filename)
    img.save(filepath, "PNG")
    return filepath


def generate_can_log(scene_type, record_count, output_dir):
    base_speed = {"highway": 110, "urban": 40, "parking": 10, "suburban": 50, "rural": 60}
    speed = base_speed.get(scene_type, 50)
    filename = "can_log.csv"
    filepath = os.path.join(output_dir, filename)
    with open(filepath, "w", newline="") as f:
        writer = csv.writer(f)
        writer.writerow(["timestamp", "speed", "steering_angle", "throttle", "brake", "gear"])
        for i in range(record_count):
            ts = round(i / FRAME_RATE, 3)
            spd = round(speed + random.uniform(-5, 5), 2)
            steer = round(random.uniform(-15, 15), 2)
            throttle = round(random.uniform(0.1, 0.8), 3)
            brake = round(random.uniform(0.0, 0.3), 3)
            gear = "D" if spd > 5 else "P"
            writer.writerow([ts, spd, steer, throttle, brake, gear])
    return filepath


def create_zip_from_frames(frame_dir, can_log_path, output_path):
    import zipfile

    with zipfile.ZipFile(output_path, "w", zipfile.ZIP_DEFLATED) as zf:
        for fname in sorted(os.listdir(frame_dir)):
            fpath = os.path.join(frame_dir, fname)
            zf.write(fpath, f"frames/{fname}")
        zf.write(can_log_path, "can_log.csv")
    return output_path


def upload_scenario(api_url, scene_type, index):
    scene_id = str(uuid.uuid4())
    scene_name = f"{SCENE_NAMES[scene_type]} #{index + 1}"
    description = SCENE_DESCRIPTIONS[scene_type]

    with tempfile.TemporaryDirectory() as tmpdir:
        frame_dir = os.path.join(tmpdir, "frames")
        os.makedirs(frame_dir)

        print(f"[Scenario {index + 1}] Generating {FRAME_COUNT} frames for '{scene_name}'...")
        for fi in range(FRAME_COUNT):
            generate_frame_image(scene_type, fi, frame_dir)
        print(f"[Scenario {index + 1}] Frames generated.")

        print(f"[Scenario {index + 1}] Generating CAN bus log with {CAN_RECORD_COUNT} records...")
        can_log_path = generate_can_log(scene_type, CAN_RECORD_COUNT, tmpdir)
        print(f"[Scenario {index + 1}] CAN log generated.")

        zip_path = os.path.join(tmpdir, f"scenario_{scene_id}.zip")
        print(f"[Scenario {index + 1}] Packaging scenario data...")
        create_zip_from_frames(frame_dir, can_log_path, zip_path)

        zip_size = os.path.getsize(zip_path)
        print(f"[Scenario {index + 1}] Package size: {zip_size / 1024:.1f} KB")

        print(f"[Scenario {index + 1}] Uploading to {api_url}/api/scenes/upload ...")
        try:
            with open(zip_path, "rb") as zf:
                files = {"file": (f"scenario_{scene_id}.zip", zf, "application/zip")}
                data = {
                    "name": scene_name,
                    "description": description,
                    "scene_type": scene_type,
                    "duration": str(DURATION),
                    "frame_count": str(FRAME_COUNT),
                    "frame_rate": str(FRAME_RATE),
                }
                response = requests.post(f"{api_url}/api/scenes/upload", files=files, data=data, timeout=60)

            if response.status_code in (200, 201):
                print(f"[Scenario {index + 1}] Upload successful! Response: {response.json()}")
            else:
                print(f"[Scenario {index + 1}] Upload failed! Status: {response.status_code}, Body: {response.text}")
        except requests.exceptions.ConnectionError:
            print(f"[Scenario {index + 1}] Connection error: Could not reach {api_url}")
        except requests.exceptions.Timeout:
            print(f"[Scenario {index + 1}] Upload timed out.")
        except Exception as e:
            print(f"[Scenario {index + 1}] Upload error: {e}")


def main():
    parser = argparse.ArgumentParser(description="Upload simulated driving scenarios")
    parser.add_argument("--api-url", default="http://localhost:8080", help="Backend API base URL")
    parser.add_argument("--count", type=int, default=3, help="Number of scenarios to upload")
    args = parser.parse_args()

    print(f"=== Autonomous Driving Scenario Upload Simulator ===")
    print(f"API URL: {args.api_url}")
    print(f"Scenarios to upload: {args.count}")
    print(f"Frames per scenario: {FRAME_COUNT} @ {FRAME_RATE}fps ({DURATION}s)")
    print(f"CAN records per scenario: {CAN_RECORD_COUNT}")
    print(f"====================================================\n")

    selected_types = SCENE_TYPES[: args.count] if args.count <= len(SCENE_TYPES) else [random.choice(SCENE_TYPES) for _ in range(args.count)]

    for i, scene_type in enumerate(selected_types):
        upload_scenario(args.api_url, scene_type, i)
        print()

    print("=== All scenarios processed ===")


if __name__ == "__main__":
    main()
