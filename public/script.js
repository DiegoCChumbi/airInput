const socket = io();

let activeButtons = new Set();

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
  if (e.target.id !== 'btn-fullscreen') {
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

document.addEventListener("DOMContentLoaded", () => {
  console.log("Script cargado y listo.");

  document.addEventListener("touchstart", scanGamePad, { passive: false });
  document.addEventListener("touchmove", scanGamePad, { passive: false });
  document.addEventListener("touchend", scanGamePad, { passive: false });
  document.addEventListener("touchcancel", scanGamePad, { passive: false });

  document.addEventListener("contextmenu", e => { e.preventDefault(); return false; });

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
});
