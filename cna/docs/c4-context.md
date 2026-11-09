# C4 - Nivel 1: Diagrama de Contexto (F5-E5)

```mermaid
flowchart TD
    classDef person fill:#08427b,stroke:#073b6f,color:#fff,stroke-width:2px;
    classDef system fill:#1168bd,stroke:#0b4884,color:#fff,stroke-width:2px;
    classDef external fill:#999999,stroke:#666666,color:#fff,stroke-width:2px;

    U["👤 <b>Usuario Final</b><br/>[Person]<br/>Usuario que interactúa con la plataforma para buscar, editar y exportar contenido."]:::person
    
    SYSTEM["<b>Sistema BookStack CNA</b><br/>[Software System]<br/>Solución Cloud Native de gestión de conocimiento distribuida."]:::system

    S3["<b>Almacenamiento de Archivos</b><br/>[External System]<br/>Almacena adjuntos y archivos multimedia (S3/Local)."]:::external
    AUTH["<b>Proveedor de Identidad</b><br/>[External System]<br/>Autentica usuarios y gestiona tokens (OIDC/OAuth2)."]:::external

    %% Relaciones
    U -->|HTTPS / REST<br/>Búsqueda, lectura y exportación| SYSTEM
    SYSTEM -->|S3 API / File System<br/>Lee y escribe archivos multimedia| S3
    SYSTEM -->|HTTPS / OIDC<br/>Valida identidad y permisos| AUTH
