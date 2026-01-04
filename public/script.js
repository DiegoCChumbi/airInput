const socket = io();

let activeButtons = new Set();
let activeJoysticks = [];

// ==========================================
// 1. DINAMIC SKIN LOADER
// ==========================================
async function loadSkin(skinName) {
  const container = document.getElementById('gamepad-container');
  const styleLink = document.getElementById('skin-style');
  const modal = document.getElementById('settings-modal');

  try {
    const response = await fetch(`skins/${skinName}/layout.html`);
    if (!response.ok) throw new Error("Skin not found");
    const html = await response.text();

    container.innerHTML = html;
    styleLink.href = `skins/${skinName}/style.css`;

    localStorage.setItem('gamepad_skin', skinName);
    if (modal) modal.style.display = 'none';

    setTimeout(() => {
      initJoysticks();
    }, 50);

  } catch (e) {
    console.error("Error loading skin:", e);
    container.innerHTML = `<h2 style="color:white; text-align:center;">Error loading skin: ${skinName}</h2>`;
  }
}

// ==========================================
// 2. BUTTON LOGIC
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
// 3. TOUCH SCANNING MOTOR
// ==========================================
function scanGamePad(e) {
  if (e.target.id === 'btn-fullscreen' ||
    e.target.id === 'btn-settings' ||
    e.target.closest('#settings-modal') ||
    e.target.closest('.stick-zone')) {
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
// 4. JOYSTICK LOGIC (Nipple.js)
// ==========================================
function initJoysticks() {
  activeJoysticks.forEach(j => j.destroy());
  activeJoysticks = [];

  const options = {
    mode: 'static',
    position: { left: '50%', top: '50%' },
    color: 'white',
    size: 90
  };

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
// 5. INITIALIZATION
// ==========================================
document.addEventListener("DOMContentLoaded", () => {
  console.log("Script loaded. Starting system...");

  document.addEventListener("touchstart", scanGamePad, { passive: false });
  document.addEventListener("touchmove", scanGamePad, { passive: false });
  document.addEventListener("touchend", scanGamePad, { passive: false });
  document.addEventListener("touchcancel", scanGamePad, { passive: false });
  document.addEventListener("contextmenu", e => { e.preventDefault(); return false; });
  window.addEventListener('resize', () => { setTimeout(initJoysticks, 200); });

  const modal = document.getElementById('settings-modal');
  const btnSettings = document.getElementById('btn-settings');
  const btnClose = document.getElementById('close-settings');
  const startOverlay = document.getElementById('start-overlay');
  const btnFullscreen = document.getElementById('btn-fullscreen');

  if (btnFullscreen) {
    btnFullscreen.addEventListener('click', () => {
      const elem = document.documentElement;
      if (elem.requestFullscreen) elem.requestFullscreen().catch(err => console.log(err));
      else if (elem.webkitRequestFullscreen) elem.webkitRequestFullscreen();
      startOverlay.style.display = 'none';
    });
  }

  btnSettings.addEventListener('click', (e) => {
    e.stopPropagation();
    modal.style.display = 'block';
  });

  btnClose.addEventListener('click', () => {
    modal.style.display = 'none';
  });

  const savedSkin = localStorage.getItem('gamepad_skin') || 'snes';

  loadSkin(savedSkin);
});
