(() => {
  "use strict";

  const page = document.querySelector(".monitor-main");
  const empty = document.querySelector("#monitor-empty");
  const latest = document.querySelector("#monitor-latest");
  const latestQueue = document.querySelector("#monitor-latest-queue");
  const latestNumber = document.querySelector("#monitor-latest-number");
  const latestCounter = document.querySelector("#monitor-latest-counter");
  const historySection = document.querySelector("#monitor-history-section");
  const history = document.querySelector("#monitor-history");
  const connection = document.querySelector("#monitor-connection");
  const soundButton = document.querySelector("#monitor-sound");
  const fullscreenButton = document.querySelector("#monitor-fullscreen");

  if (
    !page || !empty || !latest || !latestQueue || !latestNumber ||
    !latestCounter || !historySection || !history || !connection
  ) {
    return;
  }

  const limit = Number.parseInt(page.dataset.limit || "10", 10);
  const highlightMs = Number.parseInt(page.dataset.highlightMs || "5000", 10);
  const soundConfigured = page.dataset.soundEnabled === "true";
  let calls = [];
  let highlightTimer;
  let audioContext;
  let soundArmed = false;

  const message = (name) => page.dataset[name] || "";
  const localTime = (value) => {
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return "";
    }
    return new Intl.DateTimeFormat(document.documentElement.lang, {
      hour: "2-digit",
      minute: "2-digit",
    }).format(date);
  };
  const historyItem = (call) => {
    const item = document.createElement("li");
    item.dataset.callId = String(call.id);

    const number = document.createElement("strong");
    number.className = "history-number";
    number.textContent = call.number;
    const counter = document.createElement("span");
    counter.className = "history-counter";
    counter.textContent = call.counter;
    const queue = document.createElement("span");
    queue.className = "history-queue";
    queue.textContent = call.queue;
    const time = document.createElement("time");
    time.dateTime = call.called_at;
    time.textContent = localTime(call.called_at);
    item.append(number, counter, queue, time);
    return item;
  };
  const render = (emphasize) => {
    const current = calls[0];
    empty.hidden = Boolean(current);
    latest.hidden = !current;
    historySection.hidden = !current;
    if (!current) {
      return;
    }

    latestQueue.textContent = current.queue;
    latestNumber.textContent = current.number;
    latestCounter.textContent = current.counter;
    history.replaceChildren(...calls.map(historyItem));

    if (emphasize) {
      window.clearTimeout(highlightTimer);
      latest.classList.remove("is-new");
      void latest.offsetWidth;
      latest.style.setProperty("--highlight-duration", `${highlightMs}ms`);
      latest.classList.add("is-new");
      highlightTimer = window.setTimeout(() => latest.classList.remove("is-new"), highlightMs);
    }
  };
  const mergeCall = (call) => {
    calls = [call, ...calls.filter((existing) => existing.id !== call.id)].slice(0, limit);
  };
  const beep = () => {
    if (!soundConfigured || !soundArmed || !audioContext) {
      return;
    }
    const oscillator = audioContext.createOscillator();
    const gain = audioContext.createGain();
    oscillator.type = "sine";
    oscillator.frequency.value = 880;
    gain.gain.setValueAtTime(0.0001, audioContext.currentTime);
    gain.gain.exponentialRampToValueAtTime(0.18, audioContext.currentTime + 0.02);
    gain.gain.exponentialRampToValueAtTime(0.0001, audioContext.currentTime + 0.35);
    oscillator.connect(gain);
    gain.connect(audioContext.destination);
    oscillator.start();
    oscillator.stop(audioContext.currentTime + 0.36);
  };
  const armSound = async () => {
    if (!soundConfigured) {
      return false;
    }
    try {
      const AudioContext = window.AudioContext || window.webkitAudioContext;
      if (!AudioContext) {
        if (soundButton) {
          soundButton.hidden = true;
        }
        return false;
      }
      audioContext ||= new AudioContext();
      await audioContext.resume();
      soundArmed = audioContext.state === "running";
      if (soundArmed && soundButton) {
        soundButton.textContent = message("soundReady");
        soundButton.disabled = true;
      }
      return soundArmed;
    } catch {
      return false;
    }
  };

  if (soundButton) {
    soundButton.addEventListener("click", () => {
      void armSound();
    });
  }
  void armSound();

  if (fullscreenButton) {
    const syncFullscreenButton = () => {
      const fullscreen = Boolean(document.fullscreenElement);
      fullscreenButton.textContent = message(fullscreen ? "exitFullscreen" : "fullscreen");
      fullscreenButton.setAttribute("aria-pressed", String(fullscreen));
    };
    fullscreenButton.addEventListener("click", async () => {
      try {
        if (document.fullscreenElement) {
          await document.exitFullscreen();
        } else {
          await document.documentElement.requestFullscreen();
        }
      } catch {
        // Fullscreen can be denied by browser or kiosk policy.
      }
    });
    document.addEventListener("fullscreenchange", syncFullscreenButton);
    syncFullscreenButton();
  }

  const source = new EventSource("/api/v1/events/calls");
  source.addEventListener("open", () => {
    connection.textContent = message("connected");
    connection.classList.add("connected");
  });
  source.addEventListener("error", () => {
    connection.textContent = message("reconnecting");
    connection.classList.remove("connected");
  });
  source.addEventListener("snapshot", (event) => {
    try {
      calls = JSON.parse(event.data).slice(0, limit);
      render(false);
    } catch {
      // EventSource reconnects automatically if the stream becomes unusable.
    }
  });
  source.addEventListener("call", (event) => {
    try {
      const call = JSON.parse(event.data);
      if (calls[0]?.id === call.id) {
        return;
      }
      mergeCall(call);
      render(true);
      beep();
    } catch {
      // Ignore a malformed event; a reconnect snapshot restores the state.
    }
  });
})();
