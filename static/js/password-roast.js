(function () {
  var input = document.getElementById("password-input");
  var clear = document.getElementById("clear-password");
  var generate = document.getElementById("generate-password");
  var bar = document.getElementById("strength-bar");
  var scoreLine = document.getElementById("score-line");
  var copy = document.getElementById("roast-copy");
  var feedbackList = document.getElementById("feedback-list");

  var common = ["password", "qwerty", "letmein", "welcome", "admin", "iloveyou", "dragon", "football", "monkey", "abc123"];
  var lowerChars = "abcdefghijkmnopqrstuvwxyz";
  var upperChars = "ABCDEFGHJKLMNPQRSTUVWXYZ";
  var numberChars = "23456789";
  var symbolChars = "!#$%&*?@";
  var allChars = lowerChars + upperChars + numberChars + symbolChars;

  function hasAny(value, list) {
    var lower = value.toLowerCase();
    return list.some(function (word) { return lower.indexOf(word) !== -1; });
  }

  function evaluate(value) {
    var checks = [
      { ok: value.length >= 16, text: "Use at least 16 characters." },
      { ok: /[a-z]/.test(value) && /[A-Z]/.test(value), text: "Mix upper and lower case." },
      { ok: /\d/.test(value), text: "Add a number that is not just a birthday cameo." },
      { ok: /[^A-Za-z0-9]/.test(value), text: "Add a symbol for extra crunch." },
      { ok: !hasAny(value, common), text: "Avoid common password words." },
      { ok: !/(.)\1\1/.test(value) && !/1234|abcd|qwer|asdf/i.test(value), text: "Avoid repeats and keyboard walks." }
    ];
    var passed = checks.filter(function (check) { return check.ok; }).length;
    var score = Math.round((passed / checks.length) * 100);
    return { checks: checks, score: score };
  }

  function randomIndex(max) {
    if (window.crypto && window.crypto.getRandomValues) {
      var values = new Uint32Array(1);
      var limit = Math.floor(0x100000000 / max) * max;
      do {
        window.crypto.getRandomValues(values);
      } while (values[0] >= limit);
      return values[0] % max;
    }
    return Math.floor(Math.random() * max);
  }

  function randomChar(chars) {
    return chars.charAt(randomIndex(chars.length));
  }

  function shuffle(chars) {
    for (var i = chars.length - 1; i > 0; i--) {
      var j = randomIndex(i + 1);
      var swap = chars[i];
      chars[i] = chars[j];
      chars[j] = swap;
    }
    return chars;
  }

  function generatePassword() {
    var chars = [
      randomChar(lowerChars),
      randomChar(lowerChars),
      randomChar(upperChars),
      randomChar(upperChars),
      randomChar(numberChars),
      randomChar(numberChars),
      randomChar(symbolChars),
      randomChar(symbolChars)
    ];
    while (chars.length < 18) {
      chars.push(randomChar(allChars));
    }
    return shuffle(chars).join("");
  }

  function roastFor(score, value) {
    if (!value) return ["Waiting for a password to judge", "The studio is sharpening its tiny clipboard."];
    if (score < 35) return ["Security found this in a bargain bin", "This password has the defensive posture of a screen door in a thunderstorm."];
    if (score < 65) return ["Better, but still wearing flip-flops to a firewall", "It has ambition. It also has a few habits attackers have seen since dial-up."];
    if (score < 90) return ["Respectable chaos", "This is getting harder to bully. Add length or randomness and it starts looking like adult supervision."];
    return ["Actually solid", "The roast has been canceled due to operational competence."];
  }

  function render() {
    var value = input.value;
    var result = evaluate(value);
    var roast = roastFor(result.score, value);
    bar.style.width = (value ? result.score : 0) + "%";
    bar.dataset.score = String(result.score);
    scoreLine.textContent = value ? "strength estimate: " + result.score + "/100" : "awaiting questionable decisions...";
    copy.textContent = value ? roast[1] : "The studio keeps the judgment local.";
    feedbackList.innerHTML = "";
    feedbackList.hidden = !value;
    if (!value) return;
    result.checks.forEach(function (check) {
      var item = document.createElement("li");
      item.className = check.ok ? "is-good" : "is-missing";
      item.textContent = (check.ok ? "pass: " : "fix: ") + check.text;
      feedbackList.appendChild(item);
    });
  }

  input.addEventListener("input", render);
  clear.addEventListener("click", function () {
    input.value = "";
    input.focus();
    render();
  });
  generate.addEventListener("click", function () {
    input.value = generatePassword();
    input.focus();
    render();
  });
  render();
}());
