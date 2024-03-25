import Navigationmenu from "./ui/components/navigationmenu";
import LoginButton from "./ui/components/loginbutton";
import RegisterButton from "./ui/components/registerbutton";

// Define custom elements
window.customElements.define('navigation-menu', Navigationmenu);
window.customElements.define('login-button', LoginButton);
window.customElements.define('register-button', RegisterButton);

// Register form reset HTMX extension
interface Htmx {
  defineExtension(name: string, ext: any): void,
}
declare var htmx: Htmx
htmx.defineExtension('form-reset-on-success', {
  onEvent: function(name, event) {
    if (name !== 'htmx:afterRequest') return;
    if (!event.detail.successful) return;

    const triggeringElt = event.detail.requestConfig.elt;
    if (!triggeringElt.closest('[hx-form-reset-on-success]')) return;

    triggeringElt.reset();
  }
});
