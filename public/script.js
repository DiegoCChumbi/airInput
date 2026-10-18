// ==========================================
// 1. SETUP, STATE & DOM ELEMENTS
// ==========================================
const socket = io({ autoConnect: false, reconnection: true, reconnectionDelay: 500 });

const joinScreen = document.getElementById('join-screen');
const usernameInput = document.getElementById('username-input');
const joinBtn = document.getElementById('join-btn');
const joinFeedback = document.getElementById('join-feedback');
const joinLed = document.getElementById('join-led');

const gamepadContainer = document.getElementById('gamepad-container');
const skinStage = document.getElementById('skin-stage');
const hudBar = document.getElementById('hud-bar');
const btnSettings = document.getElementById('btn-settings');
const statusBadge = document.getElementById('status-badge');
const statusText = document.getElementById('status-text');
const settingsModal = document.getElementById('settings-modal');
const btnCloseSettings = document.getElementById('close-settings');

const togglePortraitFlip = document.getElementById('toggle-portrait-flip');
const toggleInvertY = document.getElementById('toggle-invert-y');
const toggleHaptics = document.getElementById('toggle-haptics');
const rangeDeadzone = document.getElementById('range-deadzone');
const deadzoneVal = document.getElementById('deadzone-val');
const btnToggleFullscreen = document.getElementById('btn-toggle-fullscreen');

let activeButtons = new Set();
let activeJoysticks = [];
const activeJoystickTouchIds = new Set();
let isGamepadInitialized = false;
let currentUsername = '';
let pingInterval = null;

// User Preferences (persisted in localStorage)
const userSettings = {
  portraitFlip: localStorage.getItem('airinput_portrait_flip') === 'true',
  invertY: localStorage.getItem('airinput_invert_y') === 'true',
  haptics: localStorage.getItem('airinput_haptics') !== 'false',
  deadzone: parseInt(localStorage.getItem('airinput_deadzone') ?? '8', 10)
};

function syncSkinButtonsUI(skinName) {
  document.querySelectorAll('.layout-selector .sel-btn').forEach(btn => {
    btn.classList.toggle('active', btn.dataset.skin === skinName);
  });
}

function selectInitialSkin(skinName) {
  localStorage.setItem('gamepad_skin', skinName);
  syncSkinButtonsUI(skinName);
  if (userSettings.haptics && navigator.vibrate) {
    navigator.vibrate(15);
  }
}

function showJoinFeedback(msg) {
  if (!joinFeedback) return;
  if (!msg) {
    joinFeedback.textContent = '';
    joinFeedback.classList.remove('visible');
    return;
  }
  joinFeedback.textContent = msg;
  joinFeedback.classList.add('visible');
}

function applyUserSettingsUI() {
  document.body.classList.toggle('portrait-ccw', userSettings.portraitFlip);
  if (togglePortraitFlip) togglePortraitFlip.checked = userSettings.portraitFlip;
  if (toggleInvertY) toggleInvertY.checked = userSettings.invertY;
  if (toggleHaptics) toggleHaptics.checked = userSettings.haptics;
  if (rangeDeadzone) rangeDeadzone.value = String(userSettings.deadzone);
  if (deadzoneVal) deadzoneVal.textContent = `${userSettings.deadzone}%`;
  const savedSkin = localStorage.getItem('gamepad_skin') || 'snes';
  syncSkinButtonsUI(savedSkin);
}

