import {
  createIcons,
  FileCode,
  FileText,
  Folder,
  HardDrive,
  HatGlasses,
  SquarePen,
  Terminal,
} from "lucide";

// getSelectedLines will return the lines specified in the hash.
const getSelectedLines = () => {
  if (!location.hash.startsWith("#L")) return [];
  return location.hash
    .slice(1)
    .split("-")
    .map((n) => parseInt(n.slice(1), 10))
    .filter((e) => !Number.isNaN(e))
    .sort((a, b) => a - b);
};

// highlightLines will highlight the lines specified in the hash.
const highlightLines = () => {
  [...document.querySelectorAll(".hl")].forEach((el) => {
    el.classList.remove("hl");
  });

  const [start, end = start] = getSelectedLines();
  if (!start) return;

  for (let i = start; i <= end; i++) {
    const el = document.querySelector(`#L${i}`);
    if (!el) return;
    el.parentElement.classList.add("hl");
  }
};

// scrollToLine will scroll to the selected lines on hash #L2
const scrollToLine = () => {
  const [start] = getSelectedLines();
  if (!start) return;

  // needs to defer the execution to be able to scroll even when page gets refresh
  setTimeout(() => {
    document.querySelector(`#L${start}`).scrollIntoView({ behavior: "smooth" });
  }, 100);
};

// watchForShiftClick watches for shift-clicks on line numbers, and will set the anchor appropriately.
const watchForShiftClick = () => {
  const chroma = document.querySelector(".chroma");
  if (!chroma) return;

  chroma.addEventListener("click", (event) => {
    if (!event.shiftKey) return;

    const el = event.target;
    if (!el.matches(".lnlinks")) return;

    event.preventDefault();

    const lineNum = parseInt(el.href.split("#")[1].slice(1), 10);
    if (Number.isNaN(lineNum)) return;

    const lines = getSelectedLines();
    switch (lines.length) {
      case 0:
        location.hash = `#L${lineNum}`;
        break;
      case 1:
        if (lineNum < lines[0]) {
          lines.unshift(lineNum);
        } else {
          lines.push(lineNum);
        }
        location.hash = `#L${lines[0]}-L${lines[1]}`;
        break;
      case 2:
        if (lineNum < lines[0]) {
          lines[1] = lines[0];
          lines[0] = lineNum;
        } else if (lineNum > lines[0] && lineNum < lines[1]) {
          lines[1] = lineNum;
        } else if (lineNum > lines[1]) {
          lines[1] = lineNum;
        }
        location.hash = `#L${lines[0]}-L${lines[1]}`;
        break;
      default:
        return;
    }
  });
};

const initMermaid = async () => {
  if (!document.querySelector("code.language-mermaid")) return;

  const { default: mermaid } = await import("mermaid");
  mermaid.initialize({
    startOnLoad: false,
    theme: "dark",
  });
  mermaid.run({ querySelector: "code.language-mermaid" });
};

// initHeaderObserver will hide the "to top" button when the top of the page is visible, and shows it when it's not.
const initHeaderObserver = () => {
  const element = document.querySelector("#to-top");
  if (!element) return;

  const nav = element.closest("nav");
  if (!nav) return;

  const observer = new IntersectionObserver(
    ([entry]) => {
      element.toggleAttribute("data-hide", entry.isIntersecting);
    },
    // https://stackoverflow.com/a/61115077
    { rootMargin: "-1px 0px 0px 0px", threshold: [1] },
  );

  observer.observe(nav);

  // do not remove the hightlighted lines when scroll to the top
  element.addEventListener("click", (event) => {
    event.preventDefault();
    window.scrollTo({ top: 0, behavior: "smooth" });
  });
};

const initIcons = () => {
  createIcons({
    icons: {
      Terminal,
      FileCode,
      HardDrive,
      SquarePen,
      HatGlasses,
      Folder,
      FileText,
    },
  });
};

const initKeyboardShortcuts = () => {
  document.addEventListener("keydown", (event) => {
    // ignore if user is typing in an input or textarea
    if (event.target.matches("input, textarea, [contenteditable]")) return;
    // ignore if modifier keys are pressed
    if (event.metaKey || event.ctrlKey || event.altKey) return;

    const shortcutEl = document.querySelector(`[data-shortcut="${event.key}"]`);
    if (!shortcutEl) return;

    event.preventDefault();
    shortcutEl.click();
  });
};

const parseColor = (cssColor) => {
  const ctx = document.createElement("canvas").getContext("2d");
  ctx.fillStyle = cssColor;
  ctx.fillRect(0, 0, 1, 1);
  return ctx.getImageData(0, 0, 1, 1).data;
};

const updateFavicon = (cssColor) => {
  const img = new Image();
  img.crossOrigin = "anonymous";
  img.src = "/assets/img/favicon.png";
  img.onload = () => {
    const canvas = document.createElement("canvas");
    canvas.width = img.width;
    canvas.height = img.height;
    const ctx = canvas.getContext("2d");
    ctx.drawImage(img, 0, 0);
    const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
    const d = imageData.data;
    const [nr, ng, nb] = parseColor(cssColor);
    for (let i = 0; i < d.length; i += 4) {
      if (d[i + 3] === 0) continue;
      const isWhite = d[i] > 200 && d[i + 1] > 200 && d[i + 2] > 200;
      if (!isWhite) {
        d[i] = nr;
        d[i + 1] = ng;
        d[i + 2] = nb;
      }
    }
    ctx.putImageData(imageData, 0, 0);
    const link = document.querySelector("link[rel='icon']");
    if (link) link.href = canvas.toDataURL("image/png");
  };
};

const resolveColor = (colorName) =>
  getComputedStyle(document.documentElement)
    .getPropertyValue(`--color-${colorName}`)
    .trim();

const applyColor = (colorName) => {
  document.documentElement.style.setProperty(
    "--color-primary",
    `var(--color-${colorName})`,
  );
  updateFavicon(resolveColor(colorName));
};

const initColorPicker = () => {
  const swatches = document.querySelectorAll(".color-swatch");
  if (!swatches.length) return;

  const saved = localStorage.getItem("color-primary");
  if (saved) applyColor(saved);

  const active = saved || "blue";
  swatches.forEach((swatch) => {
    if (swatch.dataset.color === active) swatch.classList.add("active");

    swatch.addEventListener("click", () => {
      const color = swatch.dataset.color;
      localStorage.setItem("color-primary", color);
      swatches.forEach((s) => {
        s.classList.toggle("active", s === swatch);
      });
      applyColor(color);
    });
  });
};

const initCopyButton = () => {
  const copyBtn = document.querySelector("#copy-content");
  if (!copyBtn) return;

  copyBtn.addEventListener("click", async () => {
    const rawContent = document.querySelector("#raw-content");
    if (!rawContent) return;

    await navigator.clipboard.writeText(rawContent.textContent);

    const kbd = copyBtn.querySelector("kbd");
    copyBtn.textContent = "copied!";
    copyBtn.prepend(kbd);

    setTimeout(() => {
      copyBtn.textContent = "copy";
      copyBtn.prepend(kbd);
    }, 1500);
  });
};

window.addEventListener("hashchange", highlightLines);
window.addEventListener("DOMContentLoaded", async () => {
  initHeaderObserver();
  watchForShiftClick();
  highlightLines();
  scrollToLine();
  initIcons();
  initKeyboardShortcuts();
  initCopyButton();
  initColorPicker();

  await initMermaid();
});
