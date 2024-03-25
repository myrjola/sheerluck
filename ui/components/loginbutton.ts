import {loginUser} from "./util/webauthn.js";

class LoginButton extends HTMLElement {
  connectedCallback() {
    const button = this.querySelector("button");
    button.addEventListener("click", loginUser)
  }
}

export default LoginButton;
