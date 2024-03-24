(() => {
  // ui/components/navigationmenu.ts
  var Navigationmenu = class extends HTMLElement {
    connectedCallback() {
      const button = this.querySelector("button");
      const dialog = this.querySelector("dialog");
      button.addEventListener("click", () => {
        dialog.showModal();
      });
    }
  };
  var navigationmenu_default = Navigationmenu;

  // ui/components/counter.ts
  var MyCounter = class extends HTMLElement {
    connectedCallback() {
      const tmpl = this.querySelector("template");
      tmpl.replaceWith(tmpl.content);
      const btn = this.querySelector("button");
      const output = this.querySelector("output");
      const input = this.querySelector("input");
      let value = parseInt(output.innerText);
      btn.addEventListener("click", () => {
        value++;
        output.innerText = value;
        input.value = value;
      });
    }
  };
  var counter_default = MyCounter;

  // ui/components/custom-elements.ts
  window.customElements.define("navigation-menu", navigationmenu_default);
  window.customElements.define("my-counter", counter_default);
})();
