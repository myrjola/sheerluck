// URLBase64 to ArrayBuffer
function bufferDecode(base64String) {
  const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, "+").replace(/_/g, "/");

  const rawData = window.atob(base64);
  const outputArray = new Uint8Array(rawData.length);

  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }

  return outputArray;
}

// ArrayBuffer to URLBase64
function bufferEncode(value) {
  return btoa(String.fromCharCode.apply(null, new Uint8Array(value)))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=/g, "");
}

function registerUser() {
  fetch("/api/registration/start")
    .then((resp) => resp.json())
    .then((credentialCreationOptions) => {
      console.log(credentialCreationOptions);
      credentialCreationOptions.publicKey.challenge = bufferDecode(
        credentialCreationOptions.publicKey.challenge,
      );
      credentialCreationOptions.publicKey.user.id = bufferDecode(
        credentialCreationOptions.publicKey.user.id,
      );
      if (credentialCreationOptions.publicKey.excludeCredentials) {
        for (
          var i = 0;
          i < credentialCreationOptions.publicKey.excludeCredentials.length;
          i++
        ) {
          credentialCreationOptions.publicKey.excludeCredentials[i].id =
            bufferDecode(
              credentialCreationOptions.publicKey.excludeCredentials[i].id,
            );
        }
      }
      return navigator.credentials.create({
        publicKey: credentialCreationOptions.publicKey,
      });
    })
    .then((credential) => {
      console.log(credential);
      let attestationObject = credential.response.attestationObject;
      let clientDataJSON = credential.response.clientDataJSON;
      let rawId = credential.rawId;

      return fetch("/api/registration/finish", {
        method: "post",
        body: JSON.stringify({
          id: credential.id,
          rawId: bufferEncode(rawId),
          type: credential.type,
          response: {
            attestationObject: bufferEncode(attestationObject),
            clientDataJSON: bufferEncode(clientDataJSON),
          },
        }),
      }).then((resp) => {
        if (!resp.ok) {
          throw new Error("Finishing registration failed!");
        }
      });
    })
    .then((success) => {
      alert("successfully registered!");
    })
    .catch((error) => {
      console.log(error);
      alert("failed to register");
    });
}

function loginUser() {
  username = $("#email").val();
  if (username === "") {
    alert("Please enter a username");
    return;
  }

  $.get(
    "/login/begin/" + username,
    null,
    function (data) {
      return data;
    },
    "json",
  )
    .then((credentialRequestOptions) => {
      console.log(credentialRequestOptions);
      credentialRequestOptions.publicKey.challenge = bufferDecode(
        credentialRequestOptions.publicKey.challenge,
      );
      credentialRequestOptions.publicKey.allowCredentials.forEach(
        function (listItem) {
          listItem.id = bufferDecode(listItem.id);
        },
      );

      return navigator.credentials.get({
        publicKey: credentialRequestOptions.publicKey,
      });
    })
    .then((assertion) => {
      console.log(assertion);
      let authData = assertion.response.authenticatorData;
      let clientDataJSON = assertion.response.clientDataJSON;
      let rawId = assertion.rawId;
      let sig = assertion.response.signature;
      let userHandle = assertion.response.userHandle;

      $.post(
        "/login/finish/" + username,
        JSON.stringify({
          id: assertion.id,
          rawId: bufferEncode(rawId),
          type: assertion.type,
          response: {
            authenticatorData: bufferEncode(authData),
            clientDataJSON: bufferEncode(clientDataJSON),
            signature: bufferEncode(sig),
            userHandle: bufferEncode(userHandle),
          },
        }),
        function (data) {
          return data;
        },
        "json",
      );
    })
    .then((success) => {
      alert("successfully logged in " + username + "!");
      return;
    })
    .catch((error) => {
      console.log(error);
      alert("failed to register " + username);
    });
}

