class MyCounter extends HTMLElement {
  connectedCallback() {
    const tmpl = this.querySelector("template");
    tmpl.replaceWith(tmpl.content);

    const btn = this.querySelector("button");
    const output = this.querySelector("output");
    const input = this.querySelector("input");

    let value = parseInt(output.innerText);
    btn.addEventListener("click", () => {
      value++;
      output.innerText = value.toString();
      input.value = value.toString();
    })
  }
}

export default MyCounter
