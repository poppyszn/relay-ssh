import { mount } from "svelte";
import "./theme.css";
import App from "./App.svelte";
import { installPlainInputs } from "./lib/plain_inputs";

installPlainInputs();
mount(App, { target: document.getElementById("app")! });
