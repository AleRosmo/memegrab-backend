import { showMenu } from "./topnav.js"
import { ready } from "./utils.js"

(() => {
    ready(() => { 
        let menuButton = document.getElementById("menuButton") 
        menuButton.addEventListener('click', showMenu, false)
    })
})()