## Despliegue de Rust API en GKE\n- Deployment: rust-api (1-3 réplicas con HPA, CPU > 30%)\n- Service: rust-api (port 8080)\n- Ingress: /tweet -> rust-api\n- Imagen: 34.123.45.67:5000/rust-api:latest