// ==========================================
// 2. DYNAMIC SKIN LOADER
// ==========================================
async function loadSkin(skinName) {
  try {
    releaseAllInputs();

    const cacheBuster = Date.now();
    const response = await fetch(`skins/${skinName}/layout.html?v=${cacheBuster}`);
    if (!response.ok) throw new Error("Skin not found");

    const html = await response.text();
    const mountTarget = skinStage || gamepadContainer;
    mountTarget.innerHTML = html;

    let styleLink = document.getElementById('skin-style');
    if (!styleLink) {
      styleLink = document.createElement('link');
      styleLink.id = 'skin-style';
      styleLink.rel = 'stylesheet';
      document.head.appendChild(styleLink);
    }

    syncSkinButtonsUI(skinName);
    localStorage.setItem('gamepad_skin', skinName);
    if (settingsModal) settingsModal.style.display = 'none';

    // Wait for stylesheet to load (or fallback timeout if cached) before initializing joysticks
    await new Promise((resolve) => {
      let resolved = false;
      const done = () => {
        if (!resolved) {
          resolved = true;
          requestAnimationFrame(() => resolve());
        }
      };
      styleLink.onload = done;
      styleLink.onerror = done;
      styleLink.href = `skins/${skinName}/style.css?v=${cacheBuster}`;
      setTimeout(done, 120);
    });

    initJoysticks();

  } catch (e) {
    console.error("Error loading skin:", e);
    const mountTarget = skinStage || gamepadContainer;
    mountTarget.innerHTML = `<h2 style="color:white; text-align:center;">Error al cargar carcasa: ${skinName}</h2>`;
  }
}

// ==========================================
// 3. GAMEPAD LOGIC (Buttons & Rotation-Aware Joysticks)
// ==========================================
function syncComboButtonsVisuals() {
  document.querySelectorAll('button[data-btns]').forEach(btn => {
    const parts = (btn.dataset.btns || '').split(',').map(s => s.trim()).filter(Boolean);
    const allActive = parts.length > 0 && parts.every(p => activeButtons.has(p));
    btn.classList.toggle('active', allActive);
  });
}

function updateButton(btnName, state) {
  const cleanName = btnName.trim();
  if (!cleanName) return;

  socket.emit("input", { button: cleanName, state: state });

  document.querySelectorAll(`button[data-btn="${cleanName}"]`).forEach(el => {
    if (state === 1) el.classList.add("active");
    else el.classList.remove("active");
  });

  if (state === 1 && userSettings.haptics && navigator.vibrate) {
    navigator.vibrate(25);
  }
}

function releaseAllInputs() {
  activeButtons.forEach(btnName => updateButton(btnName, 0));
  activeButtons.clear();
  syncComboButtonsVisuals();
  activeJoysticks.forEach(j => j.reset());
}

function scanGamePad(e) {
  // Ignore interactions inside the settings modal or HUD bar
  if (e.target && (e.target.closest('#hud-bar') || e.target.closest('#btn-settings') || e.target.closest('#settings-modal'))) {
    return;
  }

  if (e.cancelable && e.type !== 'click' && !e.type.startsWith('mouse')) {
    e.preventDefault();
  }

  const buttonsBeingTouched = new Set();

  const collectFromPoint = (clientX, clientY) => {
    const element = document.elementFromPoint(clientX, clientY);
    if (!element || element.closest('.stick-zone')) {
      return;
    }
    const btn = element.closest('button');
    if (btn && btn.id !== 'btn-settings') {
      const rawData = btn.dataset.btns || btn.dataset.btn;
      if (rawData) {
        rawData.split(',').forEach(t => {
          const clean = t.trim();
          if (clean) buttonsBeingTouched.add(clean);
        });
      }
    }
  };

  if (e.touches) {
    for (let i = 0; i < e.touches.length; i++) {
      const touch = e.touches[i];
      // Skip touches that belong to an analog stick
      if (activeJoystickTouchIds.has(touch.identifier)) {
        continue;
      }
      collectFromPoint(touch.clientX, touch.clientY);
    }
  } else if (e.type.startsWith('mouse') && e.buttons === 1) {
    collectFromPoint(e.clientX, e.clientY);
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
  syncComboButtonsVisuals();
}

/**
 * Extracts the 2D rotation basis (a, b, c, d) of #gamepad-container so that any screen
 * vector (screenDx, screenDy) is accurately mapped into the controller's local coordinate
 * space regardless of whether the screen is in Landscape (0°), Portrait (+90°),
 * Portrait Flipped (-90°), or Upside-Down (180°).
 */
function getContainerRotationBasis() {
  const style = window.getComputedStyle(gamepadContainer);
  const transform = style.transform || style.webkitTransform;
  if (!transform || transform === 'none') {
    return { a: 1, b: 0, c: 0, d: 1 };
  }
  const match = transform.match(/matrix\(([^)]+)\)/);
  if (!match) {
    return { a: 1, b: 0, c: 0, d: 1 };
  }
  const parts = match[1].split(',').map(Number);
  const rawA = parts[0], rawB = parts[1], rawC = parts[2], rawD = parts[3];
  const scaleX = Math.hypot(rawA, rawB) || 1;
  const scaleY = Math.hypot(rawC, rawD) || 1;
  return {
    a: rawA / scaleX,
    b: rawB / scaleX,
    c: rawC / scaleY,
    d: rawD / scaleY
  };
}

