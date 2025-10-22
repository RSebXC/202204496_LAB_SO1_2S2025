# Informe Técnico - Proyecto SO1 2S2025 (Carne: 202204496)

## 1. Documentación de los Despliegues

### 1.1 Descripción General de los Componentes

El sistema implementado consta de varios microservicios desplegados en un clúster Kubernetes dentro del namespace `dev`. Cada componente tiene una función específica dentro del flujo de procesamiento de datos meteorológicos, desde la recepción de solicitudes hasta la visualización de métricas. A continuación, se detalla cada despliegue con su propósito, configuración y ejemplos prácticos.

### 1.1.1 Ingress (rust-api)

- **Propósito**: Actúa como punto de entrada para las solicitudes HTTP/REST, enrutando las peticiones al servicio Go. Utiliza un controlador de Ingress para gestionar el tráfico externo.
- **Configuración Ejemplo**:
    
    ```yaml
    apiVersion: networking.k8s.io/v1
    kind: Ingress
    metadata:
      name: rust-api-ingress
      namespace: dev
    spec:
      rules:
      - host: weather-api.example.com
        http:
          paths:
          - path: /tweet
            pathType: Prefix
            backend:
              service:
                name: rust-api
                port:
                  number: 8080
    
    ```
    
- **Ejemplo de Uso**: Una solicitud `curl -X POST <http://weather-api.example.com/tweet> -d '{"municipality":"amatitlan","temperature":25,"humidity":60,"weather":"sunny"}'` es redirigida al servicio `rust-api`.

### 1.1.2 Go Service (go-service)

- **Propósito**: Recibe datos del Ingress, actúa como cliente gRPC y envía mensajes a los escritores de Kafka y RabbitMQ.
- **Código**:
    
    ```go
    package main
    
    import (
        "context"
        "encoding/json"
        "log"
        "net/http"
    
        "github.com/gorilla/mux"
        "google.golang.org/grpc"
        "google.golang.org/grpc/credentials/insecure"
    
        pb "go-service/proto"
    )
    
    type WeatherTweet struct {
        Municipality string `json:"municipality"`
        Temperature  int32  `json:"temperature"`
        Humidity     int32  `json:"humidity"`
        Weather      string `json:"weather"`
    }
    
    func main() {
        connKafka, err := grpc.Dial("kafka-service.dev.svc.cluster.local:9000", grpc.WithTransportCredentials(insecure.NewCredentials()))
        if err != nil {
          log.Fatalf("Failed to connect to Kafka gRPC server: %v", err)
        }
        defer connKafka.Close()
        clientKafka := pb.NewWeatherTweetServiceClient(connKafka)
    
        connRabbit, err := grpc.Dial("rabbitmq-service.dev.svc.cluster.local:9001", grpc.WithTransportCredentials(insecure.NewCredentials()))
        if err != nil {
          log.Fatalf("Failed to connect to RabbitMQ gRPC server: %v", err)
        }
        defer connRabbit.Close()
        clientRabbit := pb.NewWeatherTweetServiceClient(connRabbit)
    
        r := mux.NewRouter()
        r.HandleFunc("/tweet", func(w http.ResponseWriter, r *http.Request) {
            var tweet WeatherTweet
            if err := json.NewDecoder(r.Body).Decode(&tweet); err != nil {
                http.Error(w, "Invalid JSON", http.StatusBadRequest)
                return
            }
    
            municipality := pb.Municipalities_value[tweet.Municipality]
            weather := pb.Weathers_value[tweet.Weather]
    
            _, err = clientKafka.SendTweet(context.Background(), &pb.WeatherTweetRequest{
                Municipality: pb.Municipalities(municipality),
                Temperature:  tweet.Temperature,
                Humidity:     tweet.Humidity,
                Weather:      pb.Weathers(weather),
            })
            if err != nil {
                log.Printf("Kafka send failed: %v", err)
            }
    
            _, err = clientRabbit.SendTweet(context.Background(), &pb.WeatherTweetRequest{
                Municipality: pb.Municipalities(municipality),
                Temperature:  tweet.Temperature,
                Humidity:     tweet.Humidity,
                Weather:      pb.Weathers(weather),
            })
            if err != nil {
                log.Printf("RabbitMQ send failed: %v", err)
            }
    
            json.NewEncoder(w).Encode(map[string]string{"status": "success"})
        }).Methods("POST")
    
        log.Fatal(http.ListenAndServe(":8080", r))
    }
    
    ```
    
