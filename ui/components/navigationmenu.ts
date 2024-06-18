class Navigationmenu extends HTMLElement {
  connectedCallback() {
    const openButton = this.querySelector("#openmobilemenu");
    const closeButton = this.querySelector("#closemobilemenu");
    const dialog = this.querySelector("dialog");
    openButton.addEventListener("click", () => {
      dialog.showModal()
    })
    closeButton.addEventListener("click", () => {
      dialog.close()
    })
  }
}

export default Navigationmenu;
