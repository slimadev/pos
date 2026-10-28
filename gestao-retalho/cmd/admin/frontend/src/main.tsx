import React from "react";
import { createRoot } from "react-dom/client";
import "./App.css";
import "./mobile.css";
import App from "./App";
import { ativarTabelasResponsivas } from "./tabelasResponsivas";

ativarTabelasResponsivas();

const container = document.getElementById("root");
const root = createRoot(container!);

root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
