(function () {
  function normalize(value) {
    return value.trim().toLowerCase();
  }

  function initBootAnimation() {
    var terminal = document.querySelector(".not-found-terminal");
    if (!terminal) {
      return;
    }

    window.setTimeout(function () {
      terminal.classList.remove("is-booting");
    }, 700);
  }

  function initCommandPalette() {
    var form = document.querySelector("[data-not-found-command]");
    if (!form) {
      return;
    }

    var input = form.querySelector("input[name='command']");
    var output = form.querySelector("[data-not-found-output]");

    function setOutput(message) {
      output.textContent = message;
    }

    function runCommand(raw) {
      var command = normalize(raw);
      if (!command) {
        setOutput("usage: help | home | projects");
        return;
      }

      if (command === "help") {
        setOutput("commands: help, home, projects");
        return;
      }

      if (command === "home") {
        window.location.href = "/";
        return;
      }

      if (command === "projects") {
        window.location.href = "/section/projects";
        return;
      }

      setOutput("unknown opcode: " + command);
    }

    form.addEventListener("submit", function (event) {
      event.preventDefault();
      runCommand(input.value);
      input.value = "";
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", function () {
      initBootAnimation();
      initCommandPalette();
    });
  } else {
    initBootAnimation();
    initCommandPalette();
  }
})();
