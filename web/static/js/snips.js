import {
  createIcons,
  Terminal,
  FileCode,
  HardDrive,
  SquarePen,
  HatGlasses,
  Folder,
  FileText,
} from "lucide";

// getSelectedLines will return the lines specified in the hash.
const getSelectedLines = () => {
  if (!location.hash.startsWith("#L")) return [];
  return location.hash
    .slice(1)
    .split("-")
    .map((n) => parseInt(n.slice(1)))
    .filter((e) => !isNaN(e))
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

    const lineNum = parseInt(el.href.split("#")[1].slice(1));
    if (isNaN(lineNum)) return;

    const lines = getSelectedLines();
    switch (lines.length) {
      case 0:
        location.hash = `#L${lineNum}`;
      case 1:
        if (lineNum < lines[0]) {
          lines.unshift(lineNum);
        } else {
          lines.push(lineNum);
        }
        location.hash = `#L${lines[0]}-L${lines[1]}`;
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
      default:
        return;
    }
  });
};

const initMermaid = async () => {
  if (!document.querySelector("code.language-mermaid")) return;

  const { default: mermaid } = await import("mermaid");
  const isLight =
    document.documentElement.getAttribute("data-theme") === "light";
  mermaid.initialize({
    startOnLoad: false,
    theme: isLight ? "default" : "dark",
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

  await initMermaid();
});
