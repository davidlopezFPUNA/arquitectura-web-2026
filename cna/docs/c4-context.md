# C4 - Nivel 1: Diagrama de Contexto (F5-E5)

```mermaid
C4Context
  title Diagrama de Contexto de la Solucion (BookStack Cloud Native)

  Person(usuario, "Usuario Final", "Usuario que interactua con la plataforma para buscar, editar y exportar contenido.")
  
  System(bookstack_cna, "Sistema BookStack CNA", "Solucion Cloud Native de gestion de conocimiento distribuida.")

  System_Ext(storage_ext, "Almacenamiento de Archivos (S3/Local)", "Almacena adjuntos y archivos multimedia.")
  System_Ext(auth_ext, "Proveedor de Identidad (OIDC/OAuth2)", "Autentica usuarios y gestiona tokens.")

  Rel(usuario, bookstack_cna, "Usa la plataforma para busqueda, lectura y exportacion", "HTTPS/REST")
  Rel(bookstack_cna, storage_ext, "Lee y escribe archivos multimedia", "S3 API / File System")
  Rel(bookstack_cna, auth_ext, "Valida identidad y permisos", "HTTPS/OIDC")