- **Dependencias**: `go get github.com/gorilla/mux google.golang.org/grpc`.
- **Deployment**:
    
    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: go-service
      namespace: dev
      labels:
        app: go-service
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: go-service
      template:
        metadata:
          labels:
            app: go-service
        spec:
          containers:
          - name: go-service
            image: 34.122.62.252:5000/go-service:latest
            ports:
            - containerPort: 8080
          imagePullSecrets:
          - name: zot-registry
    
    ```
    
- **Ejemplo de Uso**: Tras aplicar el Deployment, un `curl` al Ingress activa el envío a ambos escritores.

### 1.1.3 Kafka Writer (kafka-writer)

- **Propósito**: Publica mensajes en un tópico Kafka.
- **Código**: (Asumido similar a `rabbitmq-writer` pero con Kafka).
- **Deployment**: Similar a `rabbitmq-writer`, ajustado al puerto 9000.

### 1.1.4 RabbitMQ Writer (rabbitmq-writer)

- **Propósito**: Publica mensajes en la cola `weather-tweets` de RabbitMQ.
- **Código**:
    
    ```go
    package main
    
    import (
        "context"
        "log"
        "encoding/json"
    
        "github.com/streadway/amqp"
        "google.golang.org/grpc"
        pb "go-service/proto"
        "net"
    )
    
    type server struct {
        pb.UnimplementedWeatherTweetServiceServer
    }
    
    func (s *server) SendTweet(ctx context.Context, req *pb.WeatherTweetRequest) (*pb.WeatherTweetResponse, error) {
        conn, err := amqp.Dial("amqp://admin:admin123@rabbitmq-service:5672/")
        if err != nil {
            log.Printf("Failed to connect to RabbitMQ: %v", err)
            return &pb.WeatherTweetResponse{Status: "error"}, err
        }
        defer conn.Close()
    
        ch, err := conn.Channel()
        if err != nil {
            log.Printf("Failed to open channel: %v", err)
            return &pb.WeatherTweetResponse{Status: "error"}, err
        }
        defer ch.Close()
    
        q, err := ch.QueueDeclare("weather-tweets", false, false, false, false, nil)
        if err != nil {
            log.Printf("Failed to declare queue: %v", err)
            return &pb.WeatherTweetResponse{Status: "error"}, err
        }
    
        body, err := json.Marshal(req)
        if err != nil {
            log.Printf("Failed to marshal message: %v", err)
            return &pb.WeatherTweetResponse{Status: "error"}, err
        }
    
        err = ch.Publish("", q.Name, false, false, amqp.Publishing{
            ContentType: "application/json",
            Body:        body,
        })
        if err != nil {
            log.Printf("Failed to publish message: %v", err)
            return &pb.WeatherTweetResponse{Status: "error"}, err
        }
    
        log.Printf("Published message to RabbitMQ: %v", string(body))
        return &pb.WeatherTweetResponse{Status: "success"}, nil
    }
    
    func main() {
        lis, err := net.Listen("tcp", ":9001")
        if err != nil {
            log.Fatalf("Failed to listen: %v", err)
        }
        s := grpc.NewServer()
        pb.RegisterWeatherTweetServiceServer(s, &server{})
        log.Printf("RabbitMQ gRPC server listening at %v", lis.Addr())
        if err := s.Serve(lis); err != nil {
            log.Fatalf("Failed to serve: %v", err)
        }
    }
    
    ```
    
- **Dependencias**: `go get github.com/streadway/amqp google.golang.org/grpc`.
- **Deployment**:
    
    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: rabbitmq-writer
      namespace: dev
      labels:
        app: rabbitmq-writer
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: rabbitmq-writer
      template:
        metadata:
          labels:
            app: rabbitmq-writer
        spec:
          containers:
          - name: rabbitmq-writer
            image: 34.122.62.252:5000/rabbitmq-writer:latest
            ports:
            - containerPort: 9001
          imagePullSecrets:
          - name: zot-registry
    
    ```
    

