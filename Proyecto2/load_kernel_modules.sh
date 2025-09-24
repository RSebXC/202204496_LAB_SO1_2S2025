#!/bin/bash
BASE_DIR=$(dirname "$0")

echo "Descargando módulos existentes..."
sudo rmmod continfo 2>/dev/null || true
sudo rmmod sysinfo 2>/dev/null || true

echo "Cargando módulo continfo..."
sudo insmod "$BASE_DIR/kernel_continfo/continfo.ko" 2>&1 || { echo "Error cargando continfo"; exit 1; }

echo "Cargando módulo sysinfo..."
sudo insmod "$BASE_DIR/kernel_sysinfo/sysinfo.ko" 2>&1 || { echo "Error cargando sysinfo"; exit 1; }

echo "Verificando archivos /proc..."
ls /proc/continfo_so1_202204496 2>/dev/null && echo "✅ /proc/continfo OK"
ls /proc/sysinfo_so1_202204496 2>/dev/null && echo "✅ /proc/sysinfo OK"

echo "✅ Módulos cargados"