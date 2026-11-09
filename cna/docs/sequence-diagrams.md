# Diagramas de Secuencia (F5-E7)

## 1. Búsqueda Síncrona (`GET /v2/search`)

```mermaid
sequenceDiagram
    autonumber
    actor U as Usuario / Cliente
    participant GW as API Gateway
    participant SS as Search Service
    participant DB as Core DB

    alt Flujo Exitoso (200 OK)
        U->>GW: GET /v2/search?q=arquitectura
        GW->>SS: Forward request GET /v2/search?q=arquitectura
        SS->>DB: Query: Buscar páginas por término "arquitectura"
        DB-->>SS: Retorna resultados de búsqueda
        SS-->>GW: 200 OK { "results": [...] }
        GW-->>U: 200 OK { "results": [...] }
    else Error de Parámetros (400 Bad Request)
        U->>GW: GET /v2/search (sin parámetro query)
        GW->>SS: Forward request GET /v2/search
        SS-->>GW: 400 Bad Request { "error": "Query parameter 'q' is required" }
        GW-->>U: 400 Bad Request { "error": "Query parameter 'q' is required" }
    else Error de Servidor / DB Caída (503 Service Unavailable)
        U->>GW: GET /v2/search?q=arquitectura
        GW->>SS: Forward request
        SS->>DB: Query
        DB-->>SS: Connection timeout / Error
        SS-->>GW: 503 Service Unavailable { "error": "Database unavailable" }
        GW-->>U: 503 Service Unavailable { "error": "Database unavailable" }
    end

---

## 2. Solicitar Exportación Asíncrona (`POST /v2/export/{page_id}`)

sequenceDiagram
    autonumber
    actor U as Usuario / Cliente
    participant GW as API Gateway
    participant ES as Export Service
    participant EB as Message Broker (RabbitMQ)

    alt Flujo Exitoso (202 Accepted)
        U->>GW: POST /v2/export/123
        GW->>ES: Forward request POST /v2/export/123
        ES->>EB: Publish Event: export.requested { page_id: 123, job_id: "uuid-123" }
        EB-->>ES: ACK Event Enqueued
        ES-->>GW: 202 Accepted { job_id: "uuid-123", status: "pending" }
        GW-->>U: 202 Accepted { job_id: "uuid-123", status: "pending" }
    else Recurso No Encontrado (404 Not Found)
        U->>GW: POST /v2/export/99999
        GW->>ES: Forward request POST /v2/export/99999
        ES->>ES: Validar existencia de page_id 99999
        ES-->>GW: 404 Not Found { "error": "Page ID not found" }
        GW-->>U: 404 Not Found { "error": "Page ID not found" }
    end

---

## 3. Publicación y Consumo de Eventos de Dominio

sequenceDiagram
    autonumber
    actor U as Usuario
    participant Core as BookStack Core (Monolito)
    participant EB as Message Broker (RabbitMQ)
    participant ES as Export Service (Worker)
    participant S3 as Object Storage (MinIO)

    alt Procesamiento Exitoso de Evento
        U->>Core: Crear/Editar Página "Página 1"
        Core->>Core: Guardar en Base de Datos
        Core->>EB: Publish event: bookstack.page.updated (v2)
        EB->>ES: Consume event: bookstack.page.updated
        ES->>ES: Re-generar cache de PDF / Export
        ES->>S3: Upload PDF actualizado
        ES->>EB: Publish event: export.completed (v2)
    else Error en Consumo / Formato Inválido
        Core->>EB: Publish event: bookstack.page.updated (payload corrupto)
        EB->>ES: Consume event
        ES->>ES: Error al deserializar / validar sobre de evento
        ES->>EB: NACK / Reject event
        EB->>EB: Enviar mensaje a Dead Letter Queue (DLQ)
    end

---

<Steps>
  <Step subtitle="< 1 min" title="Guardar el archivo">
    Abre o crea el archivo desde la carpeta `cna/docs` ejecutando:
    ```bash
    nano sequence-diagrams.md
    ```
    Pega el texto anterior, presiona `Ctrl + O`, `Enter` para confirmar y `Ctrl + X` para salir.
  </Step>
</Steps>

<ElicitationsGroup message="¿Pudiste guardar sequence-diagrams.md?">
  <Elicitation label="Subir todo a GitHub" query="Ya guardé sequence-diagrams.md. Quiero hacer git add, commit y push de todos los diagramas creados en cna/docs."/>
  <Elicitation label="Verificar la carpeta docs" query="¿Cómo me me aseguro de que los tres archivos creados están en la carpeta correcta?"/>
</ElicitationsGroup>
