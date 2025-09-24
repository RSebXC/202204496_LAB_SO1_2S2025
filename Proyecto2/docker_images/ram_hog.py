# ram_hog.py
import time
data = []
while True:
    data.append([0] * 10_000_000)  # Consume ~80MB por iteración
    time.sleep(1)  # Lento para no saturar de inmediato