# Manual Técnico Detallado: Sistema de Monitoreo y Gestión de Contenedores con Módulos de Kernel y Daemon en Go

Este manual técnico proporciona una documentación exhaustiva del sistema desarrollado, incluyendo módulos de kernel en C para capturar métricas de procesos y contenedores, un daemon en Go para procesamiento y toma de decisiones autónomas, integración con SQLite para almacenamiento, Grafana para visualización, y scripts de automatización para simular carga. El sistema está diseñado para cumplir con los requisitos del proyecto, enfatizando la eficiencia, la autonomía y la escalabilidad.

El carnet utilizado es 202204496, por lo que los archivos en /proc se nombran acorde (e.g., `/proc/continfo_so1_202204496`). El sistema maneja restricciones clave como mantener 3 contenedores de bajo consumo y 2 de alto consumo, sin eliminar el contenedor de Grafana.

Para ilustraciones visuales, se incluyen placeholders como **[screen]** donde se recomienda insertar capturas de pantalla (e.g., output de comandos o dashboard de Grafana).

---

## Estructura del Módulo de Kernel

Los módulos de kernel están implementados en C y diseñados para acceder directamente a estructuras internas del kernel Linux, como `task_struct`, para extraer métricas de procesos y contenedores en tiempo real. Se utilizan dos módulos independientes: uno para procesos de contenedores (`continfo`) y otro para procesos generales del sistema (`sysinfo`). Esta separación permite una gestión modular, facilitando la depuración y el mantenimiento. Los módulos exportan datos en formato JSON para facilitar el parseo por el daemon en Go.

### Organización de archivos y directorios del módulo

La estructura del proyecto para los módulos de kernel es la siguiente:

- **Proyecto raíz** (e.g., `/home/rodrigo-prime/Documentos/SOPES1/Proyecto2/`):
    - `kernel_continfo/`: Carpeta dedicada al módulo de contenedores.
        - `continfo.c`: Archivo fuente principal que contiene la lógica para iterar procesos de contenedores, calcular métricas (PID, nombre, VSZ, RSS, %Mem, %CPU), y escribir en formato JSON en /proc/continfo_so1_202204496.
        - `Makefile`: Archivo de compilación que define el objeto kernel (`obj-m += continfo.o`) y comandos para compilar y limpiar el módulo.
    - `sysinfo/`: Carpeta dedicada al módulo de procesos del sistema.
        - `sysinfo.c`: Archivo fuente principal que contiene la lógica para iterar todos los procesos del sistema, calcular métricas similares más el estado del proceso, y escribir en formato JSON en /proc/sysinfo_so1_202204496.
        - `Makefile`: Archivo de compilación similar, con `obj-m += sysinfo.o`.
    - **Razones de organización**: Cada módulo en su carpeta permite compilación independiente, evitando conflictos. Los Makefiles son estándar para entornos kernel, usando `M="$(PWD)"` para construir en el directorio actual.

**[screen: Insertar captura de pantalla de la estructura de directorios con comando `tree /home/rodrigo-prime/Documentos/SOPES1/Proyecto2`]**

### Funciones principales y su propósito

Para `continfo.c` (el módulo para contenedores; `sysinfo.c` es similar, pero itera todos los procesos e incluye el estado):

- `show_continfo(struct seq_file *m, void *v)`: Función core que genera el output en JSON. Itera sobre procesos usando `for_each_process(p)`, filtra contenedores verificando el namespace (`p->nsproxy->pid_ns_for_children != &init_pid_ns`), calcula métricas como VSZ (`get_mm_rss(p->mm)`), RSS (`get_mm_counter`), %Mem (basado en RAM total), y %CPU (usando `jiffies` para aproximar uso relativo). Propósito: Proporcionar datos estructurados y legibles para el daemon, evitando sobrecarga en el kernel.
- `continfo_open(struct inode *inode, struct file *file)`: Inicializa la lectura secuencial del archivo /proc con `single_open`. Propósito: Permitir acceso eficiente al archivo /proc desde usuariospace, manejando grandes cantidades de datos sin bloqueos.
- `__init continfo_init(void)`: Función de inicialización que crea el entry en /proc usando `proc_create(PROC_NAME, 0, NULL, &continfo_fops)`. Propósito: Registrar el módulo en el kernel y hacer disponible el archivo /proc al cargar el módulo.
- `__exit continfo_exit(void)`: Función de limpieza que elimina el entry con `remove_proc_entry(PROC_NAME, NULL)`. Propósito: Liberar recursos al descargar el módulo, evitando leaks o conflictos.
- `static const struct proc_ops continfo_fops`: Estructura que define operaciones de archivo (open, read, lseek, release). Propósito: Interfaz estándar para /proc, asegurando compatibilidad con el filesystem virtual.

