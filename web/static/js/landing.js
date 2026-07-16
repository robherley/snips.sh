const initLanding = () => {
  const screen = document.querySelector("#terminal-screen");
  const heroCmd = document.querySelector("#hero-command");
  if (!screen || !heroCmd) return;

  const esc = (text) =>
    text
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;");

  const host = esc(heroCmd.dataset.sshHost || "snips.sh");
  const port = heroCmd.dataset.sshPort || "";
  const httpAddr = esc(heroCmd.dataset.httpAddr || "https://snips.sh");
  const portFlag = port && port !== "22" ? `-p ${esc(port)} ` : "";
  const sshTo = (dest) => `ssh ${portFlag}${dest}`;

  const bar = '<span class="t-bar">┃</span> ';
  const noti = (title, lines) => [
    `${bar}<span class="t-ok">${title}</span>`,
    ...lines.map((l) => `${bar}${l}`),
  ];

  const scenes = [
    {
      label: "upload",
      cmd: `echo '{ "hello": "world" }' | ${sshTo(host)}`,
      output: [
        "",
        ...noti("File Uploaded 📤", [
          '<span class="t-muted">id:</span> <span class="t-white">y3ex1nbcje</span>',
          '<span class="t-muted">size:</span> <span class="t-white">24 B</span> <span class="t-muted">• type:</span> <span class="t-white">json</span> <span class="t-muted">• visibility:</span> <span class="t-white">public</span>',
        ]),
        "",
        ...noti("URL 🔗", [
          `<span class="t-url">${httpAddr}/f/y3ex1nbcje</span>`,
        ]),
        "",
        ...noti("SSH 📠", [
          `<span class="t-url">${sshTo(`f:y3ex1nbcje@${host}`)}</span>`,
        ]),
      ],
    },
    {
      label: "download",
      cmd: `${sshTo(`f:y3ex1nbcje@${host}`)} | jq .`,
      output: [
        '<span class="t-white">{</span>',
        `  <span class="t-url">"hello"</span><span class="t-white">:</span> <span class="t-ok">"world"</span>`,
        '<span class="t-white">}</span>',
      ],
    },
    {
      label: "private",
      cmd: `cat draft.md | ${sshTo(host)} -- -private -ttl 24h`,
      output: [
        "",
        ...noti("File Uploaded 📤", [
          '<span class="t-muted">id:</span> <span class="t-white">mkw2n8fhx0</span>',
          '<span class="t-muted">size:</span> <span class="t-white">118 B</span> <span class="t-muted">• type:</span> <span class="t-white">markdown</span> <span class="t-muted">• visibility:</span> <span class="t-red">private</span>',
        ]),
        "",
        ...noti("URL 🔐", [
          `<span class="t-url">${httpAddr}/f/mkw2n8fhx0?exp=1752…&amp;sig=NHo2…</span>`,
        ]),
        "",
        ...noti("Expiration ⌛", [
          '<span class="t-muted">24 hours from now</span>',
        ]),
      ],
    },
    {
      label: "tui",
      cmd: sshTo(host),
      output: [
        "",
        `<span class="t-white">snips.sh</span> <span class="t-muted">· 3 snips</span>`,
        "",
        '<span class="t-url">❯ y3ex1nbcje</span>   <span class="t-white">json</span>    <span class="t-muted">24 B   just now</span>',
        '  <span class="t-white">mkw2n8fhx0</span>   <span class="t-white">md</span>     <span class="t-muted">118 B   1 minute ago</span>',
        '  <span class="t-white">qp0d81xk2v</span>   <span class="t-white">go</span>    <span class="t-muted">1.9 kB   3 days ago</span>',
        "",
        '<span class="t-muted">↑/↓ navigate · enter view · q quit</span>',
      ],
    },
  ];

  const reducedMotion = window.matchMedia(
    "(prefers-reduced-motion: reduce)",
  ).matches;

  const HOLD_MS = 5000;
  const TYPE_MS = 32;
  const LINE_MS = 90;

  let timers = [];
  let autoCycle = !reducedMotion;
  let current = 0;

  const clearTimers = () => {
    timers.forEach(clearTimeout);
    timers = [];
  };

  const later = (fn, ms) => {
    timers.push(setTimeout(fn, ms));
  };

  const prompt = '<span class="t-prompt">$</span> ';
  const cursor = '<span class="t-cursor"></span>';

  const tabs = [];

  const scheduleNext = () => {
    if (!autoCycle) return;
    later(() => {
      if (document.hidden) {
        scheduleNext();
        return;
      }
      showScene((current + 1) % scenes.length, true);
    }, HOLD_MS);
  };

  const revealOutput = (scene, typed) => {
    let shown = 0;
    const revealLine = () => {
      shown++;
      const lines = scene.output.slice(0, shown).join("\n");
      const done = shown >= scene.output.length;
      screen.innerHTML = `${prompt}${typed}\n${lines}${done ? `\n${prompt}${cursor}` : ""}`;
      if (done) {
        scheduleNext();
      } else {
        later(revealLine, LINE_MS);
      }
    };
    revealLine();
  };

  const showScene = (index, animate) => {
    clearTimers();
    current = index;
    tabs.forEach((tab, i) => {
      tab.setAttribute("aria-pressed", i === index ? "true" : "false");
    });

    const scene = scenes[index];
    const full = esc(scene.cmd);

    if (!animate || reducedMotion) {
      screen.innerHTML = `${prompt}${full}\n${scene.output.join("\n")}\n${prompt}${cursor}`;
      scheduleNext();
      return;
    }

    let pos = 0;
    const chars = [...scene.cmd];
    const typeChar = () => {
      pos++;
      screen.innerHTML = `${prompt}${esc(chars.slice(0, pos).join(""))}${cursor}`;
      if (pos < chars.length) {
        later(typeChar, TYPE_MS + Math.random() * 40);
      } else {
        later(() => revealOutput(scene, full), 350);
      }
    };
    typeChar();
  };

  const tabContainer = document.querySelector("#terminal-tabs");
  scenes.forEach((scene, i) => {
    const tab = document.createElement("button");
    tab.className = "terminal-tab";
    tab.textContent = scene.label;
    tab.type = "button";
    tab.setAttribute("aria-pressed", "false");
    tab.addEventListener("click", () => {
      autoCycle = false;
      showScene(i, !reducedMotion);
    });
    tabContainer.appendChild(tab);
    tabs.push(tab);
  });

  showScene(0, true);
};

const initLandingCopyButtons = () => {
  document.querySelectorAll(".hero-copy, .recipe-copy").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const code = btn.parentElement.querySelector("code");
      if (!code) return;
      await navigator.clipboard.writeText(code.textContent);
      btn.classList.add("copied");
      setTimeout(() => {
        btn.classList.remove("copied");
      }, 1200);
    });
  });
};

window.addEventListener("DOMContentLoaded", () => {
  initLanding();
  initLandingCopyButtons();
});
