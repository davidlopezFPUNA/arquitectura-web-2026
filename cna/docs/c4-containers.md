# C4 - Nivel 2: Diagrama de Contenedores (F5-E6)

C4Container
  title Diagrama de Contenedores de la Solucion (BookStack CNA)

  Person(usuario, "Usuario Final", "Usuario interactuando via Web/API client.")

  System_Boundary(cna_boundary, "Plataforma BookStack CNA") {
    Container(gateway, "API Gateway / Nginx", "Nginx", "Ruteo de trafico, SSL termination y politicas de rate limit.", "Port: 80/443 (HTTP/HTTPS)")
    Container(app_monolith, "BookStack Core", "PHP/Laravel", "Gestion principal de usuarios, paginas y permisos.", "Port: 8080 (HTTP)")
    Container(search_svc, "Search Service", "Node.js / Express", "Microservicio de busqueda indexada y filtrado.", "Port: 3001 (HTTP/REST /v2/search)")
    Container(export_svc, "Export Service", "Node.js / Worker", "Microservicio asincrono de generacion de reportes y exportaciones.", "Port: 3002 (HTTP/REST /v2/export)")
    
    ContainerDb(db_core, "Core Database", "MySQL 8.0", "Almacena contenido de paginas, usuarios y metadata.", "Port: 3306 (MySQL Protocol)")
    ContainerDb(event_bus, "Message Broker / Event Bus", "RabbitMQ / Redpanda", "Bus de eventos asincronos para desacoplamiento.", "Port: 5672/9092 (AMQP/Kafka)")
  }

  System_Ext(storage_ext, "Object Storage", "MinIO / S3", "Almacenamiento de exports y multimedia.")

  Rel(usuario, gateway, "Envía peticiones", "HTTPS / 443")
  Rel(gateway, app_monolith, "Rutera trafico legacy", "HTTP / 8080")
  Rel(gateway, search_svc, "Rutera /v2/search", "HTTP / 3001")
  Rel(gateway, export_svc, "Rutera /v2/export", "HTTP / 3002")

  Rel(app_monolith, db_core, "Consulta y persiste datos", "TCP / 3306")
  Rel(search_svc, db_core, "Consulta indices de busqueda", "TCP / 3306")
  
  Rel(app_monolith, event_bus, "Publica eventos (bookstack.page.created, etc.)", "AMQP / 5672")
  Rel(export_svc, event_bus, "Consume/Publica eventos (export.requested, export.completed)", "AMQP / 5672")
  Rel(export_svc, storage_ext, "Guarda archivos exportados", "S3 API / 9000")

---