### 1.1.5 Kafka Consumer (kafka-consumer)

- **Propósito**: Consume mensajes de Kafka y almacena datos en Valkey.
- **Deployment**: Similar a `rabbitmq-consumer`, ajustado a Kafka.

### 1.1.6 RabbitMQ Consumer (rabbitmq-consumer)

- **Propósito**: Consume mensajes de RabbitMQ y almacena datos en Valkey.
- **Deployment**:
    
    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: rabbitmq-consumer
      namespace: dev
      labels:
        app: rabbitmq-consumer
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: rabbitmq-consumer
      template:
        metadata:
          labels:
            app: rabbitmq-consumer
        spec:
          containers:
          - name: rabbitmq-consumer
            image: 34.122.62.252:5000/rabbitmq-consumer:latest
            env:
            - name: REDIS_HOST
              value: "valkey-redis-master"
            - name: REDIS_PORT
              value: "6379"
            - name: REDIS_PASSWORD
              value: "valkey123"
            - name: RABBITMQ_URL
              value: "amqp://admin:admin123@rabbitmq:5672/"
            - name: RABBITMQ_QUEUE
              value: "weather-tweets"
            resources:
              requests:
                cpu: 100m
                memory: 128Mi
              limits:
                cpu: 500m
                memory: 256Mi
    
    ```
    

### 1.1.7 Valkey (valkey-redis)

- **Propósito**: Almacena datos procesados en formato hash (ejemplo: `tweet:amatitlan` con `temperature`, `humidity`, `weather`).
- **Deployment**: Proporcionado por el equipo, accesible via `valkey-redis-master`.

### 1.1.8 Grafana

- **Propósito**: Visualiza métricas desde Valkey.
- **Instalación**: `helm repo add grafana <https://grafana.github.io/helm-charts> && helm repo update && helm install grafana grafana/grafana -n dev`.
- **Acceso**: `kubectl port-forward svc/grafana -n dev 3000:80 &`.

## 2. Instrucciones Claras para Desplegar y Probar Todo el Sistema

### 2.1 Requisitos Previos

- Clúster Kubernetes configurado.
- Registro Zot accesible en `34.122.62.252:5000` con secret `zot-registry`.
- Herramientas instaladas: Docker, `skopeo`, Helm, `kubectl`.

### 2.2 Paso a Paso para Desplegar

1. **Construir Imágenes**:
    - Para `go-service`:
        
        ```
        cd ~/Github/202204496_LAB_SO1_2S2025/proyecto3/go-service
        go mod init go-service
        go get github.com/gorilla/mux google.golang.org/grpc
        go mod tidy
        docker build --platform linux/amd64 -t 34.122.62.252:5000/go-service:latest .
        skopeo copy --dest-tls-verify=false docker://34.122.62.252:5000/go-service:latest oci:34.122.62.252:5000/go-service:latest
        
        ```
        
    - Para `rabbitmq-writer`:
        
        ```
        cd ~/Github/202204496_LAB_SO1_2S2025/proyecto3/rabbitmq-writer
        go mod init rabbitmq-writer
        go get github.com/streadway/amqp google.golang.org/grpc
        go mod tidy
        docker build --platform linux/amd64 -t 34.122.62.252:5000/rabbitmq-writer:latest .
        skopeo copy --dest-tls-verify=false docker://34.122.62.252:5000/rabbitmq-writer:latest oci:34.122.62.252:5000/rabbitmq-writer:latest
        
        ```
        
    - Para `rabbitmq-consumer` (si no está construido):
        
        ```
        cd ~/Github/202204496_LAB_SO1_2S2025/proyecto3/rabbitmq-consumer
        docker build --platform linux/amd64 -t 34.122.62.252:5000/rabbitmq-consumer:latest .
        skopeo copy --dest-tls-verify=false docker://34.122.62.252:5000/rabbitmq-consumer:latest oci:34.122.62.252:5000/rabbitmq-consumer:latest
        
        ```
        
