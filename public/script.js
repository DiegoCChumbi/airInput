// ==========================================
// 1. SETUP & DOM ELEMENTS
// ==========================================
const socket = io({ autoConnect: false });

const joinScreen = document.getElementById('join-screen');
const usernameInput = document.getElementById('username-input');
const joinBtn = document.getElementById('join-btn');

const gamepadContainer = document.getElementById('gamepad-container');
const btnSettings = document.getElementById('btn-settings');
const settingsModal = document.getElementById('settings-modal');
const btnCloseSettings = document.getElementById('close-settings');

let activeButtons = new Set();
let activeJoysticks = [];

// ==========================================
// 2. DINAMIC SKIN LOADER
// ==========================================
async function loadSkin(skinName) {
  try {
    const response = await fetch(`skins/${skinName}/layout.html`);
    if (!response.ok) throw new Error("Skin not found");

    const html = await response.text();
    gamepadContainer.innerHTML = html;

    const styleLink = document.getElementById('skin-style') || document.createElement('link');
    styleLink.id = 'skin-style';
    styleLink.rel = 'stylesheet';
    styleLink.href = `skins/${skinName}/style.css`;
    document.head.appendChild(styleLink);

    localStorage.setItem('gamepad_skin', skinName);
    if (settingsModal) settingsModal.style.display = 'none';

    setTimeout(initJoysticks, 50);

  } catch (e) {
    console.error("Error loading skin:", e);
    gamepadContainer.innerHTML = `<h2 style="color:white; text-align:center;">Error loading skin: ${skinName}</h2>`;
  }
}

// ==========================================
// 3. GAMEPAD LOGIC (Buttons & Joysticks)
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

function scanGamePad(e) {
  if (e.target.closest('#btn-settings') || e.target.closest('#settings-modal') || e.target.closest('.stick-zone')) {
    return;
  }

  if (e.type !== 'click') e.preventDefault();

  const buttonsBeingTouched = new Set();
  for (let i = 0; i < e.touches.length; i++) {
    const touch = e.touches[i];
    const element = document.elementFromPoint(touch.clientX, touch.clientY);
    if (element) {
      const btn = element.closest('button');
      if (btn) {
        const rawData = btn.dataset.btns || btn.dataset.btn;
        if (rawData) {
          rawData.split(',').forEach(t => buttonsBeingTouched.add(t.trim()));
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

function initJoysticks() {
  activeJoysticks.forEach(j => j.destroy());
  activeJoysticks = [];

  const processJoystickData = (data) => {
    if (!data.vector) return { x: 0, y: 0 };
    let x = data.vector.x;
    let y = -data.vector.y;

    if (window.innerHeight > window.innerWidth) {
      const tempX = x;
      x = y;
      y = -tempX;
    }
    return { x, y };
  };

  const createJoystick = (zoneId, onMove, onEnd) => {
    const zone = document.getElementById(zoneId);
    if (zone) {
      const joystick = nipplejs.create({
        zone,
        mode: 'static',
        position: { left: '50%', top: '50%' },
        color: 'white',
        size: 100
      });
      joystick.on('move', (evt, data) => onMove(processJoystickData(data)));
      joystick.on('end', onEnd);
      activeJoysticks.push(joystick);
    }
  };

  createJoystick('stick-left-zone',
    ({ x, y }) => {
      socket.emit("axis", { axis: 'lx', value: x });
      socket.emit("axis", { axis: 'ly', value: y });
    },
    () => {
      socket.emit("axis", { axis: 'lx', value: 0 });
      socket.emit("axis", { axis: 'ly', value: 0 });
    }
  );

  createJoystick('stick-right-zone',
    ({ x, y }) => {
      socket.emit("axis", { axis: 'rx', value: x });
      socket.emit("axis", { axis: 'ry', value: y });
    },
    () => {
      socket.emit("axis", { axis: 'rx', value: 0 });
      socket.emit("axis", { axis: 'ry', value: 0 });
    }
  );
}

// ==========================================
// 4. INITIALIZATION & JOIN LOGIC
// ==========================================
function initializeGamepad() {
  gamepadContainer.style.display = 'block';
  btnSettings.style.display = 'block';
  joinScreen.style.display = 'none';

  document.addEventListener("touchstart", scanGamePad, { passive: false });
  document.addEventListener("touchmove", scanGamePad, { passive: false });
  document.addEventListener("touchend", scanGamePad, { passive: false });
  document.addEventListener("touchcancel", scanGamePad, { passive: false });
  document.addEventListener("contextmenu", e => e.preventDefault());
  window.addEventListener('resize', () => setTimeout(initJoysticks, 200));

  btnSettings.addEventListener('click', (e) => {
    e.stopPropagation();
    settingsModal.style.display = 'block';
  });

  btnCloseSettings.addEventListener('click', () => {
    settingsModal.style.display = 'none';
  });

  // Request fullscreen on first interaction
  const requestFullscreen = () => {
    const elem = document.documentElement;
    if (elem.requestFullscreen) elem.requestFullscreen().catch(err => console.log(err));
    else if (elem.webkitRequestFullscreen) elem.webkitRequestFullscreen();
    document.body.removeEventListener('touchstart', requestFullscreen);
  };
  document.body.addEventListener('touchstart', requestFullscreen);


  const savedSkin = localStorage.getItem('gamepad_skin') || 'snes';
  loadSkin(savedSkin);
}


document.addEventListener("DOMContentLoaded", () => {
  // Restore username if exists
  const savedUsername = localStorage.getItem('airinput_username');
  if (savedUsername) {
    usernameInput.value = savedUsername;
  }
  
  // Connect to server
  socket.connect();

  joinBtn.addEventListener('click', () => {
    const username = usernameInput.value.trim();
    if (username) {
      localStorage.setItem('airinput_username', username);
      socket.emit('register_player', { username });
    } else {
      alert('Por favor, ingresa un nombre.');
    }
  });

  socket.on('registration_success', () => {
    console.log('Registration successful. Initializing gamepad...');
    initializeGamepad();
  });

  socket.on('registration_failed', (message) => {
    alert(`Error: ${message}`);
  });

  // Handle disconnection
  socket.on('disconnect', () => {
    alert('Desconectado del servidor. Por favor, recarga la página.');
    gamepadContainer.style.display = 'none';
    btnSettings.style.display = 'none';
    joinScreen.style.display = 'flex';
  });
});
