/**
 * @param base64String {string}
 * @returns {Uint8Array}
 */
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

/**
 * @param value {ArrayBuffer}
 * @returns {string}
 */
function bufferEncode(value) {
  return btoa(String.fromCharCode.apply(null, new Uint8Array(value)))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=/g, "");
}

/**
 * @param e {Event}
 */
export async function registerUser(e) {
  e.preventDefault()
  const data = new FormData(e.target)
  const headers = {'X-CSRF-Token': data.get('csrf_token')}
  const registerStartURLPath = e.target.action
  const resp = await fetch(registerStartURLPath, {method: "post", headers})

  if (!resp.ok) {
    throw new Error(`Failed to start registration!`);
  }

  const credentialCreationOptions = await resp.json()

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
  const credential = await navigator.credentials.create({
    publicKey: credentialCreationOptions.publicKey,
  });

  let attestationObject = credential.response.attestationObject;
  let clientDataJSON = credential.response.clientDataJSON;
  let rawId = credential.rawId;

  const finishResp = await fetch("/api/registration/finish", {
    method: "post",
    headers,
    body: JSON.stringify({
      id: credential.id,
      rawId: bufferEncode(rawId),
      type: credential.type,
      response: {
        attestationObject: bufferEncode(attestationObject),
        clientDataJSON: bufferEncode(clientDataJSON),
      },
    }),
  })

  if (!finishResp.ok) {
    throw new Error("Finishing registration failed!");
  }

  window.location.reload();
}

/**
 * @param e {Event}
 */
export async function loginUser(e) {
  e.preventDefault()
  const data = new FormData(e.target)
  const headers = {'X-CSRF-Token': data.get('csrf_token')}
  const loginStartURLPath = e.target.action
  const resp = await fetch(loginStartURLPath, {method: "post", headers})
  const credentialRequestOptions = await resp.json()

  credentialRequestOptions.publicKey.challenge = bufferDecode(
    credentialRequestOptions.publicKey.challenge,
  );
  const assertion = await navigator.credentials.get({
    publicKey: credentialRequestOptions.publicKey,
  });

  let authData = assertion.response.authenticatorData;
  let clientDataJSON = assertion.response.clientDataJSON;
  let rawId = assertion.rawId;
  let sig = assertion.response.signature;
  let userHandle = assertion.response.userHandle;

  const finishResp = await fetch("/api/login/finish", {
    method: "post",
    headers,
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
  })

  if (!finishResp.ok) {
    throw new Error('Finishing login failed!');
  }

  window.location.reload();
}
