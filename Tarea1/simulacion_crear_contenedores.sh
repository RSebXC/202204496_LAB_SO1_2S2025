#!/bin/bash
NUM_FILES=$((RANDOM % 4 + 1))
CARNET="202204496"
# Array de nombres aleatorios para los archivos
NAMES=("alpha" "beta" "gamma" "delta" "epsilon" "zeta")
# Crear los archivos
for ((i=1; i<=NUM_FILES; i++))
do
    # Seleccionar un nombre aleatorio del array
    RANDOM_NAME=${NAMES[$((RANDOM % ${#NAMES[@]}))]}
    # Generar el nombre del archivo
    FILENAME="contenedor_${CARNET}_${RANDOM_NAME}.txt"
    # Crear el archivo y escribir su nombre como contenido
    echo "$FILENAME" > "$FILENAME"
    echo "Archivo creado: $FILENAME"
done
echo "Se crearon $NUM_FILES archivos."
