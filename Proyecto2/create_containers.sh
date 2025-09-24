#!/bin/bash
IMAGES=("ram_hog" "cpu_hog" "low_consumption")
CURRENT=$(docker ps -q | wc -l)  # Contenedores activos
MAX_CONTAINERS=15
NEEDED=$((MAX_CONTAINERS - CURRENT))

if [ $NEEDED -gt 0 ]; then
    # Crear hasta un máximo de 10 nuevos contenedores si es necesario
    for i in $(seq 1 $(($NEEDED > 10 ? 10 : $NEEDED))); do
        IMAGE=${IMAGES[$RANDOM % ${#IMAGES[@]}]}
        docker run -d --name "container-$i-$(date +%s)" $IMAGE > /dev/null 2>&1
    done
fi