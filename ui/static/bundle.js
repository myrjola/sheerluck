(() => {
  // ui/components/navigationmenu.ts
  var Navigationmenu = class extends HTMLElement {
    connectedCallback() {
      const button = this.querySelector("#mobilemenu");
      const dialog = this.querySelector("dialog");
      button.addEventListener("click", () => {
        dialog.showModal();
      });
    }
  };
  var navigationmenu_default = Navigationmenu;

  // ui/components/custom-elements.ts
  window.customElements.define("navigation-menu", navigationmenu_default);
})();
