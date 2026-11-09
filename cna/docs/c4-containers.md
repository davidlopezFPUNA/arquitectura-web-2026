# C4 - Nivel 2: Diagrama de Contenedores (F5-E6)

```mermaid
C4Container
  title Diagrama de Contenedores de la Solucion (BookStack CNA)

  UpdateLayoutConfig($c4ShapeInRow="4", $c4BoundaryInRow="1")

  Person(usuario, "Usuario Final", "Cliente Web/API")

  System_Boundary(cna_boundary, "Plataforma BookStack CNA") {
    Container(gateway, "API Gateway", "Nginx", "Ruteo y SSL", "Port 80/443")
    Container(app_monolith, "BookStack Core", "PHP/Laravel", "Gestion principal", "Port 8080")
    Container(search_svc, "Search Service", "Node.js", "Busqueda indexada", "Port 3001")
    Container(export_svc, "Export Service", "Node.js", "Exportaciones", "Port 3002")
    
    ContainerDb(db_core, "Core Database", "MySQL 8.0", "Datos y usuarios", "Port 3306")
    ContainerDb(event_bus, "Message Broker", "RabbitMQ", "Bus de eventos", "Port 5672")
  }

  System_Ext(storage_ext, "Object Storage", "MinIO / S3", "Archivos y PDFs")

  Rel(usuario, gateway, "Peticiones", "HTTPS")
  Rel(gateway, app_monolith, "Legacy", "HTTP")
  Rel(gateway, search_svc, "/v2/search", "HTTP")
  Rel(gateway, export_svc, "/v2/export", "HTTP")

  Rel(app_monolith, db_core, "Persistencia", "MySQL")
  Rel(search_svc, db_core, "Consultas", "MySQL")
  
  Rel(app_monolith, event_bus, "Publica eventos", "AMQP")
  Rel(export_svc, event_bus, "Eventos export", "AMQP")
  Rel(export_svc, storage_ext, "Guarda archivos", "S3 API")
