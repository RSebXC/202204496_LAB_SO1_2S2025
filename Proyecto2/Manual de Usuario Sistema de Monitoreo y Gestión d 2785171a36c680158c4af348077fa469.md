# Manual de Usuario: Sistema de Monitoreo y Gestión de Contenedores

Este manual de usuario proporciona instrucciones claras y prácticas para utilizar el sistema de monitoreo y gestión de contenedores. El sistema permite supervisar métricas de procesos y contenedores en tiempo real, tomar decisiones autónomas (como eliminar contenedores excedentes), almacenar datos en una base de datos SQLite y visualizarlos en un dashboard de Grafana. El usuario final puede interactuar principalmente con el daemon en Go para iniciar/detener el servicio y acceder a Grafana para ver las métricas.

El sistema está diseñado para ser fácil de usar, con un enfoque en automatización: una vez iniciado, el daemon maneja todo el monitoreo y las acciones, mientras el usuario observa los resultados en Grafana.

---

## Instrucciones claras para el usuario final

El sistema se opera principalmente a través del daemon en Go, que se ejecuta en la terminal. Sigue estos pasos básicos para usarlo:

1. **Iniciar el sistema**:
    - Abre una terminal y navega a la carpeta del daemon (e.g., `/home/rodrigo-prime/Documentos/SOPES1/Proyecto2/daemon_go/cmd/daemon`).
    - Compila el daemon si es necesario: `go build`.
    - Ejecuta el daemon: `./daemon`.
        - Esto cargará los módulos del kernel, iniciará Grafana, configurará el cronjob para simular carga (creando contenedores cada 2 minutos), y comenzará el monitoreo cada 20 segundos.
    - Verás logs en la terminal como "✅ Módulos del kernel cargados", "✅ Grafana iniciado", y resúmenes de cada ciclo (e.g., "🏆 TOP 3 RAM").
2. **Monitorear el sistema**:
    - Mientras el daemon corre, observa los logs en la terminal para ver contenedores activos, métricas de RAM/CPU, y decisiones tomadas (e.g., "🎯 Tomando 2 decisiones...").
    - Accede a Grafana para visuales (ver sección "Cómo acceder al Dashboard de Grafana").
3. **Detener el sistema**:
    - Presiona `Ctrl + C` en la terminal del daemon.
        - Verás "🛑 Recibida señal de terminación..." seguido de "🧹 Limpiando recursos...".
        - Esto eliminará el cronjob, descargará los módulos del kernel y detendrá Grafana automáticamente.
4. **Ver logs adicionales**:
    - Logs del cronjob: `cat /tmp/container_cron.log` (muestra creación de contenedores).
    - Logs del kernel: `dmesg | tail` (muestra carga/descarga de módulos).

**[screen: Insertar captura de pantalla de la terminal mostrando el inicio del daemon con logs como "✅ Grafana iniciado" y un ciclo de análisis completo]**

**[screen: Insertar captura de pantalla de la terminal mostrando el shutdown con "✅ Limpieza completada"]**

### Ejemplos de uso práctico

- **Ejemplo 1: Monitoreo básico de carga**:
    - Inicia el daemon: `./daemon`.
    - Espera 2 minutos para que el cronjob cree contenedores (verás ~5 contenedores activos en los logs).
    - Observa en los logs: "🐳 Contenedores activos: 5" y TOP3 RAM/CPU.
    - Accede a Grafana: Ve el dashboard para gráficos como "Uso de RAM" (línea ascendente si hay carga) y "Top 5 Contenedores por RAM" (pie chart con nombres como "grafana" ~262MB).
- **Ejemplo 2: Simulación de sobrecarga**:
    - Inicia el daemon.
    - Manualmente crea contenedores extra: `docker run -d --name test_high busybox sh -c "while true; do :; done"` (alto CPU).
    - En el siguiente ciclo (20s), el daemon detectará excedentes (>5) y eliminará automáticamente, loggeando "🎯 Tomando 1 decisiones..." y "🗑️ Contenedor eliminado".
    - Ver en Grafana: "Contenedores Eliminados" aumenta en el gráfico de líneas.