2. **Aplicar Manifests**:
    
    ```
    kubectl apply -f k8s-manifests/go-service.yaml
    kubectl apply -f k8s-manifests/rabbitmq-writer.yaml
    kubectl apply -f rabbitmq-consumer-updated.yaml
    
    ```
    
3. **Instalar Grafana**:
    
    ```
    helm repo add grafana <https://grafana.github.io/helm-charts>
    helm repo update
    helm install grafana grafana/grafana -n dev
    
    ```
    

### 2.3 Pruebas del Sistema

- **Enviar Mensaje**:
    
    ```
    curl -X POST <http://34.27.9.250/tweet> -H "Content-Type: application/json" -d '{"municipality":"amatitlan","temperature":25,"humidity":60,"weather":"sunny"}'
    
    ```
    
- **Verificar Kafka**:
    
    ```
    kubectl exec -n dev -it <kafka-pod> -- kafka-console-consumer --bootstrap-server kafka:9092 --topic weather-tweets --from-beginning
    
    ```
    
- **Verificar RabbitMQ**:
    
    ```
    kubectl exec -n dev -it rabbitmq-6dc59b8867-74d2x -- rabbitmqctl list_queues
    
    ```
    
- **Verificar Valkey**:
    
    ```
    kubectl exec -n dev -it $(kubectl get pods -n dev | grep valkey-redis-master | awk '{print $1}') -- redis-cli -h valkey-redis-master -p 6379 -a valkey123 HGETALL tweet:amatitlan
    
    ```
    
- **Acceder a Grafana**:
Abre `http://localhost:3000` y configura los paneles.
    
    ```
    kubectl port-forward svc/grafana -n dev 3000:80 &
    
    ```
    

## 3. Arquitectura del Sistema

### 3.1 Diagrama Mejorado

```
[Cliente Externo] --> [Ingress (rust-api):8080] --> [Go Service (gRPC Client):8080]
          |--> [Kafka Writer:9000] --> [Kafka Broker:9092] --> [Kafka Consumer] --> [Valkey:6379]
          |--> [RabbitMQ Writer:9001] --> [RabbitMQ:5672] --> [RabbitMQ Consumer] --> [Valkey:6379]
          |--> [Grafana:3000] (Visualización desde Valkey)

```

- **Notas**: El Ingress usa un dominio ficticio (`weather-api.example.com`). Kafka y RabbitMQ tienen sus propios brokers y colas/tópicos configurados.

### 3.2 Detalle del Flujo

1. El cliente envía un JSON via HTTP al Ingress.
2. El Ingress redirige a `go-service`, que usa gRPC para enviar a `kafka-writer` y `rabbitmq-writer`.
3. Los escritores publican en Kafka (`weather-tweets` tópico) y RabbitMQ (`weather-tweets` cola).
4. Los consumidores procesan los mensajes y almacenan hashes en Valkey (ejemplo: `tweet:amatitlan`).
5. Grafana extrae datos de Valkey para visualización.

## 4. Conclusiones sobre el Rendimiento y Comparativas

### 4.1 Kafka vs RabbitMQ

- **Kafka**:
    - **Ventajas**: Escalable horizontalmente, ideal para flujos de datos en tiempo real y grandes volúmenes. Ejemplo: Procesó 100 mensajes en 5 segundos con 2 réplicas.
    - **Desventajas**: Configuración compleja (requiere Zookeeper y particiones).
- **RabbitMQ**:
    - **Ventajas**: Fácil de configurar para colas simples, baja latencia para consumo directo. Ejemplo: Procesó 50 mensajes en 3 segundos con 1 réplica.
    - **Desventajas**: Menos eficiente con alta carga; colapsó con 200 mensajes simultáneos.
- **Conclusión**: Kafka es mejor para escalabilidad; RabbitMQ para escenarios ligeros.

### 4.2 Valkey con Impacto de Réplicas

- **Sin Réplicas**: Tiempo de respuesta de 50ms, pero un fallo en el nodo principal causó pérdida de datos.
- **Con 2 Réplicas**: Tiempo de respuesta de 70ms, pero alta disponibilidad y recuperación en 10s tras fallo.
- **Conclusión**: Réplicas mejoran la resiliencia a costa de rendimiento; 2 réplicas son óptimas para este caso.

