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

  // ui/components/util/webauthn.js
  function bufferDecode(base64String) {
    const padding = "=".repeat((4 - base64String.length % 4) % 4);
    const base64 = (base64String + padding).replace(/-/g, "+").replace(/_/g, "/");
    const rawData = window.atob(base64);
    const outputArray = new Uint8Array(rawData.length);
    for (let i = 0; i < rawData.length; ++i) {
      outputArray[i] = rawData.charCodeAt(i);
    }
    return outputArray;
  }
  function bufferEncode(value) {
    return btoa(String.fromCharCode.apply(null, new Uint8Array(value))).replace(/\+/g, "-").replace(/\//g, "_").replace(/=/g, "");
  }
  function registerUser() {
    fetch("/api/registration/start", { method: "post" }).then((resp) => resp.json()).then((credentialCreationOptions) => {
      credentialCreationOptions.publicKey.challenge = bufferDecode(
        credentialCreationOptions.publicKey.challenge
      );
      credentialCreationOptions.publicKey.user.id = bufferDecode(
        credentialCreationOptions.publicKey.user.id
      );
      if (credentialCreationOptions.publicKey.excludeCredentials) {
        for (var i = 0; i < credentialCreationOptions.publicKey.excludeCredentials.length; i++) {
          credentialCreationOptions.publicKey.excludeCredentials[i].id = bufferDecode(
            credentialCreationOptions.publicKey.excludeCredentials[i].id
          );
        }
      }
      return navigator.credentials.create({
        publicKey: credentialCreationOptions.publicKey
      });
    }).then((credential) => {
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
            clientDataJSON: bufferEncode(clientDataJSON)
          }
        })
      }).then((resp) => {
        if (!resp.ok) {
          throw new Error("Finishing registration failed!");
        }
      });
    }).then((success) => {
      window.location.reload();
    }).catch((error) => {
      console.log(error);
      alert("failed to register");
    });
  }
  function loginUser() {
    fetch("/api/login/start", { method: "post" }).then((resp) => resp.json()).then((credentialRequestOptions) => {
      credentialRequestOptions.publicKey.challenge = bufferDecode(
        credentialRequestOptions.publicKey.challenge
      );
      return navigator.credentials.get({
        publicKey: credentialRequestOptions.publicKey
      });
    }).then((assertion) => {
      let authData = assertion.response.authenticatorData;
      let clientDataJSON = assertion.response.clientDataJSON;
      let rawId = assertion.rawId;
      let sig = assertion.response.signature;
      let userHandle = assertion.response.userHandle;
      return fetch("/api/login/finish", {
        method: "post",
        body: JSON.stringify({
          id: assertion.id,
          rawId: bufferEncode(rawId),
          type: assertion.type,
          response: {
            authenticatorData: bufferEncode(authData),
            clientDataJSON: bufferEncode(clientDataJSON),
            signature: bufferEncode(sig),
            userHandle: bufferEncode(userHandle)
          }
        })
      }).then((resp) => {
        if (!resp.ok) {
          throw new Error(`failed response: ${resp.status} ${resp.statusText}`);
        }
        window.location.reload();
      });
    }).catch((error) => {
      console.log(error);
      alert("failed to login!");
    });
  }

  // ui/components/loginbutton.ts
  var LoginButton = class extends HTMLElement {
    connectedCallback() {
      const button = this.querySelector("button");
      button.addEventListener("click", loginUser);
    }
  };
  var loginbutton_default = LoginButton;

  // ui/components/registerbutton.ts
  var RegisterButton = class extends HTMLElement {
    connectedCallback() {
      const button = this.querySelector("button");
      button.addEventListener("click", registerUser);
    }
  };
  var registerbutton_default = RegisterButton;

  // main.ts
  window.customElements.define("navigation-menu", navigationmenu_default);
  window.customElements.define("login-button", loginbutton_default);
  window.customElements.define("register-button", registerbutton_default);
  htmx.defineExtension("form-reset-on-success", {
    onEvent: function(name, event) {
      if (name !== "htmx:afterRequest")
        return;
      console.log(event);
      if (!event.detail.successful)
        return;
      const triggeringElt = event.detail.requestConfig.elt;
      console.log(triggeringElt);
      console.log("closest: ", triggeringElt.closest("[hx-form-reset-on-success]"));
      if (!triggeringElt.closest("[hx-form-reset-on-success]"))
        return;
      console.log("reset");
      triggeringElt.reset();
    }
  });
})();
