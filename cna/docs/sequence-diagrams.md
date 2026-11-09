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
