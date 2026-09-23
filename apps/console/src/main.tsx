import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import "@fontsource-variable/geist";
import "./index.css";
import { App } from "./App";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { appName } from "@/lib/app-config";

document.title = appName;

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <BrowserRouter>
      <TooltipProvider delayDuration={200}>
        <App />
        <Toaster richColors position="top-right" />
      </TooltipProvider>
    </BrowserRouter>
  </React.StrictMode>,
);
