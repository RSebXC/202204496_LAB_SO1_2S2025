# cpu_hog.py
while True:
    sum = 0
    for i in range(1000000):
        sum += i ** 2  # Cálculo intensivo