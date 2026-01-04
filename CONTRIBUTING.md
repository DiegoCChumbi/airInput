# Contributing to AirInput

Thank you for your interest in contributing to AirInput! This project exists thanks to the open source community and we love receiving help, whether it's fixing bugs, adding new features, or designing amazing skins.

Please take a moment to read this guide before getting started.

## Table of Contents

1. [Reporting Bugs](#reporting-bugs)
2. [Requesting Features](#requesting-features)
3. [Development Setup](#development-setup)
4. [Skin Creation Guide](#skin-creation-guide-new)
5. [Submitting a Pull Request](#submitting-a-pull-request)
6. [Code Style](#code-style)

---

## Reporting Bugs

If you find a bug, please open an Issue on GitHub and include:

- PC Operating System: e.g., Windows 11, Ubuntu 22.04.
- Mobile device and browser: e.g., Pixel 6 with Chrome, iPhone 12 with Safari.
- Terminal logs: copy the relevant error or trace when running AirInput.
- Steps to reproduce: describe what you did just before it failed.

---

## Requesting Features

Have a great idea? Open an Issue with the `enhancement` label. Clearly explain:
* What the current problem or limitation is.
* How you would like your solution to work.

---

## Development Setup

If you're going to modify the source code (`server.js`, Python or Go scripts), follow these steps to work locally without compiling the executable every time.

1.  **Fork and Clone** the repository.
2.  **Install dependencies** (see `README.md` for the exact commands for your OS).
3.  **Running in Debug mode:**
    Instead of using the Go launcher, we recommend running the processes in two separate terminals to better see the logs:

    * **Terminal 1 (Server):**
        ```bash
        node server.js
        ```
    * **Terminal 2 (Controller):**
        * Windows: `python controller-win.py`
        * Linux: `sudo python3 controller-linux.py`

---

## Skin Creation Guide

We love new layouts! If you want to add a layout (e.g. GameCube, N64, PlayStation), you don't need to know complex programming, just HTML and CSS.

1.  Go to the `public/skins/` folder.
2.  Create a folder with your skin's name (e.g. `gamecube`).
3.  Create two files inside:
    * `layout.html`: The button structure.
    * `style.css`: The colors and positions.

### Skin Rules
* Use the `data-btn` attribute to connect buttons.
    * Accepted values: `A, B, X, Y, L, R, START, SELECT, UP, DOWN, LEFT, RIGHT`.
* For joysticks, use `id="stick-left-zone"` and `id="stick-right-zone"`.

Basic example of `layout.html`:
```html
<div id="gamepad" class="layout-my-skin">
    <button data-btn="A">A</button>
    <div id="stick-left-zone" class="stick-zone"></div>
</div>
```

---

## Submitting a Pull Request

1. **Fork** the repository.
2. Create a branch for your change:
```bash
git checkout -b feature/new-gamecube-skin
# or
git checkout -b fix/linux-connection-error
```


3. Make your changes and **Commit**:
```bash
git commit -m "Add: New GameCube style skin"
```


4. **Push** to your fork:
```bash
git push origin feature/new-gamecube-skin
```


5. Open a **Pull Request** in this repository describing your changes.

---

## Code Style

- Keep the code clear, readable and consistent with the existing style.
- Add comments only when the logic requires it.
- Prefer descriptive names for variables, functions and files.
