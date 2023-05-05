const getSelectedLines = () => {
  if (!location.hash?.startsWith("#L")) return [];
  return location.hash
    .slice(1)
    .split("-")
    .map((n) => parseInt(n.slice(1)))
    .filter((e) => !isNaN(e))
    .sort((a, b) => a - b);
};

const highlightLines = () => {
  [...document.querySelectorAll(".hl")].forEach((el) => {
    el.classList.remove("hl");
  });

  const hash = location.hash;
  if (!hash) return;

  const lines = getSelectedLines();
  if (!lines.length) return;

  const start = lines[0];
  const end = lines[1] || start;

  for (let i = start; i <= end; i++) {
    const el = document.querySelector(`#L${i}`);
    if (!el) return;
    el.parentElement.classList.add("hl");
  }
};

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

const setToTopButton = () => {
  const element = document.querySelector("#to-top");
  if (!element) return;

  const parent = element.parentElement;
  if (!parent) return;

  const { top } = parent.getBoundingClientRect();

  if (top === 0) {
    element.removeAttribute("data-hide");
  } else {
    element.setAttribute("data-hide", "");
  }
};

window.addEventListener("scroll", setToTopButton);
window.addEventListener("hashchange", highlightLines);
window.addEventListener("load", () => {
  watchForShiftClick();
  highlightLines();
  setToTopButton();
});
