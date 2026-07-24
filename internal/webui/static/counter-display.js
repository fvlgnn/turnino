(() => {
  "use strict";

  const callerTokenKey = "turnino.counterToken";
  const displayCounterKey = "turnino.displayCounter";
  const page = document.querySelector(".counter-display-main");
  const name = document.querySelector("#counter-display-name");
  const connection = document.querySelector("#counter-display-connection");
  const empty = document.querySelector("#counter-display-empty");
  const emptyText = empty?.querySelector("p");
  const call = document.querySelector("#counter-display-call");
  const queue = document.querySelector("#counter-display-queue");
  const number = document.querySelector("#counter-display-number");
  const status = document.querySelector("#counter-display-status");

  if (!page || !name || !connection || !empty || !emptyText || !call || !queue || !number || !status) {
    return;
  }

  const message = (key) => page.dataset[key] || "";
  const setStatus = (text, kind = "") => {
    status.textContent = text;
    status.className = `caller-status${kind ? ` ${kind}` : ""}`;
  };
  const readDisplayCounter = () => {
    try {
      return JSON.parse(localStorage.getItem(displayCounterKey) || "null");
    } catch {
      return null;
    }
  };
  const saveDisplayCounter = (counter) => {
    try {
      localStorage.setItem(displayCounterKey, JSON.stringify(counter));
      return true;
    } catch {
      return false;
    }
  };
  const configuredFromURL = () => {
    const id = Number.parseInt(page.dataset.displayCounterId || "0", 10);
    const counterName = page.dataset.displayCounterName || "";
    if (id < 1 || !counterName) {
      return null;
    }
    const counter = { id, name: counterName };
    if (!saveDisplayCounter(counter)) {
      setStatus(message("loadFailed"), "error");
    }
    window.history.replaceState({}, "", window.location.pathname);
    return counter;
  };
  const callerToken = () => {
    try {
      return localStorage.getItem(callerTokenKey) || "";
    } catch {
      return "";
    }
  };
  const counterFromCaller = async () => {
    const token = callerToken();
    if (!token) {
      return null;
    }
    try {
      const response = await fetch("/api/v1/counters/me", {
        headers: {
          Accept: "application/json",
          Authorization: `Bearer ${token}`,
        },
      });
      if (!response.ok) {
        return null;
      }
      const counter = await response.json();
      const selected = { id: counter.id, name: counter.name };
      saveDisplayCounter(selected);
      return selected;
    } catch {
      setStatus(message("loadFailed"), "error");
      return null;
    }
  };
  const showCall = (record) => {
    queue.textContent = record.queue;
    number.textContent = record.number;
    empty.hidden = true;
    call.hidden = false;
  };
  const connect = (counter) => {
    name.textContent = counter.name;
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
        const latest = JSON.parse(event.data).find((record) => record.counter_id === counter.id);
        if (latest) {
          showCall(latest);
        }
      } catch {
        // EventSource reconnects automatically and the next snapshot restores state.
      }
    });
    source.addEventListener("call", (event) => {
      try {
        const record = JSON.parse(event.data);
        if (record.counter_id === counter.id) {
          showCall(record);
        }
      } catch {
        // Ignore a malformed event; a reconnect snapshot restores the state.
      }
    });
  };

  void (async () => {
    const counter = configuredFromURL() || await counterFromCaller() || readDisplayCounter();
    if (!counter?.id || !counter?.name) {
      name.textContent = document.title.split(" · ")[0];
      emptyText.textContent = message("notConfigured");
      connection.hidden = true;
      setStatus("");
      return;
    }
    connect(counter);
  })();
})();