class VirtualJoystick {
  constructor(zone, onChange) {
    this.zone = zone;
    this.onChange = onChange;
    this.maxRadius = 50;
    this.touchId = null;
    this.mouseActive = false;
    this.lastX = 0;
    this.lastY = 0;
    this.lastEmitTime = 0;

    // Force-clear any legacy counter-rotation on .stick-zone even if an old stylesheet was cached
    this.zone.style.setProperty('transform', 'none', 'important');

    // Create visual elements matching skin CSS (.nipple > .back + .front)
    this.nippleEl = document.createElement('div');
    this.nippleEl.className = 'nipple';
    this.backEl = document.createElement('div');
    this.backEl.className = 'back';
    this.frontEl = document.createElement('div');
    this.frontEl.className = 'front';

    // Invisible reference points at (+10px, 0) and (0, +10px) in local coordinates
    // to measure the exact screen-space direction of Controller Right (+X) and Controller Down (+Y)
    this.probeXEl = document.createElement('div');
    this.probeXEl.style.cssText = 'position:absolute;left:10px;top:0;width:0;height:0;pointer-events:none;';
    this.probeYEl = document.createElement('div');
    this.probeYEl.style.cssText = 'position:absolute;left:0;top:10px;width:0;height:0;pointer-events:none;';

    this.nippleEl.appendChild(this.backEl);
    this.nippleEl.appendChild(this.frontEl);
    this.nippleEl.appendChild(this.probeXEl);
    this.nippleEl.appendChild(this.probeYEl);
    this.zone.innerHTML = '';
    this.zone.appendChild(this.nippleEl);

    this.onTouchStart = this.onTouchStart.bind(this);
    this.onTouchMove = this.onTouchMove.bind(this);
    this.onTouchEnd = this.onTouchEnd.bind(this);
    this.onMouseDown = this.onMouseDown.bind(this);
    this.onMouseMove = this.onMouseMove.bind(this);
    this.onMouseUp = this.onMouseUp.bind(this);

    this.zone.addEventListener('touchstart', this.onTouchStart, { passive: false });
    window.addEventListener('touchmove', this.onTouchMove, { passive: false });
    window.addEventListener('touchend', this.onTouchEnd, { passive: false });
    window.addEventListener('touchcancel', this.onTouchEnd, { passive: false });

    // Desktop mouse support for testing
    this.zone.addEventListener('mousedown', this.onMouseDown);
    window.addEventListener('mousemove', this.onMouseMove);
    window.addEventListener('mouseup', this.onMouseUp);
  }

