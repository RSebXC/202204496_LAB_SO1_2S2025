## Configuración Inicial de GCP\n- Proyecto creado: proyecto3-k8s-2025-so1\n- APIs habilitadas: container.googleapis.com, compute.googleapis.com\n- Zona por defecto: us-central1-a
## NGINX Ingress Controller\n- Instalado en namespace: dev\n- IP externa: 136.114.87.48
## Verificación de Clúster GKE\n- Clúster: proyecto3-cluster\n- Namespace: dev\n- Nodos verificados con kubectl get nodes.
## Kafka\n- Instalado con Strimzi en namespace dev\n- Clúster: my-kafka (1 réplica)\n- Puerto: 9092 (interno, sin TLS)
## RabbitMQ\n- Instalado con Bitnami en namespace dev\n- Credenciales: admin/admin123\n- URL: rabbitmq.dev.svc.cluster.local:5672
## Valkey\n- Instalado con Bitnami en namespace dev\n- Réplicas: 2\n- Credenciales: valkey123\n- URL: valkey-master.dev.svc.cluster.local:6379
## Grafana\n- Instalado en namespace dev\n- Credenciales: admin/admin123\n- URL: http://34.57.222.111
## API REST en Rust\n- Proyecto creado en rust-api/\n- Endpoint: POST /tweet\n- Valida municipality y weather, envía a go-service.dev.svc.cluster.local:8080
## Despliegue de Rust API\n- Imagen rust-api:latest subida a Zot (34.122.62.252:5000)\n- Configurado imagePullSecret para GKE\n- Pod rust-api en Running
## Consumidores y Valkey\n- Implementado consumidor Go para Kafka\n- Implementado consumidor Go para RabbitMQ\n- Configurado Valkey en GKE\n- Resuelto problema de rabbitmq-service con credenciales
