// Frontend Application Logic
(function () {
  'use strict';

  // DOM Elements
  const btnScreenOff = document.getElementById('btnScreenOff');
  const btnBlackout = document.getElementById('btnBlackout');
  const btnExit = document.getElementById('btnExit');
  const statusLabel = document.getElementById('statusLabel');
  const statusDot = document.querySelector('.status-dot');
  
  const timerPresetBtns = document.querySelectorAll('.timer-preset-btn');
  const timerStatusBox = document.getElementById('timerStatusBox');
  const timerCountdown = document.getElementById('timerCountdown');
  const timerProgressBar = document.getElementById('timerProgressBar');
  const btnCancelTimer = document.getElementById('btnCancelTimer');

  // State
  let timerInterval = null;
  let timerRemainingSec = 0;
  let timerTotalSec = 0;

  // Helpers
  function setStatus(text, isActive = false) {
    if (statusLabel) statusLabel.textContent = text;
    if (statusDot) {
      if (isActive) {
        statusDot.classList.add('active');
      } else {
        statusDot.classList.remove('active');
      }
    }
  }

  function formatTime(seconds) {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${String(mins).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
  }

  // Action Handlers
  async function triggerScreenOff() {
    setStatus('SCREEN OFF...', true);
    if (window.goTurnOffScreen) {
      try {
        await window.goTurnOffScreen(1000); // 1s delay
      } catch (err) {
        console.error('Error turning off screen:', err);
      }
    }
    setTimeout(() => setStatus('READY', false), 2000);
  }

  async function triggerBlackout() {
    setStatus('BLACKOUT ACTIVE', true);
    if (window.goShowBlinder) {
      try {
        await window.goShowBlinder();
      } catch (err) {
        console.error('Error showing blinder:', err);
      }
    }
    setTimeout(() => setStatus('READY', false), 500);
  }

  async function triggerExit() {
    if (window.goExitApp) {
      await window.goExitApp();
    } else {
      window.close();
    }
  }

  // Timer Management
  function startTimer(minutes) {
    cancelTimer();

    timerTotalSec = minutes * 60;
    timerRemainingSec = timerTotalSec;

    timerStatusBox.classList.remove('hidden');
    timerCountdown.textContent = formatTime(timerRemainingSec);
    timerProgressBar.style.width = '100%';
    setStatus(`TIMER (${minutes}m)`, true);

    timerInterval = setInterval(() => {
      timerRemainingSec--;

      if (timerRemainingSec <= 0) {
        cancelTimer();
        triggerScreenOff();
        return;
      }

      timerCountdown.textContent = formatTime(timerRemainingSec);
      const percentage = (timerRemainingSec / timerTotalSec) * 100;
      timerProgressBar.style.width = `${percentage}%`;
    }, 1000);
  }

  function cancelTimer() {
    if (timerInterval) {
      clearInterval(timerInterval);
      timerInterval = null;
    }
    timerStatusBox.classList.add('hidden');
    timerPresetBtns.forEach(btn => btn.classList.remove('active'));
    setStatus('READY', false);
  }

  // Event Listeners
  if (btnScreenOff) {
    btnScreenOff.addEventListener('click', triggerScreenOff);
  }

  if (btnBlackout) {
    btnBlackout.addEventListener('click', triggerBlackout);
  }

  if (btnExit) {
    btnExit.addEventListener('click', triggerExit);
  }

  timerPresetBtns.forEach(btn => {
    btn.addEventListener('click', () => {
      const minutes = parseInt(btn.getAttribute('data-minutes'), 10);
      timerPresetBtns.forEach(b => b.classList.remove('active'));
      btn.classList.add('active');
      startTimer(minutes);
    });
  });

  if (btnCancelTimer) {
    btnCancelTimer.addEventListener('click', cancelTimer);
  }

  // Global Key Shortcuts within GUI window
  window.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      cancelTimer();
    }
  });

  // Expose callback for Go side notifications if needed
  window.onBlinderDismissed = function() {
    setStatus('READY', false);
  };

  console.log('T2 Display Blinder frontend loaded.');
})();