  updateFromClientPoint(clientX, clientY) {
    // Measure live center and local +X / +Y unit vectors directly from rendered DOM positions
    const originRect = this.nippleEl.getBoundingClientRect();
    const xRect = this.probeXEl.getBoundingClientRect();
    const yRect = this.probeYEl.getBoundingClientRect();

    const centerX = originRect.left;
    const centerY = originRect.top;

    const screenDx = clientX - centerX;
    const screenDy = clientY - centerY;

    let ux = xRect.left - centerX;
    let uy = xRect.top - centerY;
    const uLen = Math.hypot(ux, uy) || 1;
    ux /= uLen;
    uy /= uLen;

    let vx = yRect.left - centerX;
    let vy = yRect.top - centerY;
    const vLen = Math.hypot(vx, vy) || 1;
    vx /= vLen;
    vy /= vLen;

    // Project screen delta onto the controller's local Right (+X) and Down (+Y) unit vectors
    let localDx = screenDx * ux + screenDy * uy;
    let localDy = screenDx * vx + screenDy * vy;

    const dist = Math.hypot(localDx, localDy);
    const maxDist = this.maxRadius;

    if (dist > maxDist && dist > 0) {
      const ratio = maxDist / dist;
      localDx *= ratio;
      localDy *= ratio;
    }

    // Move knob visually in local coordinates
    this.frontEl.style.transition = 'none';
    this.frontEl.style.transform = `translate(${localDx.toFixed(1)}px, ${localDy.toFixed(1)}px)`;

    // Compute normalized [-1, 1] vector with radial deadzone
    const rawMag = Math.min(dist / maxDist, 1.0);
    const deadzone = (userSettings.deadzone || 0) / 100;

    let normX = 0;
    let normY = 0;

    if (rawMag > deadzone && dist > 0) {
      const rescaledMag = (rawMag - deadzone) / (1 - deadzone);
      const unitX = localDx / Math.min(dist, maxDist);
      const unitY = localDy / Math.min(dist, maxDist);
      normX = Math.max(-1, Math.min(1, unitX * rescaledMag));
      normY = Math.max(-1, Math.min(1, unitY * rescaledMag));
    }

    if (userSettings.invertY) {
      normY = -normY;
    }

    this.emitIfNeeded(normX, normY, false);
  }

  emitIfNeeded(x, y, force) {
    const now = performance.now();
    const dx = Math.abs(x - this.lastX);
    const dy = Math.abs(y - this.lastY);

    if (!force && dx < 0.008 && dy < 0.008) {
      return;
    }
    if (!force && now - this.lastEmitTime < 14 && (x !== 0 || y !== 0)) {
      return;
    }

    this.lastX = x;
    this.lastY = y;
    this.lastEmitTime = now;
    this.onChange({ x, y });
  }

  onTouchStart(e) {
    if (e.cancelable) e.preventDefault();
    if (this.touchId !== null) return;

    const touch = e.changedTouches[0];
    if (!touch) return;

    this.touchId = touch.identifier;
    activeJoystickTouchIds.add(this.touchId);
    this.updateFromClientPoint(touch.clientX, touch.clientY);
  }

  onTouchMove(e) {
    if (this.touchId === null) return;
    for (let i = 0; i < e.changedTouches.length; i++) {
      const touch = e.changedTouches[i];
      if (touch.identifier === this.touchId) {
        if (e.cancelable) e.preventDefault();
        this.updateFromClientPoint(touch.clientX, touch.clientY);
        break;
      }
    }
  }

  onTouchEnd(e) {
    if (this.touchId === null) return;
    for (let i = 0; i < e.changedTouches.length; i++) {
      const touch = e.changedTouches[i];
      if (touch.identifier === this.touchId) {
        this.reset();
        break;
      }
    }
  }

  onMouseDown(e) {
    e.preventDefault();
    this.mouseActive = true;
    this.updateFromClientPoint(e.clientX, e.clientY);
  }

  onMouseMove(e) {
    if (!this.mouseActive) return;
    this.updateFromClientPoint(e.clientX, e.clientY);
  }

  onMouseUp() {
    if (!this.mouseActive) return;
    this.reset();
  }

  reset() {
    if (this.touchId !== null) {
      activeJoystickTouchIds.delete(this.touchId);
      this.touchId = null;
    }
    this.mouseActive = false;
    this.frontEl.style.transition = 'transform 0.12s ease-out';
    this.frontEl.style.transform = 'translate(0px, 0px)';
    this.emitIfNeeded(0, 0, true);
  }

