## 4.1. A1 – Análisis del Monolito y Mapa de Contextos

### Inventario de Módulos Funcionales y Tablas (Evidencia F5-E1)

Se inspeccionó la base de datos embebida SQLite de PocketBase (`data.db`) ejecutando la consulta sobre la tabla maestra `sqlite_master`:

<img width="1113" height="149" alt="image" src="https://github.com/user-attachments/assets/d34b35e1-c2bc-4500-bd2d-83e474527ada" />

---

### Evidencia F5-E2: Matriz de Acoplamiento

A continuación se detalla la matriz de dependencia entre módulos funcionales y tablas del motor SQLite, identificando las tablas compartidas que representan acoplamiento directo en la arquitectura actual:

| Tabla / Módulo | Usuarios / Permisos | Colecciones / Contenido | Parámetros Globales | Migraciones | ¿Tabla Compartida / Acoplada? |
| :--- | :---: | :---: | :---: | :---: | :---: |
| `_admins` | X | | | | No |
| `users` | X | X (Relación con contactos) | | | Sí (Acoplada) |
| `_collections` | | X | | | No |
| `contactos` | | X | | | No |
| `_externalAuths` | X | | | | No |
| `_params` | X | X | X | | Sí (Global) |
| `_migrations` | | | | X | No |

> **Nota de Acoplamiento:** Las tablas `users` y `_params` actúan como puntos de acoplamiento fuerte en el monolito, ya que la gestión de datos (`contactos`) depende directamente del modelo de usuarios y las configuraciones globales del sistema.

---

### Mapa de Contextos / DDD (Evidencia F5-E3)

El diseño del Mapa de Contextos (Bounded Contexts) bajo Domain-Driven Design para el desacoplamiento del monolito se estructura de la siguiente manera:

+------------------------------------+          +------------------------------------+
|   Contexto: Autenticación / User   |          |      Contexto: Almacenamiento      |
|              (Auth-SVC)            |          |           (Storage-SVC)            |
+------------------------------------+          +------------------------------------+
^                                       ^
| (Cliente / Proveedor)                 | (ACL / Permisos)
v                                       v
+------------------------------------------------------------------------------------+
|                        Contexto Core: Datos / Contactos                            |
|                                  (PocketBase Core)                                 |
+------------------------------------------------------------------------------------+

---

### 5.3. B3 – Contenerización con Buenas Prácticas

En esta sección se rediseñó la estrategia de contenerización de los microservicios aplicando un enfoque de mínima superficie de ataque y optimización de recursos, cumpliendo con los siguientes criterios:

*   **Dockerfile Multi-stage:** Se implementó un *builder* temporal para compilar dependencias (`wheels`), dejando una imagen final `python:3.11-slim` limpia y ligera.
*   **Seguridad:** Ejecución con el usuario sin privilegios `appuser` (no-root) y sistema de archivos montado como solo lectura (`read_only: true`), utilizando `/tmp` para escrituras efímeras.
*   **Tolerancia a fallos:** Incorporación de la instrucción `HEALTHCHECK` en Dockerfile, sondeando el endpoint `/healthz` de forma nativa.
*   **Versionamiento:** Despliegue seguro prohibiendo la etiqueta `:latest`. Se utilizó versionamiento semántico ligado al hash del commit (Ej: `1.0.0-eba4c0b`).
*   **Límites (Cgroups):** Contenedores restringidos por diseño a `0.25` CPUs y `128m` de memoria RAM, garantizando que el sumatorio (512MB RAM) no sature el nodo.

<details open>
<summary>📸 <b>Evidencia F6-E9: Tabla de Imágenes y Tamaño</b></summary>
<br>

Listado de los microservicios compilados, mostrando el uso estricto de versionamiento semántico y la reducción de tamaño de las imágenes.

![Evidencia F6-E9](./evidencias/F6-E9-out1.png)

</details>

<details open>
<summary>📸 <b>Evidencia F6-E10: Inspección de Seguridad y Límites (docker inspect)</b></summary>
<br>

Validación mediante `docker inspect` al contenedor `search-svc` demostrando: usuario `appuser`, límite de RAM en bytes (128MB), límite de CPU (0.25), `ReadonlyRootfs: true` y la definición explícita del `HEALTHCHECK`.

![Evidencia F6-E10](./evidencias/F6-E10-out1.png)

</details>