- **Ejemplo 3: Ver métricas del sistema**:
    - Mientras el daemon corre, ejecuta `cat /proc/sysinfo_so1_202204496` manualmente para ver JSON con procesos del sistema (incluyendo estado).
    - Usa el dashboard de Grafana para ver "Total RAM: 15.0GB" y "RAM Usada: 6.1GB".

**[screen: Insertar captura de pantalla de un ejemplo de uso en terminal mostrando creación manual de un contenedor y su eliminación automática en el log del daemon]**

**[screen: Insertar captura de pantalla de Grafana mostrando el dashboard con un ejemplo de "Top 5 Contenedores por RAM"]**

---

## Guía de Instalación

### Requisitos del sistema

- **Hardware**: CPU multi-core (al menos 4 núcleos), 16GB RAM (para manejar contenedores), 50GB de almacenamiento.
- **Software**:
    - Sistema operativo: Ubuntu 22.04 LTS o similar (Linux kernel 5.15+).
    - Go 1.20+ (instala con `sudo apt install golang-go`).
    - Docker 24+ (instala con `sudo apt install docker.io` y habilita con `sudo systemctl enable --now docker`).
    - SQLite3 (instalado por default en Ubuntu; verifica con `sqlite3 --version`).
    - Headers del kernel: `sudo apt install linux-headers-$(uname -r) build-essential`.
    - Herramientas adicionales: `make`, `jq` (para depuración: `sudo apt install jq`).
- **Acceso**: Privilegios root/sudo para `insmod`, `docker`, y `crontab`.

### Pasos detallados de instalación y configuración

1. **Clonar o descargar el proyecto**:
    - `git clone <repo_url>` o descarga el ZIP en `/home/rodrigo-prime/Documentos/SOPES1/Proyecto2`.
2. **Compilar módulos de kernel**:
    - Navega a `kernel_continfo/`: `cd kernel_continfo`.
    - Ejecuta `make`.
    - Repite en `sysinfo/`: `cd ../sysinfo; make`.
    - Verifica: Deben generarse `continfo.ko` y `sysinfo.ko`.
3. **Construir imágenes Docker**:
    - Navega a `docker_images/`: `cd docker_images`.
    - Construye `ram_hog`: `docker build -f Dockerfile.ram -t ram_hog .`.
    - Construye `cpu_hog`: `docker build -f Dockerfile.cpu -t cpu_hog .`.
    - Construye `low_consumption`: `docker build -f Dockerfile.low -t low_consumption .`.
    - Verifica: `docker images | grep hog` (debería listar las 3).
4. **Compilar el daemon en Go**:
    - Navega a `daemon_go/cmd/daemon/`: `cd daemon_go/cmd/daemon`.
    - Ejecuta `go mod tidy` (para dependencias).
    - Compila: `go build -o daemon`.
5. **Configurar y ejecutar**:
    - Haz ejecutable el script: `chmod +x ../create_containers.sh`.
    - Ejecuta el daemon: `./daemon`.
    - Accede a Grafana: Abre un navegador en `http://localhost:3000` (user: admin, pass: admin).
6. **Verificación de instalación**:
    - `docker ps`: Debería mostrar Grafana y 5 contenedores de prueba.
    - `lsmod | grep so1`: Debería mostrar `continfo` y `sysinfo` cargados.
    - `crontab -l`: Debería mostrar la entry del cronjob.

**[screen: Insertar captura de pantalla de los pasos de compilación de un módulo kernel con `make`]**

**[screen: Insertar captura de pantalla de la construcción de una imagen Docker con `docker build`]**

---

## Diagramas y Arquitectura

### Descripción de la arquitectura del sistema para usuarios no técnicos

