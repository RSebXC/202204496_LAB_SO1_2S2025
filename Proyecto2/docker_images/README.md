# Docker Images Documentation

## ram_hog
- **Descripción**: Simula alto consumo de RAM.
- **Dockerfile**: Dockerfile.ram
- **Comando de construcción**: `docker build -t ram_hog .`
- **Comportamiento**: Ejecuta `dd if=/dev/zero of=/dev/null bs=1M count=100` en bucle.

## cpu_hog
- **Descripción**: Simula alto consumo de CPU.
- **Dockerfile**: Dockerfile.cpu
- **Comando de construcción**: `docker build -t cpu_hog .`
- **Comportamiento**: Ejecuta `while true; do :; done`.

## low_consumption
- **Descripción**: Simula bajo consumo.
- **Dockerfile**: Dockerfile.low
- **Comando de construcción**: `docker build -t low_consumption .`
- **Comportamiento**: Ejecuta `sleep infinity`.