export class WebAuthnRegister extends HTMLElement {
  constructor() {
    super();
    this.root = this.attachShadow({ mode: "open" });
    this._onFormSubmitListener = this._onFormSubmit.bind(this);
    this.registrationStartUrl = "/api/registration/start";
    this.registrationFinishUrl = "/api/registration/finish";
    this.fetchOptions = {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
      },
    };
  }

  static get observedAttributes() {
    return ["no-username", "label", "input-type", "input-name", "button-text"];
  }

  connectedCallback() {
    this.update();
    this.root
      .querySelector("form")
      .addEventListener("submit", this._onFormSubmitListener);
  }

  disconnectedCallback() {
    this.root
      .querySelector("form")
      .removeEventListener("submit", this._onFormSubmitListener);
  }

  attributeChangedCallback(name, oldValue, newValue) {
    if (!this.root.innerHTML) return;
    if (newValue === oldValue) return;

    const label = this.root.querySelector("label");
    const input = this.root.querySelector("input");
    const button = this.root.querySelector("button");

    switch (name) {
      case "no-username":
        this._shouldUseUsername();
        break;
      case "label":
        label.textContent = newValue || this.label;
        break;
      case "button-text":
        button.textContent = newValue || this.buttonText;
        break;
      case "input-type":
        input.type = newValue || this.inputType;
        break;
      case "input-name":
        input.name = newValue || this.inputName;
        break;
    }
  }

  update() {
    if (!this.root.querySelector("form")) {
      const template = new DOMParser()
        .parseFromString(
          `
            <template>
              <form part="form">
                <label part="label" for="webauthn-username">${this.label}</label>
                <input part="input" id="webauthn-username" type="${this.inputType}" name="${this.inputName}" />
                <button part="button" type="submit">${this.buttonText}</button>
              </form>
            </template>
          `,
          "text/html",
        )
        .querySelector("template");

      this.root.replaceChildren(template.content.cloneNode(true));
    }

    this._shouldUseUsername();
  }

  get noUsername() {
    return this.hasAttribute("no-username");
  }

  set noUsername(value) {
    if (!value) {
      this.removeAttribute("no-username");
    } else {
      this.setAttribute("no-username", "");
    }
  }

  get label() {
    return this.getAttribute("label") || "Username";
  }

  set label(value) {
    this.setAttribute("label", value);
  }

  get buttonText() {
    return this.getAttribute("button-text") || "Register";
  }

  set buttonText(value) {
    this.setAttribute("button-text", value);
  }

  get inputType() {
    return this.getAttribute("input-type") || "text";
  }

  set inputType(value) {
    this.setAttribute("input-type", value);
  }

  get inputName() {
    return this.getAttribute("input-name") || "username";
  }

  set inputName(value) {
    this.setAttribute("input-name", value);
  }

  _shouldUseUsername() {
    const input = this.root.querySelector("input");
    const label = this.root.querySelector("label");

    if (this.noUsername) {
      input.required = false;
      input.hidden = true;
      label.hidden = true;
      input.value = "";
    } else {
      input.required = true;
      input.hidden = false;
      label.hidden = false;
    }
  }

  async _getPublicKeyCredentialCreateOptionsDecoder() {
    return typeof this.publicKeyCredentialCreateOptionsDecoder === "function"
      ? this.publicKeyCredentialCreateOptionsDecoder
      : decodePublicKeyCredentialCreateOptions;
  }

  async _getRegisterCredentialEncoder() {
    return typeof this.registerCredentialEncoder === "function"
      ? this.registerCredentialEncoder
      : encodeRegisterCredential;
  }

  async _onFormSubmit(event) {
    try {
      event.preventDefault();

      if (!window.PublicKeyCredential) {
        throw new Error("Web Authentication is not supported on this platform");
      }

      this.dispatchEvent(new CustomEvent("registration-started"));

      registerUser();
    } catch (error) {
      this.dispatchEvent(
        new CustomEvent("registration-error", {
          detail: { message: error.message },
        }),
      );
    }
  }
}

window.customElements.define("webauthn-register", WebAuthnRegister);
