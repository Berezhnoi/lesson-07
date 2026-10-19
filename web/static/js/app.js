document.body.addEventListener("htmx:configRequest", (event) => {
  const token = document.querySelector('meta[name="csrf-token"]')?.content;
  if (token) event.detail.headers["X-CSRF-Token"] = token;
});

document.body.addEventListener("htmx:responseError", (event) => {
  const messages = {
    401: "Please log in",
    403: "You are not allowed to do that",
    404: "Not found",
  };
  const toast = document.getElementById("toast");
  if (toast) toast.textContent = messages[event.detail.xhr.status] || "Something went wrong";
});

document.body.addEventListener("htmx:afterSwap", (event) => {
  if (event.detail.target?.id !== "movie-list") return;

  const pager = document.getElementById("pager");
  const hint = document.getElementById("pager-hint");
  if (pager) pager.hidden = true;
  if (hint) hint.hidden = false;
});
