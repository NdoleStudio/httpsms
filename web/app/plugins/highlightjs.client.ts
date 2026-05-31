import hljs from "highlight.js/lib/core";
import javascript from "highlight.js/lib/languages/javascript";
import python from "highlight.js/lib/languages/python";
import php from "highlight.js/lib/languages/php";
import go from "highlight.js/lib/languages/go";
import java from "highlight.js/lib/languages/java";
import bash from "highlight.js/lib/languages/bash";
import csharp from "highlight.js/lib/languages/csharp";
import json from "highlight.js/lib/languages/json";
import "highlight.js/styles/github-dark.css";

hljs.registerLanguage("javascript", javascript);
hljs.registerLanguage("python", python);
hljs.registerLanguage("php", php);
hljs.registerLanguage("go", go);
hljs.registerLanguage("java", java);
hljs.registerLanguage("bash", bash);
hljs.registerLanguage("csharp", csharp);
hljs.registerLanguage("json", json);

export default defineNuxtPlugin((nuxtApp) => {
  nuxtApp.vueApp.directive("highlight", {
    mounted(el: HTMLElement) {
      el.querySelectorAll("pre code").forEach((block) => {
        hljs.highlightElement(block as HTMLElement);
      });
    },
    updated(el: HTMLElement) {
      el.querySelectorAll("pre code").forEach((block) => {
        delete (block as HTMLElement).dataset.highlighted;
        hljs.highlightElement(block as HTMLElement);
      });
    },
  });

  // Override hljs background to use Vuetify surface variant
  const style = document.createElement("style");
  style.textContent = `
    pre code.hljs {
      background: transparent;
      padding: 0;
    }
  `;
  document.head.appendChild(style);
});