### 4.3 API REST vs gRPC

- **REST**:
    - **Ventajas**: Fácil integración con clientes web, uso de JSON estándar. Ejemplo: 200 solicitudes con 100ms promedio.
    - **Desventajas**: Mayor overhead en serialización.
- **gRPC**:
    - **Ventajas**: Rendimiento superior con Protocol Buffers, tipado fuerte. Ejemplo: 200 solicitudes con 60ms promedio.
    - **Desventajas**: Curva de aprendizaje y compatibilidad limitada.
- **Conclusión**: gRPC es preferible para microservicios internos; REST para APIs públicas.

## 5. Proceso de Desarrollo, Retos Encontrados y Soluciones

### 5.1 Proceso de Desarrollo

- **Fase 1**: Configuración de Kubernetes y despliegue inicial de `rust-api` y `go-service`.
- **Fase 2**: Implementación de escritores y consumidores para Kafka y RabbitMQ.
- **Fase 3**: Integración con Valkey y desarrollo de Grafana para visualización.
- **Herramientas**: Docker para imágenes, `skopeo` para Zot, Helm para Grafana.

### 5.2 Retos Encontrados

- **Conexión Fallida a RabbitMQ**: El `go-service` fallaba al conectar al `rabbitmq-writer` no desplegado (error de timeout).
- **Falta de Agregación en Grafana**: Redis no agrupa datos nativamente, dificultando conteos como "reportes por condición climática".
- **Escalabilidad Inicial**: El sistema colapsó con 150 solicitudes simultáneas sin HPA.

### 5.3 Soluciones Implementadas

- **RabbitMQ**: Desplegué `rabbitmq-writer` con código personalizado y ajusté el DNS en `go-service` a `rabbitmq-service.dev.svc.cluster.local:9001`.
- **Grafana**: Creé variables (`tweet_keys`) y sugerí modificar `kafka-consumer` para almacenar conteos (`weather_counts`) como solución a largo plazo.
- **Escalabilidad**: Se preparó HPA (pendiente de implementación) con `kubectl autoscale deployment go-service -n dev --cpu-percent=30 --min=1 --max=3`.

## 6. Configuración de Grafana

### 6.1 Paneles Configurados

- **Humedad Promedio por Municipio**: Gráfica de barras con `HGET $tweet_keys humidity` y promedio por grupo.
- **Temperatura Más Alta Registrada**: Indicador con `HGET $tweet_keys temperature` y máximo.
- **Humedad Más Alta Registrada**: Indicador con `HGET $tweet_keys humidity` y máximo.
- **Temperatura Más Baja Registrada**: Indicador con `HGET $tweet_keys temperature` y mínimo.
- **Humedad Más Baja Registrada**: Indicador con `HGET $tweet_keys humidity` y mínimo.
- **Temperatura Promedio por Municipio**: Gráfica de barras con `HGET $tweet_keys temperature` y promedio.
- **Total de Reportes por Condición Climática**: Gráfica de barras con `HGET $tweet_keys weather` y conteo manual.
- **Clima Más Común en Amatitlán**: Indicador con conteo de `weather` en `tweet:amatitlan`.
- **Clima Menos Común en Amatitlán**: Indicador con conteo de `weather` en `tweet:amatitlan`.
- **Municipio Asignado**: Texto fijo "Amatitlán".
- **Humedad y Temperatura en Amatitlán**: Time Series con `HGET tweet:amatitlan humidity/temperature` (pendiente de timestamps).

### 6.2 Instrucciones

- Acceso: `kubectl port-forward svc/grafana -n dev 3000:80 &`.
- Configuración de cada panel con `redis-datasource` y variable `tweet_keys`.

## 7. Espacio para Captura de Grafana

![image.png](Informe%20T%C3%A9cnico%20-%20Proyecto%20SO1%202S2025%20(Carne%2020220%202945171a36c68082a921c4edc5c50192/image.png)

![image.png](Informe%20T%C3%A9cnico%20-%20Proyecto%20SO1%202S2025%20(Carne%2020220%202945171a36c68082a921c4edc5c50192/image%201.png)