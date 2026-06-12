import sys
import json

print("[Python] Pipe memory stream listener activated.")

# Continuous loop waiting for real-time memory packets via standard input
for line in sys.stdin:
    try:
        packet = json.loads(line.strip())

        space_name = packet.get("memory_name")
        memory_data = packet.get("data")

        print(f"[Python] Target Memory Space: '{space_name}'")
        print(f"Value of test1: {memory_data.get('test1')}")
        print(f"Value of counter: {memory_data.get('counter')}")
    except Exception as e:
        print(f"[Python Error]: {e}")
        break