class Navigationmenu extends HTMLElement {
  connectedCallback() {
    const button = this.querySelector("#mobilemenu");
    const dialog = this.querySelector("dialog");
    button.addEventListener("click", () => {
      dialog.showModal()
    })
  }
}

export default Navigationmenu;