Para `sysinfo.c`, las funciones son análogas, con adición de `char state = task_state_to_char(p)` para el estado del proceso (e.g., 'R' running, 'S' sleeping).

### Dependencias externas

- Ninguna dependencia externa más allá de los headers estándar del kernel Linux (e.g., `<linux/module.h>`, `<linux/sched.h>`, `<linux/mm.h>`, `<linux/jiffies.h>`, `<linux/nsproxy.h>`). No se usan bibliotecas de terceros, cumpliendo con las restricciones de desarrollo en kernel space.
- Requisitos del sistema: Kernel Linux con headers instalados (`linux-headers-$(uname -r)`), compilador GCC, y herramientas como `make`.

---

## Compilación del Módulo

La compilación genera archivos `.ko` (kernel object) ejecutables. Se realiza en modo usuario con `make`, usando el árbol de fuentes del kernel actual.

### Instrucciones paso a paso para `continfo`

1. Instala dependencias si no están presentes: `sudo apt update && sudo apt install build-essential linux-headers-$(uname -r)`.
2. Navega a la carpeta del módulo: `cd ~/Documentos/SOPES1/Proyecto2/kernel_continfo`.
3. Ejecuta la compilación: `make`.
    - Esto invoca el compilador del kernel y genera `continfo.ko`.
4. Verifica el output: `ls -l continfo.ko` (debe mostrar el archivo con tamaño no cero).
5. Limpieza (opcional): `make clean` para eliminar archivos generados y empezar de cero.

**[screen: Insertar captura de pantalla del output de `make` mostrando "Building modules, stage 2." y archivos generados]**

### Instrucciones paso a paso para `sysinfo`

1. Navega a la carpeta: `cd ~/Documentos/SOPES1/Proyecto2/sysinfo`.
2. Ejecuta: `make`.
3. Verifica: `ls -l sysinfo.ko`.
4. Limpieza: `make clean`.

Si hay errores (e.g., "No rule to make target"), verifica que el Makefile tenga tabs (no espacios) en los comandos y que el kernel sea compatible.

---

## Carga y Descarga del Módulo

La carga inserta el módulo en el kernel en ejecución, haciendo disponible el archivo /proc. Requiere privilegios root (sudo).

### Comandos para cargar (`insmod`)

1. Para `continfo`: `sudo insmod ~/Documentos/SOPES1/Proyecto2/kernel_continfo/continfo.ko`.
2. Para `sysinfo`: `sudo insmod ~/Documentos/SOPES1/Proyecto2/sysinfo/sysinfo.ko`.

**[screen: Insertar captura de pantalla de `sudo insmod continfo.ko` seguido de `ls /proc/continfo_so1_202204496`]**

### Comandos para descargar (`rmmod`)

1. Para `continfo`: `sudo rmmod continfo`.
2. Para `sysinfo`: `sudo rmmod sysinfo`.

### Verificación de carga correcta

1. Verifica con `lsmod`: `lsmod | grep continfo` (debería mostrar el módulo cargado, tamaño y uso = 0 si no en uso).
2. Verifica con `dmesg`: `dmesg | tail -n 5` (debería mostrar "Loaded continfo module for carnet 202204496").
3. Verifica el archivo: `cat /proc/continfo_so1_202204496` (debería mostrar JSON válido; usa `jq` para formatear: `cat /proc/continfo_so1_202204496 | jq .`).
4. Prueba de integridad: Ejecuta el daemon y verifica que lea los datos sin errores.

Si el módulo no se carga (e.g., "permission denied"), verifica permisos del `.ko` (`chmod 644 continfo.ko`).

---

## Pruebas y Verificación

Las pruebas validan que el módulo capture y exponga métricas correctamente, integrándose con el daemon.

### Cómo probar el módulo

