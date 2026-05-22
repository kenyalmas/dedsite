(function () {
  function normalize(value) {
    return value.trim().toLowerCase();
  }

  function sectionLinks() {
    return Array.from(document.querySelectorAll("#section-tabs .tab"));
  }

  var pendingFilter = null;
  var activeFilter = null;

  function items() {
    return Array.from(document.querySelectorAll("#section-panel .item"));
  }

  function indexEntries() {
    return Array.from(document.querySelectorAll("[data-command-entry]"));
  }

  function availableTags() {
    var tags = [];
    var seen = {};

    document.querySelectorAll("[data-tag]").forEach(function (tag) {
      var value = tag.textContent.trim();
      var key = normalize(value);
      if (!seen[key]) {
        seen[key] = true;
        tags.push(value);
      }
    });

    return tags.sort();
  }

  function allTags() {
    var tags = [];
    var seen = {};

    indexEntries().forEach(function (entry) {
      entry.dataset.tags.split("|").forEach(function (tag) {
        var value = tag.trim();
        var key = normalize(value);
        if (value && !seen[key]) {
          seen[key] = true;
          tags.push(value);
        }
      });
    });

    return tags.sort();
  }

  function suggestionList() {
    var commands = [
      "help",
      "work",
      "experience",
      "projects",
      "security",
      "ai",
      "contact",
      "github",
      "linkedin",
      "email",
      "clear",
      "tags",
      "find hardware",
      "find networking",
      "find security",
      "find ai"
    ];

    return commands.concat(allTags().map(function (tag) {
      return normalize(tag);
    }));
  }

  function inlineSuggestion(value) {
    var query = normalize(value);
    if (!query) {
      return "";
    }

    return suggestionList().find(function (suggestion) {
      return suggestion !== query && suggestion.indexOf(query) === 0;
    }) || "";
  }

  function clearFilter() {
    items().forEach(function (item) {
      item.hidden = false;
    });
    activeFilter = null;
    setActiveTags("");
  }

  function filterItems(keyword) {
    var needle = normalize(keyword);
    var count = 0;

    items().forEach(function (item) {
      var haystack = normalize(item.dataset.search || "");
      var matched = haystack.indexOf(needle) !== -1;
      item.hidden = !matched;
      if (matched) {
        count += 1;
      }
    });

    return count;
  }

  function setActiveTags(tag) {
    var needle = normalize(tag || "");
    document.querySelectorAll("[data-tag]").forEach(function (tagButton) {
      var matched = needle && normalize(tagButton.textContent) === needle;
      tagButton.classList.toggle("is-active", matched);
      if (matched) {
        tagButton.setAttribute("aria-pressed", "true");
      } else {
        tagButton.setAttribute("aria-pressed", "false");
      }
    });
  }

  function filterTag(tag) {
    var needle = normalize(tag);
    var count = 0;

    items().forEach(function (item) {
      var tags = normalize(item.dataset.tags || "");
      var matched = tags.split(/\s+/).indexOf(needle) !== -1 || tags.indexOf(needle) !== -1;
      item.hidden = !matched;
      if (matched) {
        count += 1;
      }
    });

    activeFilter = { type: "tag", value: tag };
    setActiveTags(tag);
    return count;
  }

  function showTagFilter(tag, output) {
    var count = filterTag(tag);
    if (output) {
      output.textContent = count + " item(s) tagged " + tag + " - type clear to reset";
    }
  }

  function activateSection(slug) {
    var link = sectionLinks().find(function (tab) {
      return tab.getAttribute("href") === "/section/" + slug;
    });

    if (!link) {
      return false;
    }

    link.click();
    return true;
  }

  function activeSection() {
    var active = document.querySelector("#section-tabs .tab.is-active");
    if (!active) {
      return "";
    }
    return active.getAttribute("href").replace("/section/", "");
  }

  function firstSectionForKeyword(keyword, tagOnly) {
    var needle = normalize(keyword);
    var match = indexEntries().find(function (entry) {
      var haystack = normalize(tagOnly ? entry.dataset.tags : entry.dataset.search);
      return haystack.indexOf(needle) !== -1;
    });

    return match ? match.dataset.section : "";
  }

  function applyPendingFilter() {
    if (!pendingFilter) {
      return;
    }

    if (pendingFilter.type === "tag") {
      filterTag(pendingFilter.value);
    } else {
      filterItems(pendingFilter.value);
    }
    pendingFilter = null;
  }

  function commandMap() {
    var sections = {};

    sectionLinks().forEach(function (tab) {
      var slug = tab.getAttribute("href").replace("/section/", "");
      sections[slug] = function () {
        clearFilter();
        pendingFilter = null;
        activateSection(slug);
        return "opened " + slug;
      };
      sections[tab.textContent.trim().toLowerCase()] = sections[slug];
    });

    return Object.assign(sections, {
      help: function () {
        return "commands: work, projects, security, ai, tags, find <keyword>, github, linkedin, email";
      },
      github: function () {
        window.open("https://github.com/kenyalmas", "_blank", "noopener,noreferrer");
        return "opening github";
      },
      linkedin: function () {
        window.open("https://www.linkedin.com/in/kenneth-almas-09a448329/", "_blank", "noopener,noreferrer");
        return "opening linkedin";
      },
      email: function () {
        window.location.href = "mailto:kennethalmas232@gmail.com";
        return "opening email";
      },
      contact: function () {
        return "contacts: email, github, linkedin";
      },
      tags: function () {
        var tags = allTags();
        if (tags.length === 0) {
          return "no tags available";
        }
        return "tags: " + tags.join(", ");
      },
      clear: function () {
        clearFilter();
        pendingFilter = null;
        return "";
      }
    });
  }

  function initPalette() {
    var form = document.querySelector("[data-command-palette]");
    if (!form) {
      return;
    }

    var input = form.querySelector("input[name='command']");
    var output = form.querySelector("[data-command-output]");
    var ghost = form.querySelector("[data-command-ghost]");

    function updateGhost() {
      var suggestion = inlineSuggestion(input.value);
      if (!suggestion) {
        ghost.textContent = "";
        return;
      }

      ghost.textContent = input.value + suggestion.slice(input.value.length);
    }

    function acceptGhost() {
      var suggestion = inlineSuggestion(input.value);
      if (!suggestion) {
        return false;
      }

      input.value = suggestion;
      updateGhost();
      return true;
    }

    form.addEventListener("submit", function (event) {
      event.preventDefault();

      var command = normalize(input.value);
      var commands = commandMap();

      if (!command) {
        output.textContent = "enter a command";
        return;
      }

      if (!commands[command]) {
        if (command.indexOf("find ") === 0 || command.indexOf("search ") === 0) {
          var keyword = command.replace(/^(find|search)\s+/, "");
          var targetSection = firstSectionForKeyword(keyword, false);
          if (targetSection && targetSection !== activeSection()) {
            pendingFilter = { type: "keyword", value: keyword };
            activateSection(targetSection);
          output.textContent = "opening " + targetSection + " for " + keyword;
          input.value = "";
          updateGhost();
          return;
        }

        var found = filterItems(keyword);
        output.textContent = found + " match(es) for " + keyword;
        input.value = "";
        updateGhost();
        return;
      }

        var tagMatches = allTags().filter(function (tag) {
          return normalize(tag) === command;
        });
        if (tagMatches.length > 0) {
          var section = firstSectionForKeyword(tagMatches[0], true);
          if (section && section !== activeSection()) {
            pendingFilter = { type: "tag", value: tagMatches[0] };
            activateSection(section);
            output.textContent = "opening " + section + " for " + tagMatches[0];
            input.value = "";
            updateGhost();
            return;
          }

          var count = filterTag(tagMatches[0]);
          output.textContent = count + " item(s) tagged " + tagMatches[0];
          input.value = "";
          updateGhost();
          return;
        }

        output.textContent = "unknown command: " + command;
        return;
      }

      output.textContent = commands[command]();
      input.value = "";
      updateGhost();
    });

    input.addEventListener("input", updateGhost);

    document.addEventListener("click", function (event) {
      var tag = event.target.closest("[data-tag]");
      if (!tag) {
        return;
      }

      var value = tag.textContent.trim();
      if (!value) {
        return;
      }

      showTagFilter(value, output);
    });

    input.addEventListener("keydown", function (event) {
      if (event.key !== "Tab" && event.key !== "ArrowRight") {
        return;
      }

      if (!acceptGhost()) {
        return;
      }

      event.preventDefault();
    });

    document.addEventListener("keydown", function (event) {
      if (event.key !== "/") {
        return;
      }

      if (event.target instanceof HTMLInputElement || event.target instanceof HTMLTextAreaElement) {
        return;
      }

      event.preventDefault();
      input.focus();
      updateGhost();
    });

    document.body.addEventListener("htmx:afterSwap", function (event) {
      if (event.target.id === "section-panel") {
        applyPendingFilter();
        if (activeFilter && activeFilter.type === "tag") {
          setActiveTags(activeFilter.value);
        }
      }
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initPalette);
  } else {
    initPalette();
  }
})();
