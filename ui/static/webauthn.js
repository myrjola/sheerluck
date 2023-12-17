// URLBase64 to Uint8Array
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
      window.location.reload();
    })
    .catch((error) => {
      console.log(error);
      alert("failed to register");
    });
}

function loginUser() {
  fetch("/api/login/start")
    .then((resp) => resp.json())
    .then((credentialRequestOptions) => {
      console.log(credentialRequestOptions);
      credentialRequestOptions.publicKey.challenge = bufferDecode(
        credentialRequestOptions.publicKey.challenge,
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
            userHandle: bufferEncode(userHandle),
          },
        }),
      }).then((resp) => {
        if (!resp.ok) {
          throw new Error(`failed response: ${resp.status} ${resp.statusText}`);
        }
        window.location.reload();
        return;
      });
    })
    .catch((error) => {
      console.log(error);
      alert("failed to login!");
    });
}
