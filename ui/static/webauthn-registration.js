/*
MIT License

Copyright (c) 2021 TechWebAuthn

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/
import { a as t, b as e } from "./utils/parse.js";
class i extends HTMLElement {
  constructor() {
    super(),
      (this.root = this.attachShadow({ mode: "open" })),
      (this._onFormSubmitListener = this._onFormSubmit.bind(this)),
      (this.registrationStartUrl = "/api/registration/start"),
      (this.registrationFinishUrl = "/api/registration/finish"),
      (this.fetchOptions = {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
      });
  }
  static get observedAttributes() {
    return ["no-username", "label", "input-type", "input-name", "button-text"];
  }
  connectedCallback() {
    this.update(),
      this.root
        .querySelector("form")
        .addEventListener("submit", this._onFormSubmitListener);
  }
  disconnectedCallback() {
    this.root
      .querySelector("form")
      .removeEventListener("submit", this._onFormSubmitListener);
  }
  attributeChangedCallback(t, e, i) {
    if (!this.root.innerHTML) return;
    if (i === e) return;
    const r = this.root.querySelector("label"),
      n = this.root.querySelector("input"),
      s = this.root.querySelector("button");
    switch (t) {
      case "no-username":
        this._shouldUseUsername();
        break;
      case "label":
        r.textContent = i || this.label;
        break;
      case "button-text":
        s.textContent = i || this.buttonText;
        break;
      case "input-type":
        n.type = i || this.inputType;
        break;
      case "input-name":
        n.name = i || this.inputName;
    }
  }
  update() {
    if (!this.root.querySelector("form")) {
      const t = new DOMParser()
        .parseFromString(
          `\n            <template>\n              <form part="form">\n                <label part="label" for="webauthn-username">${this.label}</label>\n                <input part="input" id="webauthn-username" type="${this.inputType}" name="${this.inputName}" />\n                <button part="button" type="submit">${this.buttonText}</button>\n              </form>\n            </template>\n          `,
          "text/html",
        )
        .querySelector("template");
      this.root.replaceChildren(t.content.cloneNode(!0));
    }
    this._shouldUseUsername();
  }
  get noUsername() {
    return this.hasAttribute("no-username");
  }
  set noUsername(t) {
    t
      ? this.setAttribute("no-username", "")
      : this.removeAttribute("no-username");
  }
  get label() {
    return this.getAttribute("label") || "Username";
  }
  set label(t) {
    this.setAttribute("label", t);
  }
  get buttonText() {
    return this.getAttribute("button-text") || "Register";
  }
  set buttonText(t) {
    this.setAttribute("button-text", t);
  }
  get inputType() {
    return this.getAttribute("input-type") || "text";
  }
  set inputType(t) {
    this.setAttribute("input-type", t);
  }
  get inputName() {
    return this.getAttribute("input-name") || "username";
  }
  set inputName(t) {
    this.setAttribute("input-name", t);
  }
  _shouldUseUsername() {
    const t = this.root.querySelector("input"),
      e = this.root.querySelector("label");
    this.noUsername
      ? ((t.required = !1), (t.hidden = !0), (e.hidden = !0), (t.value = ""))
      : ((t.required = !0), (t.hidden = !1), (e.hidden = !1));
  }
  async _getPublicKeyCredentialCreateOptionsDecoder() {
    return "function" == typeof this.publicKeyCredentialCreateOptionsDecoder
      ? this.publicKeyCredentialCreateOptionsDecoder
      : t;
  }
  async _getRegisterCredentialEncoder() {
    return "function" == typeof this.registerCredentialEncoder
      ? this.registerCredentialEncoder
      : e;
  }
  async _onFormSubmit(t) {
    try {
      if ((t.preventDefault(), !window.PublicKeyCredential))
        throw new Error("Web Authentication is not supported on this platform");
      debugger;
      this.dispatchEvent(new CustomEvent("registration-started"));
      const e = new FormData(t.target).get(this.inputName),
        i = await fetch(this.registrationStartUrl, {
          ...this.fetchOptions,
          body: JSON.stringify({ username: e }),
        }),
        {
          status: r,
          registrationId: n,
          publicKeyCredentialCreationOptions: s,
        } = await i.json();
      if (!i.ok)
        throw new Error(r || "Could not successfuly start registration");
      const o = await this._getPublicKeyCredentialCreateOptionsDecoder(),
        a = await navigator.credentials.create({ publicKey: o(s) });
      this.dispatchEvent(new CustomEvent("registration-created"));
      const u = await this._getRegisterCredentialEncoder(),
        h = await fetch(this.registrationFinishUrl, {
          ...this.fetchOptions,
          body: JSON.stringify({
            registrationId: n,
            credential: u(a),
            userAgent: window.navigator.userAgent,
          }),
        });
      if (!h.ok) throw new Error("Could not successfuly complete registration");
      const l = await h.json();
      this.dispatchEvent(
        new CustomEvent("registration-finished", { detail: l }),
      );
    } catch (t) {
      this.dispatchEvent(
        new CustomEvent("registration-error", {
          detail: { message: t.message },
        }),
      );
    }
  }
}
window.customElements.define("webauthn-registration", i);
export { i as WebAuthnRegistration };
