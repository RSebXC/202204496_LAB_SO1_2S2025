# 📌 Guion de presentación – Práctica Única

### 1. Iniciar Minikube

```bash
minikube start --driver=docker --memory=4096 --cpus=2 --insecure-registry="192.168.49.2:32000"
```

✔ Arranca el clúster con soporte para el registro inseguro en el puerto `32000`.

---

### 2. Verificar nodos y pods básicos

```bash
kubectl get nodes
kubectl get pods -A
```

✔ Demuestra que Kubernetes está corriendo correctamente.

---

### 3. Habilitar Ingress

```bash
minikube addons enable ingress
kubectl get pods -n ingress-nginx
```

✔ Muestra que el controlador NGINX está activo.

---

### 4. Aplicar los manifiestos

```bash
kubectl apply -f zot-pod.yaml
kubectl apply -f zot-service.yaml
kubectl apply -f zot-ingress.yaml
```

✔ Levanta el pod, el servicio y el ingress de tu registro.

---

### 5. Verificar recursos

```bash
kubectl get pods
kubectl get svc
kubectl get ingress
```

✔ Confirma que todo está corriendo.

---

### 6. Comprobar el registro

```bash
curl http://zot.local/v2/_catalog
curl http://zot.local/v2/alpine/tags/list
```

✔ Demuestra que tu imagen (`alpine:latest`) está en el registro.

---

### 7. Confirmar que el Pod usa la imagen del registro

```bash
kubectl describe pod zot-pod | grep -i "Image"
```

✔ Muestra la imagen usada (`192.168.49.2:32000/alpine:latest`).