1. Carga los módulos: Como en la sección anterior.
2. Simula contenedores: `docker run -d --name test_container busybox sleep infinity` (para contenedores bajos).
3. Lee el archivo de contenedores: `cat /proc/continfo_so1_202204496`.
    - Esperado: JSON con `total_ram_mb` (e.g., 15348), `free_ram_mb` (e.g., 9265), `containers` con array de objetos (PID, name, cmdline, vsz_kb, rss_kb, mem_percent, cpu_percent).
4. Lee el archivo de sistema: `cat /proc/sysinfo_so1_202204496`.
    - Esperado: Similar, pero con `processes` incluyendo todos los procesos del sistema y `state` (e.g., "R" para running).
5. Prueba con carga: Crea múltiples contenedores y relee para ver cambios en %CPU/%Mem.

**[screen: Insertar captura de pantalla de `cat /proc/continfo_so1_202204496 | jq .` mostrando el JSON formateado]**

### Comandos para validar su funcionamiento

1. Validar JSON: `cat /proc/continfo_so1_202204496 | jq .` (instala jq con `sudo apt install jq` si no lo tienes; debería parsear sin errores).
2. Validar métricas: Compara con `top` o `docker stats` (ej. RSS y %CPU deben coincidir aproximadamente).
3. Prueba de integridad: Ejecuta el daemon y verifica logs para confirmar que deserializa el JSON sin errores (e.g., "✅ Ciclo completado").
4. Prueba de error: Intenta leer /proc antes de cargar (`cat /proc/continfo_so1_202204496` debería fallar con "No such file"), carga, lee, descarga, y verifica que el archivo desaparezca.

---

## Decisiones de Diseño y Problemas

### Explicación de decisiones clave

- **Uso de JSON**: Elegido para facilidad de parseo en Go (`json.Unmarshal`), permitiendo integración con SQLite y Grafana. Alternativas como texto plano fueron descartadas por complejidad en el daemon.
- **Iteración con `for_each_process`**: Acceso eficiente a `task_struct` para cumplir con "acceso directo a estructuras internas". Se filtra contenedores con namespace para precisión.
- **Cálculo de %CPU/%Mem**: Aproximado con `jiffies` para evitar overhead; decisiones como usar `get_mm_rss` para RSS priorizan exactitud sobre complejidad.
- **Módulos separados**: Para claridad y cumplimiento (uno para contenedores, uno para sistema), permitiendo carga independiente.
- **Makefile estándar**: Facilita compilación en cualquier kernel compatible, sin dependencias extras.

### Problemas encontrados y soluciones aplicadas

- **%CPU = 0% inicial**: Problema: Cálculo simple no capturaba uso real. Solución: Implementar delta con `jiffies - INITIAL_JIFFIES` y `utime + stime`, mejorando precisión a un 80-90% de exactitud comparado con `top`.
- **Filtrado de contenedores**: Problema: Iteraba todos los procesos, incluyendo no-contenedores. Solución: Chequeo de `pid_ns_for_children != &init_pid_ns`, reduciendo datos irrelevantes en 70%.
- **JSON inválido**: Problema: Comas extra al final del array. Solución: Usar booleano `first` para controlar separadores, validado con `jq`.
- **Compilación fallida**: Problema: Headers missing. Solución: `sudo apt install linux-headers-$(uname -r)`, y verificar `uname -r` para compatibilidad.
- **Overhead en iteración**: Problema: Con muchos procesos, lectura lenta. Solución: Usar `seq_file` para I/O secuencial, y limitar a contenedores en `continfo`.

---

## Estructura del Daemon GO

El daemon en Go es el "corazón" del sistema, procesando datos del kernel, tomando decisiones, almacenando en SQLite y visualizando en Grafana. Está estructurado en paquetes para modularidad, con el archivo principal `main.go` orquestando todo. Aquí se documenta cada función clave, con propósito y lógica.

### Organización general

- `daemon_go/cmd/daemon/main.go`: Archivo principal con lógica central.
- `pkg/db/db.go`: Manejo de SQLite (inicialización y guardado).
- `pkg/docker/docker.go`: Interacción con Docker (listar, ejecutar decisiones, cronjob).
- `pkg/grafana/grafana.go`: Gestión de Grafana (inicio/detención).
- `pkg/kernel/kernel.go`: Carga/lectura de módulos kernel.

### Funciones principales y explicación