  destroy() {
    this.reset();
    this.zone.removeEventListener('touchstart', this.onTouchStart);
    window.removeEventListener('touchmove', this.onTouchMove);
    window.removeEventListener('touchend', this.onTouchEnd);
    window.removeEventListener('touchcancel', this.onTouchEnd);
    this.zone.removeEventListener('mousedown', this.onMouseDown);
    window.removeEventListener('mousemove', this.onMouseMove);
    window.removeEventListener('mouseup', this.onMouseUp);
    this.zone.innerHTML = '';
  }
}

function initJoysticks() {
  activeJoysticks.forEach(j => j.destroy());
  activeJoysticks = [];
  activeJoystickTouchIds.clear();

  const leftZone = document.getElementById('stick-left-zone');
  if (leftZone) {
    activeJoysticks.push(new VirtualJoystick(leftZone, ({ x, y }) => {
      socket.emit("axis", { axis: 'lx', value: x });
      socket.emit("axis", { axis: 'ly', value: y });
    }));
  }

  const rightZone = document.getElementById('stick-right-zone');
  if (rightZone) {
    activeJoysticks.push(new VirtualJoystick(rightZone, ({ x, y }) => {
      socket.emit("axis", { axis: 'rx', value: x });
      socket.emit("axis", { axis: 'ry', value: y });
    }));
  }
}

// ==========================================
// 4. INITIALIZATION & JOIN LOGIC
// ==========================================
function toggleFullscreen() {
  const doc = document;
  const elem = doc.documentElement;
  if (!doc.fullscreenElement && !doc.webkitFullscreenElement) {
    if (elem.requestFullscreen) elem.requestFullscreen().catch(() => {});
    else if (elem.webkitRequestFullscreen) elem.webkitRequestFullscreen();
  } else {
    if (doc.exitFullscreen) doc.exitFullscreen().catch(() => {});
    else if (doc.webkitExitFullscreen) doc.webkitExitFullscreen();
  }
}

function initializeGamepad() {
  gamepadContainer.style.display = 'flex';
  if (hudBar) hudBar.style.display = 'flex';
  btnSettings.style.display = 'inline-flex';
  statusBadge.style.display = 'flex';
  joinScreen.style.display = 'none';
  showJoinFeedback('');

  if (!isGamepadInitialized) {
    isGamepadInitialized = true;

    document.addEventListener("touchstart", scanGamePad, { passive: false });
    document.addEventListener("touchmove", scanGamePad, { passive: false });
    document.addEventListener("touchend", scanGamePad, { passive: false });
    document.addEventListener("touchcancel", scanGamePad, { passive: false });
    document.addEventListener("mousedown", scanGamePad);
    document.addEventListener("mousemove", scanGamePad);
    document.addEventListener("mouseup", scanGamePad);
    document.addEventListener("contextmenu", e => e.preventDefault());
    window.addEventListener("blur", releaseAllInputs);

    btnSettings.addEventListener('click', (e) => {
      e.stopPropagation();
      releaseAllInputs();
      settingsModal.style.display = 'block';
    });

    btnCloseSettings.addEventListener('click', () => {
      settingsModal.style.display = 'none';
    });

    if (togglePortraitFlip) {
      togglePortraitFlip.addEventListener('change', (e) => {
        userSettings.portraitFlip = e.target.checked;
        localStorage.setItem('airinput_portrait_flip', String(userSettings.portraitFlip));
        applyUserSettingsUI();
      });
    }

    if (toggleInvertY) {
      toggleInvertY.addEventListener('change', (e) => {
        userSettings.invertY = e.target.checked;
        localStorage.setItem('airinput_invert_y', String(userSettings.invertY));
      });
    }

    if (toggleHaptics) {
      toggleHaptics.addEventListener('change', (e) => {
        userSettings.haptics = e.target.checked;
        localStorage.setItem('airinput_haptics', String(userSettings.haptics));
      });
    }

    if (rangeDeadzone) {
      rangeDeadzone.addEventListener('input', (e) => {
        userSettings.deadzone = parseInt(e.target.value, 10) || 0;
        localStorage.setItem('airinput_deadzone', String(userSettings.deadzone));
        if (deadzoneVal) deadzoneVal.textContent = `${userSettings.deadzone}%`;
      });
    }

    if (btnToggleFullscreen) {
      btnToggleFullscreen.addEventListener('click', () => {
        toggleFullscreen();
      });
    }

    // Request fullscreen on first touch interaction
    const requestFullscreenOnce = () => {
      const elem = document.documentElement;
      if (elem.requestFullscreen) elem.requestFullscreen().catch(() => {});
      else if (elem.webkitRequestFullscreen) elem.webkitRequestFullscreen();
      document.body.removeEventListener('touchstart', requestFullscreenOnce);
    };
    document.body.addEventListener('touchstart', requestFullscreenOnce, { once: true });

    const savedSkin = localStorage.getItem('gamepad_skin') || 'snes';
    loadSkin(savedSkin);
  }
}

