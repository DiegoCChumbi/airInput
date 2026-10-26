**English**: Read the contribution guide in English [here](CONTRIBUTING.md).

# Contribuir a AirInput

¡Gracias por tu interés en contribuir a AirInput! Este proyecto existe gracias a la comunidad de código abierto y nos encanta recibir ayuda, ya sea corrigiendo errores, agregando nuevas funciones o diseñando nuevos skins de controles.

Por favor, tómate un momento para leer esta guía antes de empezar.

## Tabla de Contenidos

1. [Reportar Errores](#reportar-errores)
2. [Solicitar Funcionalidades](#solicitar-funcionalidades)
3. [Configuración de Desarrollo](#configuración-de-desarrollo)
4. [Guía para Crear Skins](#guía-para-crear-skins)
5. [Enviar un Pull Request](#enviar-un-pull-request)
6. [Estilo de Código](#estilo-de-código)

---

## Reportar Errores

Si encuentras un bug, por favor abre un Issue en GitHub e incluye:

- **Sistema Operativo del PC**: por ejemplo, Windows 11, Ubuntu 24.04, Arch Linux.
- **Dispositivo móvil y navegador**: por ejemplo, Pixel 7 con Chrome, iPhone 15 con Safari.
- **Logs de la terminal**: ejecuta AirInput con `-l` (o presiona `l` en la TUI) y copia el error o traza relevante.
- **Pasos para reproducir**: describe qué hiciste justo antes de que fallara.

---

## Solicitar Funcionalidades

¿Tienes una idea genial? Abre un Issue con la etiqueta `enhancement`. Explica claramente:
- Cuál es el problema actual o la limitación.
- Cómo te gustaría que funcionara tu solución.

---

## Configuración de Desarrollo

AirInput se compila en un único binario autocontenido en Go con la carpeta `public/` embebida mediante `//go:embed`. Para trabajar en la interfaz o diseñar nuevos skins sin recompilar el binario en cada cambio de HTML/CSS/JS, usa el flag `-dev`:

1. **Haz Fork y Clona** el repositorio.
2. **Compila y ejecuta en modo desarrollo**:
   ```bash
   go build -o airInput .
   sudo ./airInput -dev -l
   ```
   - `-dev`: Sirve `./public` directamente desde el disco para que cualquier cambio en `public/skins/<skin>/layout.html` o `style.css` se refleje al instante al cambiar de carcasa o recargar el navegador móvil.
   - `-l`: Activa el panel de logs en vivo dentro de la interfaz de terminal (TUI).
3. **Ejecuta los tests**:
   ```bash
   go test -v ./...
   ```

---

## Guía para Crear Skins

Si quieres agregar un nuevo diseño de mando (ej. GameCube, PlayStation, Saturn), solo necesitas HTML y CSS:

1. Ve a la carpeta `public/skins/`.
2. Crea una carpeta con el identificador de tu skin (ej. `public/skins/gamecube/`).
3. Crea dos archivos dentro:
   - `layout.html`: La estructura de botones y sticks.
   - `style.css`: Los colores, dimensiones y posiciones.
4. Registra el botón del nuevo skin en `public/index.html` dentro de ambos bloques `.layout-selector` (`selectInitialSkin('gamecube')` en la pantalla de conexión y `loadSkin('gamecube')` en el menú en partida).

### Reglas para Skins

- **Botones Individuales (`data-btn`)**:
  - Valores aceptados: `A, B, X, Y, L, R, L2, R2, L3, R3, START, SELECT, UP, DOWN, LEFT, RIGHT`.
- **Botones Combinados y Diagonales (`data-btns`)**:
  - Permite activar múltiples botones simultáneamente separados por coma: por ejemplo `data-btns="UP,LEFT"` para las diagonales de la cruceta o `data-btns="A,R"` para combos/macros.
- **Sticks Analógicos (`#stick-left-zone` y `#stick-right-zone`)**:
  - Usa `<div id="stick-left-zone" class="stick-zone" data-floating="true"></div>` y/o `<div id="stick-right-zone" class="stick-zone" data-floating="true"></div>`.
  - **Posicionamiento Dinámico**: Por defecto, todo `.stick-zone` tiene posicionamiento dinámico habilitado (`data-floating="true"`), por lo que la base del stick (`.nipple`) se centra automáticamente en el punto del primer toque del pulgar dentro del área `.stick-zone` (salvo que el jugador desactive la opción en el menú del mando).
  - Asigna a `.stick-zone` un área táctil amplia (por ejemplo `135px`–`160px` o un panel completo) y `position: relative` para que el pulgar tenga un margen cómodo de toque.
  - Si un diseño específico requiere que el stick permanezca siempre fijo en el centro sin importar el ajuste global, puedes definir `data-floating="false"` en su `.stick-zone`.

Ejemplo básico de `layout.html`:
```html
<div id="gamepad" class="layout-mi-skin">
  <div class="shoulders">
    <button data-btn="L">L</button>
    <button data-btn="R">R</button>
  </div>

  <div class="face">
    <div id="stick-left-zone" class="stick-zone" data-floating="true"></div>

    <div class="actions">
      <button data-btn="A">A</button>
      <button data-btn="B">B</button>
      <button data-btns="A,B">A+B</button>
    </div>

    <div id="stick-right-zone" class="stick-zone" data-floating="true"></div>
  </div>
</div>
```

---

## Enviar un Pull Request

1. Haz un **Fork** del repositorio.
2. Crea una rama (`branch`) para tu cambio:
   ```bash
   git checkout -b feat/nuevo-skin-gamecube
   ```
3. Haz tus cambios, verifica `go test -v ./...` y haz **Commit**:
   ```bash
   git commit -m "feat: agregar skin estilo GameCube"
   ```
4. Haz **Push** a tu fork:
   ```bash
   git push origin feat/nuevo-skin-gamecube
   ```
5. Abre un **Pull Request** en este repositorio describiendo tus cambios e incluyendo una captura de pantalla si modificaste o agregaste un diseño.

---

## Estilo de Código

- Mantén el código claro, legible y consistente con el estilo existente (`gofmt` para archivos Go).
- Añade comentarios solo cuando la lógica lo requiera.
- Prefiere nombres descriptivos para variables, funciones y clases CSS (encapsula todos los estilos de tu skin bajo su clase raíz, ej. `.layout-mi-skin`).
