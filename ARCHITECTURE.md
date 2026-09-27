# Архитектура GophProfile в Kubernetes

Ниже представлена схема взаимодействия компонентов системы при развертывании в локальном Kubernetes-кластере (Rancher Desktop / minikube) с использованием Helm.

```mermaid
graph TD
    Client([Клиент / Браузер]) --> NodePort[NodePort :30080]
    
    subgraph Kubernetes Cluster
        NodePort --> Srv[Service: gophprofile-server]
        
        Srv --> Pod1[Pod: gophprofile-server]
        Srv --> Pod2[Pod: gophprofile-server]
        
        Pod1 & Pod2 --> PG[(PostgreSQL)]
        Pod1 & Pod2 --> S3[(MinIO / S3)]
        Pod1 & Pod2 --> Kafka[(Kafka)]
        
        W1[Pod: gophprofile-worker] --> PG
        W1 --> S3
        W1 --> Kafka
        
        Pod1 & Pod2 & W1 -.->|OTLP gRPC| OTel[OTel Collector]
        
        OTel -->|Metrics /metrics| Prom[(Prometheus)]
        Prom --> Grafana[Grafana :30081]
        OTel -->|Traces OTLP| Jaeger[Jaeger]
        
        KExp[Kafka Exporter] -->|Metrics| Prom
    end
    
    style Client fill:#f9f,stroke:#333,stroke-width:2px
    style NodePort fill:#bbf,stroke:#333,stroke-width:2px
    style PG fill:#f96,stroke:#333,stroke-width:2px
    style S3 fill:#f96,stroke:#333,stroke-width:2px
    style Kafka fill:#f96,stroke:#333,stroke-width:2px
```

### Описание компонентов:
1. **Client:** Запросы поступают на NodePort (порт 30080).
2. **Server (API):** Обрабатывает HTTP-запросы, загружает оригиналы в S3, сохраняет метаданные в PostgreSQL и отправляет задачи в Kafka. Масштабируется с помощью HPA.
3. **Worker:** Асинхронно читает сообщения из Kafka, скачивает оригинал, делает миниатюры, грузит их обратно в S3 и обновляет статусы в БД.
4. **PostgreSQL:** Хранит метаданные аватарок.
5. **MinIO:** S3-совместимое хранилище бинарных файлов (оригиналов и миниатюр).
6. **Kafka:** Брокер сообщений для асинхронной обработки.
7. **OTel Collector:** Собирает трейсы и метрики (бизнес-метрики и метрики БД) с Server и Worker.
8. **Prometheus:** Скрапит метрики с OTel Collector (порт 8889) и Kafka Exporter. Настроен через `ServiceMonitor` (Prometheus Operator).
9. **Grafana:** Визуализирует метрики из Prometheus. Дашборды (Business KPIs, Kafka Monitoring) загружаются автоматически из ConfigMap.
