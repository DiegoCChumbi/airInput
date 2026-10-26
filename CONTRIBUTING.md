**Español**: Si prefieres leer la guía de contribución en español, haz clic [aquí](CONTRIBUTING.es.md).

# Contributing to AirInput

Thank you for your interest in contributing to AirInput! This project exists thanks to the open source community and we love receiving help, whether it's fixing bugs, adding new features, or designing new controller skins.

Please take a moment to read this guide before getting started.

## Table of Contents

1. [Reporting Bugs](#reporting-bugs)
2. [Requesting Features](#requesting-features)
3. [Development Setup](#development-setup)
4. [Skin Creation Guide](#skin-creation-guide)
5. [Submitting a Pull Request](#submitting-a-pull-request)
6. [Code Style](#code-style)

---

## Reporting Bugs

If you find a bug, please open an Issue on GitHub and include:

- **PC Operating System**: e.g., Windows 11, Ubuntu 24.04, Arch Linux.
- **Mobile device and browser**: e.g., Pixel 7 with Chrome, iPhone 15 with Safari.
- **Terminal logs**: run AirInput with `-l` (or press `l` in the TUI) and copy the relevant error or trace.
- **Steps to reproduce**: describe what you did just before it failed.

---

## Requesting Features

Have a great idea? Open an Issue with the `enhancement` label. Clearly explain:
- What the current problem or limitation is.
- How you would like your solution to work.

---

## Development Setup

AirInput compiles into a single self-contained Go binary with the `public/` folder embedded via `//go:embed`. When developing or designing skins, you can use the `-dev` flag to serve `public/` directly from disk without recompiling on every HTML/CSS/JS edit:

1. **Fork and Clone** the repository.
2. **Build and run in development mode**:
   ```bash
   go build -o airInput .
   sudo ./airInput -dev -l
   ```
   - `-dev`: Serves `./public` directly from disk so any changes to `public/skins/<skin>/layout.html` or `style.css` apply immediately when switching skins or refreshing the mobile browser.
   - `-l`: Enables the live input logs panel in the terminal UI.
3. **Run tests**:
   ```bash
   go test -v ./...
   ```

---

## Skin Creation Guide

If you want to add a new controller layout (e.g. GameCube, PlayStation, Saturn), you only need HTML and CSS:

1. Go to the `public/skins/` folder.
2. Create a folder with your skin's identifier (e.g. `public/skins/gamecube/`).
3. Create two files inside:
   - `layout.html`: The button and stick structure.
   - `style.css`: The visual styling and positioning.
4. Register the new skin button in `public/index.html` inside both `.layout-selector` blocks (`selectInitialSkin('gamecube')` on the join screen and `loadSkin('gamecube')` in the in-game menu).

### Skin Rules

- **Single Buttons (`data-btn`)**:
  - Accepted values: `A, B, X, Y, L, R, L2, R2, L3, R3, START, SELECT, UP, DOWN, LEFT, RIGHT`.
- **Multi-Button Combos & Diagonals (`data-btns`)**:
  - Pass a comma-separated list of buttons to trigger simultaneously when touched: e.g. `data-btns="UP,LEFT"` for D-Pad diagonals or `data-btns="A,R"` for genre macro buttons.
- **Analog Sticks (`#stick-left-zone` and `#stick-right-zone`)**:
  - Use `<div id="stick-left-zone" class="stick-zone" data-floating="true"></div>` and/or `<div id="stick-right-zone" class="stick-zone" data-floating="true"></div>`.
  - **Dynamic Stick Positioning**: By default, every `.stick-zone` supports dynamic touch positioning (`data-floating="true"`), where the stick origin (`.nipple`) automatically centers at the user's initial touch point inside the `.stick-zone` area (unless the player disables dynamic sticks in the in-game menu).
  - Give `.stick-zone` a generous width/height (e.g. `135px`–`160px` or a full touchpad well) and `position: relative` so the user has room to place their thumb comfortably.
  - If a specific skin requires a strictly fixed stick that never moves its center even when dynamic sticks are enabled, set `data-floating="false"` on its `.stick-zone`.

Basic example of `layout.html`:
```html
<div id="gamepad" class="layout-my-skin">
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

## Submitting a Pull Request

1. **Fork** the repository.
2. Create a branch for your change:
   ```bash
   git checkout -b feat/new-gamecube-skin
   ```
3. Make your changes, verify `go test -v ./...`, and **Commit**:
   ```bash
   git commit -m "feat: add GameCube controller skin"
   ```
4. **Push** to your fork:
   ```bash
   git push origin feat/new-gamecube-skin
   ```
5. Open a **Pull Request** describing your changes and including a screenshot if it modifies the UI.

---

## Code Style

- Keep code clear, readable, and consistent with the existing style (`gofmt` for Go files).
- Add comments only when the logic requires explanation.
- Prefer descriptive names for variables, functions, and CSS classes (scope skin styles under your root class, e.g. `.layout-my-skin`).
