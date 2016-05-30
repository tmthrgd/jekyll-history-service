hljs.initHighlightingOnLoad();

document.addEventListener("DOMContentLoaded", function () {
	var permalinkSpan = document.querySelector(".permalink");

	var permalink = document.createElement("a");
	permalink.classList.add("permalink");
	permalink.href = permalinkSpan.textContent || permalinkSpan.innerText;
	permalink.innerHTML = permalinkSpan.innerHTML;

	permalinkSpan.parentNode.replaceChild(permalink, permalinkSpan);

	var path = document.querySelector(".permalink-path");
	path.addEventListener("input", function () {
		permalink.href = (permalink.textContent || permalink.innerText) + (path.textContent || path.innerText);
	});
}, false);
