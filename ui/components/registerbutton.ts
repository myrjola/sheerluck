import {registerUser} from "./util/webauthn.js";

class RegisterButton extends HTMLElement {
  connectedCallback() {
    const button = this.querySelector("button");
    button.addEventListener("click", registerUser)
  }
}

export default RegisterButton;
