const socket = io();

let activeButtons = new Set();
let activeJoysticks = [];
// ==========================================
// 1. LÓGICA DE BOTONES (Digitales)
// ==========================================
function updateButton(btnName, state) {
  const cleanName = btnName.trim();

  socket.emit("input", { button: cleanName, state: state });

  const el = document.querySelector(`button[data-btn="${cleanName}"]`);
  if (el) {
    if (state === 1) el.classList.add("active");
    else el.classList.remove("active");
  }

  if (state === 1 && navigator.vibrate) navigator.vibrate(30);
}

// ==========================================
// 2. MOTOR DE ESCANEO TÁCTIL (Raycasting)
// ==========================================
function scanGamePad(e) {
  // LISTA BLANCA: Elementos que NO deben ser bloqueados por el escaneo
  if (e.target.id === 'btn-fullscreen' ||
    e.target.id === 'btn-settings' ||
    e.target.closest('#settings-modal') ||
    e.target.closest('.stick-zone')) { // <--- ¡NUEVO! Ignora las zonas de joystick
    return;
  }

  if (e.type !== 'click') {
    e.preventDefault();
  }

  const buttonsBeingTouched = new Set();

  for (let i = 0; i < e.touches.length; i++) {
    const touch = e.touches[i];
    const element = document.elementFromPoint(touch.clientX, touch.clientY);

    if (element) {
      const btn = element.closest('button');
      if (btn) {
        const rawData = btn.dataset.btns || btn.dataset.btn;
        if (rawData) {
          const targets = rawData.split(',');
          targets.forEach(t => buttonsBeingTouched.add(t.trim()));
        }
      }
    }
  }

  activeButtons.forEach(btnName => {
    if (!buttonsBeingTouched.has(btnName)) {
      updateButton(btnName, 0);
    }
  });

  buttonsBeingTouched.forEach(btnName => {
    if (!activeButtons.has(btnName)) {
      updateButton(btnName, 1);
    }
  });

  activeButtons = buttonsBeingTouched;
}

// ==========================================
// 3. LÓGICA DE JOYSTICKS (Nipple.js)
// ==========================================
function initJoysticks() {
  // 1. Si ya existen, los destruimos primero para limpiar memoria y eventos viejos
  activeJoysticks.forEach(j => j.destroy());
  activeJoysticks = [];

  const options = {
    mode: 'static',
    position: { left: '50%', top: '50%' },
    color: 'white',
    size: 90 // Un poco más pequeños para que no estorben
  };

  // --- STICK IZQUIERDO ---
  const zoneLeft = document.getElementById('stick-left-zone');
  if (zoneLeft) {
    const joyLeft = nipplejs.create({ zone: zoneLeft, ...options });

    joyLeft.on('move', (evt, data) => {
      if (data.vector) {
        socket.emit("axis", { axis: 'lx', value: data.vector.x });
        socket.emit("axis", { axis: 'ly', value: -data.vector.y });
      }
    });

    joyLeft.on('end', () => {
      socket.emit("axis", { axis: 'lx', value: 0 });
      socket.emit("axis", { axis: 'ly', value: 0 });
    });

    activeJoysticks.push(joyLeft);
  }

  // --- STICK DERECHO ---
  const zoneRight = document.getElementById('stick-right-zone');
  if (zoneRight) {
    const joyRight = nipplejs.create({ zone: zoneRight, ...options });

    joyRight.on('move', (evt, data) => {
      if (data.vector) {
        socket.emit("axis", { axis: 'rx', value: data.vector.x });
        socket.emit("axis", { axis: 'ry', value: -data.vector.y });
      }
    });

    joyRight.on('end', () => {
      socket.emit("axis", { axis: 'rx', value: 0 });
      socket.emit("axis", { axis: 'ry', value: 0 });
    });

    activeJoysticks.push(joyRight);
  }
}

// ==========================================
// 4. INICIALIZACIÓN
// ==========================================
document.addEventListener("DOMContentLoaded", () => {
  console.log("Script cargado y listo.");

  // Inicializar Joysticks
  initJoysticks();

  // Listeners Globales
  document.addEventListener("touchstart", scanGamePad, { passive: false });
  document.addEventListener("touchmove", scanGamePad, { passive: false });
  document.addEventListener("touchend", scanGamePad, { passive: false });
  document.addEventListener("touchcancel", scanGamePad, { passive: false });

  document.addEventListener("contextmenu", e => { e.preventDefault(); return false; });

  // Variables UI
  const gamepad = document.getElementById('gamepad');
  const modal = document.getElementById('settings-modal');
  const btnSettings = document.getElementById('btn-settings');
  const btnClose = document.getElementById('close-settings');
  const startOverlay = document.getElementById('start-overlay');
  const btnFullscreen = document.getElementById('btn-fullscreen');

  // Fullscreen
  if (btnFullscreen) {
    btnFullscreen.addEventListener('click', () => {
      const elem = document.documentElement;
      if (elem.requestFullscreen) elem.requestFullscreen().catch(err => console.log(err));
      else if (elem.webkitRequestFullscreen) elem.webkitRequestFullscreen();
      startOverlay.style.display = 'none';
    });
  }

  // Modal Settings
  btnSettings.addEventListener('click', (e) => {
    e.stopPropagation();
    modal.style.display = 'block';
  });

  btnClose.addEventListener('click', () => {
    modal.style.display = 'none';
  });

  // Cambio de Layout
  window.changeLayout = function (type) {
    gamepad.classList.remove('layout-snes', 'layout-ps4');
    gamepad.classList.add(`layout-${type}`);

    document.querySelectorAll('.sel-btn').forEach(b => b.classList.remove('active'));
    localStorage.setItem('gamepad_skin', type);

    setTimeout(() => {
      initJoysticks();
    }, 100);
  }

  window.addEventListener('resize', () => {
    setTimeout(initJoysticks, 200);
  });


  // Cargar Preferencias
  const savedSkin = localStorage.getItem('gamepad_skin');
  if (savedSkin) {
    window.changeLayout(savedSkin);
  } else {
    gamepad.classList.add('layout-snes');
  }

});
