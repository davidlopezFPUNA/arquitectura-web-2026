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

---

<Steps>
  <Step subtitle="< 1 min" title="Guardar el archivo">
    Abre o crea el archivo desde la carpeta `cna/docs` ejecutando:
    ```bash
    nano c4-context.md
    ```
    Pega el bloque anterior, presiona `Ctrl + O`, `Enter` para confirmar y `Ctrl + X` para salir.
    *Verification:* Ejecuta `cat c4-context.md` en la terminal para asegurarte de que todo el texto y el bloque de Mermaid estén continuos.
  </Step>
</Steps>

<ElicitationsGroup message="¿Pudiste guardar el archivo correctamente?">
  <Elicitation label="Guardado exitoso" query="Ya guardé el c4-context.md sin problemas. ¿Cómo seguimos con el C4 Nivel 2 de Contenedores (F5-E6)?"/>
  <Elicitation label="Subir a GitHub" query="Quiero hacer git add, commit y push de este archivo antes de pasar al siguiente diagrama."/>
</ElicitationsGroup>
