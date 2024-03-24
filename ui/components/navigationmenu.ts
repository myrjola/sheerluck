class Navigationmenu extends HTMLElement {
  connectedCallback() {
    const button = this.querySelector("button");
    const dialog = this.querySelector("dialog");
    button.addEventListener("click", () => {
      dialog.showModal()
    })
  }
}

export default Navigationmenu;