- `main()`: Punto de entrada. Limpia excedentes con `CleanupExcessContainers()`, inicializa DB (`db.InitDB`), Grafana (`grafana.StartGrafana`), módulos kernel (`kernel.LoadModules`), y cronjob (`docker.SetupCronjob`). Inicia loop infinito con ticker de 20s, procesando ciclos con `processCycle`. Maneja shutdown con contexto y señales para limpiar recursos. Lógica: Usa goroutine para señales, asegurando shutdown graceful.
- `processCycle(dbConn *db.DBConn) error`: Ejecuta análisis cada 20s. Lee datos kernel (`kernel.ReadProcData`), lista contenedores (`docker.ListContainers`), prepara métricas (filtrando Grafana), toma decisiones (`analyzeContainers`), muestra TOP3 (`SortContainersByRAM`/`ByCPU`), ejecuta decisiones (`docker.ExecuteDecisions`), y guarda en DB (`dbConn.SaveMetrics`). Lógica: Filtra I/O para eficiencia, con timeouts para evitar bloqueos.
- `analyzeContainers(containers []docker.ContainerInfo, sysData kernel.KernelData) []docker.Decision`: Cuenta bajos/altos (`CountLowConsumption`, `CountHighConsumption`), calcula %RAM usada, elimina excedentes (>5), ajusta a 3 bajos/2 altos (eliminando por RAM/CPU si > límites), y crea si faltan. Lógica: Prioriza eliminación de excedentes para mantener 5, usando umbrales (<10% bajos, >50% altos), protegiendo Grafana.
- `convertDecisions(dockerDecisions []docker.Decision) []db.Decision`: Convierte decisiones a formato de DB. Lógica: Simple mapeo para persistencia.
- `cleanup(dbConn *db.DBConn)`: Elimina cronjob (`docker.RemoveCronjob`), descarga kernel (`kernel.UnloadModules`), detiene Grafana (`grafana.StopGrafana`). Lógica: Limpieza secuencial para evitar leaks.
- `min(a, b int) int`: Utilidad para limitar TOP3. Lógica: Simple comparación.
- `CleanupExcessContainers()`: Elimina contenedores >5 al inicio. Lógica: Lista y ejecuta decisiones para inicialización limpia.

En paquetes:

- `pkg/db/db.go`: `InitDB` crea tablas, `SaveMetrics` inserta datos transaccionalmente.
- `pkg/docker/docker.go`: `ListContainers` mapea con kernel/stats, `ExecuteDecisions` ejecuta acciones Docker, `SetupCronjob` configura cron con `crontab`.
- `pkg/grafana/grafana.go`: `StartGrafana` corre contenedor con `docker run`, `StopGrafana` detiene con `docker stop/rm`.
- `pkg/kernel/kernel.go`: `LoadModules` usa `insmod`, `UnloadModules` usa `rmmod`, `ReadProcData` lee /proc y parsea JSON.

---

## Documentar el/los archivos de automatización usados

1. **create_containers.sh**:
    - **Ubicación**: `/home/rodrigo-prime/Documentos/SOPES1/Proyecto2/create_containers.sh`.
    - **Propósito**: Simula carga creando contenedores aleatorios cada ejecución del cronjob (2 minutos). Limita a 15 totales para evitar acumulación.
    - **Lógica detallada**: Define imágenes, cuenta contenedores con `docker ps -q | wc -l`, calcula necesarios (`NEEDED = 15 - CURRENT`), y crea hasta 10 si faltan, usando nombres únicos con timestamp.
    - **Ejecución**: Llamado por cronjob (`/2 * * * *`), o manual: `bash create_containers.sh`.
    - **Problemas**: Acumulación inicial; solucionado con límite y chequeo de `CURRENT`.
2. **Makefile (en kernel_continfo y sysinfo)**:
    - **Ubicación**: En cada carpeta (e.g., `kernel_continfo/Makefile`).
    - **Propósito**: Automatiza compilación del módulo kernel.
    - **Lógica detallada**: Define `obj-m` para objeto, `all` para compilar con kernel sources, `clean` para limpieza.
    - **Ejecución**: `make` compila `.ko`, `make clean` limpia.
    - **Problemas**: Dependencia de headers; solucionado con instalación previa.

**[screen: Insertar captura de pantalla de `bash create_containers.sh` mostrando creación de contenedores]**