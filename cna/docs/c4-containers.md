# C4 - Nivel 2: Diagrama de Contenedores (F5-E6)

```mermaid
flowchart TD
    classDef person fill:#08427b,stroke:#073b6f,color:#fff,stroke-width:2px;
    classDef container fill:#438dd5,stroke:#3c7fc0,color:#fff,stroke-width:2px;
    classDef db fill:#2b7dbd,stroke:#246ba3,color:#fff,stroke-width:2px;
    classDef external fill:#999999,stroke:#666666,color:#fff,stroke-width:2px;

    U["👤 <b>Usuario Final</b><br/>[Person]<br/>Cliente Web / API"]:::person
    
    subgraph CNA ["<b>Plataforma BookStack CNA</b>"]
        GW["<b>API Gateway</b><br/>[Nginx]<br/>Ruteo y SSL (Port 80/443)"]:::container
        CORE["<b>BookStack Core</b><br/>[PHP / Laravel]<br/>Gestión principal (Port 8080)"]:::container
        SEARCH["<b>Search Service</b><br/>[Node.js / Express]<br/>Búsqueda indexada (Port 3001)"]:::container
        EXPORT["<b>Export Service</b><br/>[Node.js / Worker]<br/>Exportaciones (Port 3002)"]:::container
        
        DB[("<b>Core Database</b><br/>[MySQL 8.0]<br/>Datos y usuarios (Port 3306)")]:::db
        BUS[("<b>Message Broker</b><br/>[RabbitMQ]<br/>Bus de eventos (Port 5672)")]:::db
    end

    S3["<b>Object Storage</b><br/>[MinIO / S3]<br/>Archivos y PDFs"]:::external

    %% Relaciones de Entrada
    U -->|HTTPS / 443| GW

    %% Ruteo desde Gateway
    GW -->|HTTP / Legacy| CORE
    GW -->|HTTP / /v2/search| SEARCH
    GW -->|HTTP / /v2/export| EXPORT

    %% Acceso a Datos
    CORE -->|MySQL / 3306| DB
    SEARCH -->|MySQL / 3306| DB

    %% Eventos y Almacenamiento
    CORE -->|AMQP / Publish| BUS
    EXPORT -->|AMQP / Consume-Publish| BUS
    EXPORT -->|S3 API / 9000| S3
