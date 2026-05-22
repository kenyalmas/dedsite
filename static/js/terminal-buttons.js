(function () {
  const terminal = document.querySelector(".terminal");
  if (!terminal) {
    return;
  }

  const output = document.querySelector("[data-command-output]");
  let reconnectTimer = 0;

  function setStatus(message) {
    if (output) {
      output.textContent = message;
    }
  }

  function resetCommandPalette() {
    const input = document.querySelector("[data-command-palette] input[name='command']");
    const ghost = document.querySelector("[data-command-ghost]");
    const firstTab = document.querySelector("#section-tabs .tab");

    if (input) {
      input.value = "";
    }
    if (ghost) {
      ghost.textContent = "";
    }
    if (output) {
      output.textContent = "";
    }

    document.querySelectorAll("#section-panel .item").forEach(function (item) {
      item.hidden = false;
    });

    if (firstTab && !firstTab.classList.contains("is-active")) {
      firstTab.click();
    }
  }

  function reconnect() {
    window.clearTimeout(reconnectTimer);
    terminal.classList.remove("is-minimized", "is-focus-mode");
    terminal.classList.add("is-reconnecting");
    setStatus("session terminated");

    reconnectTimer = window.setTimeout(function () {
      terminal.classList.remove("is-reconnecting");
      resetCommandPalette();
    }, 750);
  }

  function toggleProfile() {
    terminal.classList.remove("is-focus-mode");
    terminal.classList.toggle("is-minimized");

    if (terminal.classList.contains("is-minimized")) {
      return;
    }

    setStatus("terminal restored");
  }

  function toggleFocus() {
    terminal.classList.remove("is-minimized");
    terminal.classList.toggle("is-focus-mode");

    if (terminal.classList.contains("is-focus-mode")) {
      setStatus("portfolio focus mode");
      return;
    }

    setStatus("standard layout restored");
  }

  document.addEventListener("click", function (event) {
    const button = event.target.closest("[data-terminal-action]");
    if (!button) {
      return;
    }

    const action = button.getAttribute("data-terminal-action");
    if (action === "close") {
      const href = button.getAttribute("data-terminal-href");
      if (href) {
        window.location.href = href;
        return;
      }
      reconnect();
    }
    if (action === "minimize") {
      toggleProfile();
    }
    if (action === "maximize") {
      toggleFocus();
    }
  });

})();
