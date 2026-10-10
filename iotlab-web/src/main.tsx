import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

// Astryx setup — required, else components render unstyled.
import "@astryxdesign/core/reset.css";
import "@astryxdesign/core/astryx.css";
// Theme (fonts, brand palette) — layer astryx-theme, overrides core defaults.
import "@astryxdesign/theme-neutral/theme.css";
// Figtree loads from Google Fonts — see the <link> tags in index.html.
// App base styles.
import "./styles/global.css";

// i18n side-effect init (en/vi) — must load before any component renders.
import "./i18n";

import App from "./App";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
