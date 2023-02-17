export function showMenu(event) {
    event.preventDefault()
    let topnav = document.getElementById("mainNav")
    if (topnav.className === "topnav") {
        topnav.classList.add("responsive")
    } else {
        topnav.className = "topnav"
    }
}