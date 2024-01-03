const mobileMenu = document.getElementById("mobile-menu");
const openMobileMenu = document.getElementById("open-mobile-menu");
const closeMobileMenu = document.getElementById("close-mobile-menu");
openMobileMenu.addEventListener("click", () => {
    mobileMenu.showModal();
});
closeMobileMenu.addEventListener("click", () => {
    mobileMenu.close();
});