El sistema es como un "vigilante inteligente" para contenedores en tu computadora: monitorea el uso de recursos (como memoria y CPU), mantiene un número ideal de contenedores (3 "ligeros" y 2 "pesados"), y elimina automáticamente los extras para evitar sobrecarga. Todo se ejecuta en segundo plano, con datos visuales en un dashboard fácil de ver.

- **Componentes principales**:
    - **Módulos de kernel**: Como sensores internos que capturan datos reales del sistema (e.g., uso de RAM).
    - **Daemon en Go**: El "cerebro" que analiza datos, decide eliminar/crear contenedores, y guarda información.
    - **Cronjob**: Un temporizador que crea contenedores de prueba cada 2 minutos para simular uso.
    - **SQLite**: Una base de datos simple para almacenar métricas.
    - **Grafana**: Un tablero visual donde ves gráficos (e.g., "Top 5 Contenedores por RAM").

El flujo es: Kernel captura datos → Daemon analiza y actúa → Datos a SQLite → Grafana muestra gráficos.

### Diagramas de flujo o estructura

- **Diagrama de flujo principal (texto ASCII)**:
    
    ```
    Inicio del Daemon
    |
    v
    Cargar Módulos Kernel --> Leer /proc (Métricas RAM/CPU)
    |
    v
    Iniciar Grafana y Cronjob (Crea contenedores cada 2 min)
    |
    v
    Loop cada 20s:
      - Lista Contenedores
      - Analiza (Contar bajos/altos, %RAM)
      - Decide (Eliminar >5 o ajustar a 3 bajos/2 altos)
      - Ejecuta (stop_and_remove / create_high/low)
      - Guarda en SQLite
    |
    v
    Shutdown (Ctrl+C): Eliminar Cronjob, Descargar Kernel, Detener Grafana
    
    ```
    

**[screen: Insertar diagrama visual de flujo (e.g., generado con [draw.io](http://draw.io/) o similar, mostrando flechas entre Kernel → Daemon → SQLite → Grafana)]**

**[screen: Insertar diagrama de arquitectura (e.g., componentes como cajas: Kernel Modules → Go Daemon → SQLite DB → Grafana Dashboard)]**

---

## Cómo acceder al Dashboard de Grafana

Grafana es el "tablero de control" donde ves las métricas en gráficos fáciles de entender. Una vez iniciado el daemon, Grafana se levanta automáticamente.

### Pasos para acceder

1. **Requisitos**: Navegador web (Chrome/Firefox), puerto 3000 libre.
2. **Inicia el daemon**: Ejecuta `./daemon` en la terminal (inicia Grafana automáticamente).
3. **Accede al dashboard**:
    - Abre un navegador y ve a `http://localhost:3000`.
    - Usuario: `admin`, Contraseña: `admin` (cámbiala en la primera entrada por seguridad).
4. **Navega el dashboard**:
    - En el menú lateral, selecciona "Dashboards" > "Browse" > selecciona el dashboard principal (e.g., "Container Manager Dashboard").
    - Verás paneles como "Total RAM" (15.0 GB), "Uso de RAM" (gráfica de línea), "Top 5 Contenedores por RAM" (pie chart con nombres y MB), "Top 5 Contenedores por CPU" (pie chart con %CPU), "Contenedores Eliminados" (línea de tiempo), "Contenedores Activos" (número actual ~5), "RAM Usada/Libre" (valores en GB).
5. **Interacción**: Ajusta el rango de tiempo (arriba a la derecha, e.g., "Last 1 hour") para zoom. Refresh automático cada 10s.

**[screen: Insertar captura de pantalla del login de Grafana en `http://localhost:3000`]**

**[screen: Insertar captura de pantalla del dashboard completo en Grafana, destacando paneles como "Top 5 Contenedores por RAM"]**

### Consejos para uso

- Si Grafana no responde: Verifica `docker ps | grep grafana` (debe estar "Up"). Reinicia el daemon si es necesario.
- Personalización: En Grafana, edita paneles para agregar filtros o alertas (e.g., alerta si RAM usada >80%).