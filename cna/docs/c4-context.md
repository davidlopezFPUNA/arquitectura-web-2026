# C4 - Nivel 1: Diagrama de Contexto (F5-E5)

```mermaid
C4Context
  title Diagrama de Contexto de la Solucion (BookStack Cloud Native)

  UpdateLayoutConfig($c4ShapeInRow="2", $c4BoundaryInRow="1")

  Person(usuario, "Usuario Final", "Usuario de la plataforma")
  
  System(bookstack_cna, "Sistema BookStack CNA", "Plataforma de gestion de conocimiento")

  System_Ext(storage_ext, "Almacenamiento de Archivos", "Almacena multimedia (S3)")
  System_Ext(auth_ext, "Proveedor de Identidad", "Autentica usuarios (OIDC)")

  Rel(usuario, bookstack_cna, "Usa la plataforma", "HTTPS/REST")
  Rel(bookstack_cna, storage_ext, "Lee y escribe multimedia", "S3 API")
  Rel(bookstack_cna, auth_ext, "Valida identidad", "HTTPS/OIDC")
