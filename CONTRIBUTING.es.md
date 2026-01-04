# Contribuir a AirInput

¡Gracias por tu interés en contribuir a AirInput! Este proyecto existe gracias a la comunidad de código abierto y nos encanta recibir ayuda, ya sea corrigiendo errores, agregando nuevas funciones o diseñando skins increíbles.

Por favor, tómate un momento para leer esta guía antes de empezar.

## Tabla de Contenidos

1. [Reportar Errores](#reportar-errores)
2. [Solicitar Funcionalidades](#solicitar-funcionalidades)
3. [Configuración de Desarrollo](#configuración-de-desarrollo)
4. [Guía para Crear Skins](#guía-para-crear-skins-nuevo)
5. [Envíar un Pull Request](#enviar-un-pull-request)
6. [Estilo de Código](#estilo-de-código)

---

## Reportar Errores

Si encuentras un bug, por favor abre un Issue en GitHub e incluye:

- Sistema Operativo del PC: por ejemplo, Windows 11, Ubuntu 22.04.
- Dispositivo móvil y navegador: por ejemplo, Pixel 6 con Chrome, iPhone 12 con Safari.
- Logs de la terminal: copia el error o traza relevante al ejecutar AirInput.
- Pasos para reproducir: describe qué hiciste justo antes de que fallara.

---

## Solicitar Funcionalidades

¿Tienes una idea genial? Abre un Issue con la etiqueta `enhancement`. Explica claramente:
* Cuál es el problema actual o la limitación.
* Cómo te gustaría que funcionara tu solución.

---

## Configuración de Desarrollo

Si vas a modificar el código fuente (`server.js`, scripts de Python o Go), sigue estos pasos para trabajar localmente sin compilar el ejecutable a cada rato.

1.  **Fork y Clona** el repositorio.
2.  **Instala las dependencias** (mira el `README.md` para los comandos exactos de tu SO).
3.  **Ejecución en modo Debug:**
    En lugar de usar el lanzador de Go, te recomendamos correr los procesos en dos terminales separadas para ver mejor los logs:

    * **Terminal 1 (Servidor):**
        ```bash
        node server.js
        ```
    * **Terminal 2 (Controlador):**
        * Windows: `python controller-win.py`
        * Linux: `sudo python3 controller-linux.py`

---

## Guía para Crear Skins

¡Nos encantan los nuevos diseños! Si quieres agregar un layout (ej. GameCube, N64, PlayStation), no necesitas saber programación compleja, solo HTML y CSS.

1.  Ve a la carpeta `public/skins/`.
2.  Crea una carpeta con el nombre de tu skin (ej. `gamecube`).
3.  Crea dos archivos dentro:
    * `layout.html`: La estructura de los botones.
    * `style.css`: Los colores y posiciones.

### Reglas para Skins
* Usa el atributo `data-btn` para conectar los botones.
    * Valores aceptados: `A, B, X, Y, L, R, START, SELECT, UP, DOWN, LEFT, RIGHT`.
* Para los joysticks, usa `id="stick-left-zone"` y `id="stick-right-zone"`.

Ejemplo básico de `layout.html`:
```html
<div id="gamepad" class="layout-mi-skin">
    <button data-btn="A">A</button>
    <div id="stick-left-zone" class="stick-zone"></div>
</div>
```

---

## Enviar un Pull Request

1. Haz un **Fork** del repositorio.
2. Crea una rama (branch) para tu cambio:
```bash
git checkout -b feature/nuevo-skin-gamecube
# o
git checkout -b fix/error-conexion-linux
```


3. Haz tus cambios y haz **Commit**:
```bash
git commit -m "Add: Nuevo skin estilo GameCube"
```


4. Haz **Push** a tu fork:
```bash
git push origin feature/nuevo-skin-gamecube
```


5. Abre un **Pull Request** en este repositorio describiendo tus cambios.

---

## Estilo de Código

- Mantén el código claro, legible y consistente con el estilo existente.
- Añade comentarios solo cuando la lógica lo requiera.
- Prefiere nombres descriptivos para variables, funciones y archivos.