FROM ghcr.io/project-zot/zot-minimal-linux-amd64:latest

# Variable de entorno personalizada para identificar esta VM
ENV ZOT_NAME=zot-vm3

# Cuando arranque el contenedor, mostrará el nombre al hacer curl
CMD ["/zot-linux-amd64", "serve", "/etc/zot/config.json"]