function startPingLoop() {
  if (pingInterval) clearInterval(pingInterval);
  pingInterval = setInterval(() => {
    if (!socket.connected) return;
    const start = performance.now();
    socket.emit('ping_check', () => {
      const rtt = Math.round(performance.now() - start);
      if (statusBadge) {
        statusBadge.classList.remove('disconnected');
      }
      if (statusText) {
        statusText.textContent = `${rtt}ms`;
      }
    });
  }, 2000);
}

document.addEventListener("DOMContentLoaded", () => {
  applyUserSettingsUI();

  // Restore username if exists
  const savedUsername = localStorage.getItem('airinput_username');
  if (savedUsername) {
    usernameInput.value = savedUsername;
  }

  socket.connect();

  const submitJoin = () => {
    const rawName = usernameInput.value.trim();
    const sanitized = rawName.replace(/:/g, '').trim();
    if (!sanitized) {
      showJoinFeedback('Escribe tu identificador de jugador.');
      usernameInput.focus();
      return;
    }
    if (sanitized.toLowerCase() === 'system') {
      showJoinFeedback('Identificador reservado por la consola. Elige otro.');
      usernameInput.focus();
      return;
    }
    showJoinFeedback('');
    currentUsername = sanitized;
    localStorage.setItem('airinput_username', currentUsername);
    usernameInput.blur();
    if (userSettings.haptics && navigator.vibrate) {
      navigator.vibrate(30);
    }
    socket.emit('register_player', { username: currentUsername });
  };

  joinBtn.addEventListener('click', submitJoin);
  usernameInput.addEventListener('input', () => showJoinFeedback(''));
  usernameInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') submitJoin();
  });

  socket.on('connect', () => {
    if (joinLed) joinLed.classList.remove('offline');
    if (statusBadge) statusBadge.classList.remove('disconnected');
    if (statusText) statusText.textContent = 'En línea';
    // Auto re-register if we were already playing and reconnected
    if (isGamepadInitialized && currentUsername) {
      socket.emit('register_player', { username: currentUsername });
    }
  });

  socket.on('registration_success', () => {
    initializeGamepad();
    startPingLoop();
  });

  socket.on('registration_failed', (message) => {
    showJoinFeedback(message || 'No se pudo conectar el mando.');
    joinScreen.style.display = 'flex';
    gamepadContainer.style.display = 'none';
    if (hudBar) hudBar.style.display = 'none';
  });

  socket.on('disconnect', () => {
    releaseAllInputs();
    if (joinLed) joinLed.classList.add('offline');
    if (statusBadge) statusBadge.classList.add('disconnected');
    if (statusText) statusText.textContent = 'Sin señal';
  });
});


