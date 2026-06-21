// Main.js
console.log("Main.js loaded successfully.");

document.addEventListener("DOMContentLoaded", () => {
    const form = document.getElementById("postForm");
    const submitBtn = document.getElementById("submitBtn");

    if (form && submitBtn) {
        form.addEventListener("submit", (event) => {
            // 1. Immediately disable the button to prevent multiple submissions
            submitBtn.disabled = true;
            
            // 2. Change text so the user knows a large file is processing
            submitBtn.innerText = "Uploading File... Please wait.";
            submitBtn.style.opacity = "0.6";
            submitBtn.style.cursor = "not-allowed";
        });
    }
});
