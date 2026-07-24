(() => {
  "use strict";

  const storageKey = "turnino.counterToken";
  const page = document.querySelector(".caller-main");
  const tokenForm = document.querySelector("#caller-token-form");
  const tokenInput = document.querySelector("#caller-token");
  const identity = document.querySelector("#caller-identity");
  const tokenSettings = document.querySelector("#caller-token-settings");
  const status = document.querySelector("#caller-status");
  const callButtons = [...document.querySelectorAll(".caller-next")];
  const recallButtons = [...document.querySelectorAll(".caller-recall")];
  const customForms = [...document.querySelectorAll(".caller-custom-form")];

  if (!page || !tokenForm || !tokenInput || !identity || !tokenSettings || !status) {
    return;
  }

  const message = (name) => page.dataset[name] || "";
  const setStatus = (text, kind = "") => {
    status.textContent = text;
    status.className = `caller-status${kind ? ` ${kind}` : ""}`;
  };
  const storedToken = () => {
    try {
      return localStorage.getItem(storageKey) || "";
    } catch {
      return "";
    }
  };
  const saveToken = (token) => {
    try {
      localStorage.setItem(storageKey, token);
      return true;
    } catch {
      return false;
    }
  };
  const setCalling = (calling) => {
    [...callButtons, ...recallButtons, ...customForms.map((form) => form.querySelector("button"))].forEach((button) => {
      button.disabled = calling;
    });
  };
  const clearIdentity = () => {
    identity.textContent = "";
    identity.hidden = true;
  };
  const loadIdentity = async (token) => {
    clearIdentity();
    if (!token) {
      return false;
    }
    try {
      const response = await fetch("/api/v1/counters/me", {
        headers: {
          Accept: "application/json",
          Authorization: `Bearer ${token}`,
        },
      });
      if (!response.ok) {
        tokenSettings.open = true;
        if (response.status === 401) {
          setStatus(message("invalidToken"), "error");
        } else {
          setStatus(message("callFailed"), "error");
        }
        return false;
      }
      const counter = await response.json();
      identity.textContent = `${message("counterPrefix")}: ${counter.name}`;
      identity.hidden = false;
      tokenSettings.open = false;
      return true;
    } catch {
      tokenSettings.open = true;
      setStatus(message("callFailed"), "error");
      return false;
    }
  };

  const initialToken = storedToken();
  tokenInput.value = initialToken;
  void loadIdentity(initialToken);

  tokenForm.addEventListener("submit", (event) => {
    event.preventDefault();
    const token = tokenInput.value.trim();
    if (!token) {
      setStatus(message("missingToken"), "error");
      tokenInput.focus();
      return;
    }
    if (!saveToken(token)) {
      setStatus(message("callFailed"), "error");
      return;
    }
    void loadIdentity(token).then((valid) => {
      if (valid) {
        setStatus(message("tokenSaved"), "success");
      }
    });
  });

  const performCall = async (queueElement, endpoint, options = {}) => {
    const token = storedToken();
    if (!token) {
      setStatus(message("missingToken"), "error");
      tokenSettings.open = true;
      tokenInput.focus();
      return null;
    }

    setCalling(true);
    setStatus(message("calling"));
    try {
      const response = await fetch(endpoint, {
        method: "POST",
        headers: {
          Accept: "application/json",
          Authorization: `Bearer ${token}`,
          ...(options.body ? {"Content-Type": "application/json"} : {}),
        },
        ...(options.body ? {body: JSON.stringify(options.body)} : {}),
      });
      const result = await response.json().catch(() => ({}));
      if (!response.ok) {
        if (response.status === 401) {
          clearIdentity();
          tokenSettings.open = true;
        }
        setStatus(
          response.status === 401
            ? message("invalidToken")
            : options.recall && response.status === 404
              ? message("noPreviousCall")
              : options.custom && response.status === 409
                ? message("customDisabled")
                : options.custom && response.status === 400
                  ? message("customInvalid")
              : result.error || message("callFailed"),
          "error",
        );
        return null;
      }
      const currentNumber = queueElement.querySelector("[data-current-number]");
      const queueName = queueElement.querySelector("h3")?.textContent.trim() || "";
      if (currentNumber && !options.recall) {
        currentNumber.textContent = result.number;
      }
      const prefix = options.recall ? message("recalledPrefix") : message("calledPrefix");
      setStatus(`${prefix} ${result.number} ${message("calledOn")} ${queueName}`, "success");
      return result;
    } catch {
      setStatus(message("callFailed"), "error");
      return null;
    } finally {
      setCalling(false);
    }
  };

  callButtons.forEach((button) => {
    button.addEventListener("click", () => {
      const queueElement = button.closest(".caller-queue");
      if (queueElement) {
        void performCall(queueElement, `/api/v1/queues/${button.dataset.queueId}/calls/next`);
      }
    });
  });

  recallButtons.forEach((button) => {
    button.addEventListener("click", () => {
      const queueElement = button.closest(".caller-queue");
      if (queueElement) {
        void performCall(queueElement, `/api/v1/queues/${button.dataset.queueId}/calls/recall`, {recall: true});
      }
    });
  });

  customForms.forEach((form) => {
    form.addEventListener("submit", (event) => {
      event.preventDefault();
      const queueElement = form.closest(".caller-queue");
      const input = form.elements.sequence_value;
      if (!queueElement || !input || !input.reportValidity()) {
        return;
      }
      void performCall(
        queueElement,
        `/api/v1/queues/${form.dataset.queueId}/calls/custom`,
        {body: {sequence_value: Number.parseInt(input.value, 10)}, custom: true},
      );
    });
  });
})();
