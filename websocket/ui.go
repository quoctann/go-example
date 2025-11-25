package websocket

const htmlContent = `
<!DOCTYPE html>
<html>
	<head>
		<meta charset="utf-8" />
		<title>WebSocket Chat</title>
		<style>
			body {
				font-family: Arial;
				max-width: 800px;
				margin: 50px auto;
				padding: 20px;
			}
			#messages {
				border: 1px solid #ccc;
				height: 400px;
				overflow-y: scroll;
				padding: 10px;
				margin-bottom: 10px;
			}
			.message {
				margin: 5px 0;
				padding: 5px;
			}
			.join {
				color: green;
				font-style: italic;
			}
			.leave {
				color: red;
				font-style: italic;
			}
			input {
				padding: 10px;
				width: 70%;
			}
			button {
				padding: 10px 20px;
			}
		</style>
	</head>
	<body>
		<h1>WebSocket Chat Demo</h1>
		<div id="messages"></div>
		<input type="text" id="messageInput" placeholder="Nhập tin nhắn..." />
		<button onclick="sendMessage()">Gửi</button>

		<script>
			let ws;
			let username = prompt("Nhập tên của bạn:") || "Anonymous";

			function connect() {
				// Kết nối đến WebSocket server
				ws = new WebSocket(
					"ws://localhost:8080/ws?username=" + encodeURIComponent(username)
				);

				ws.onopen = function () {
					console.log("Đã kết nối WebSocket");
					addMessage("Hệ thống", "Đã kết nối thành công!", "join");
				};

				ws.onmessage = function (event) {
					let msg = JSON.parse(event.data);
					addMessage(msg.username, msg.content, msg.type);
				};

				ws.onclose = function () {
					console.log("WebSocket đã đóng");
					addMessage(
						"Hệ thống",
						"Kết nối đã đóng. Đang kết nối lại...",
						"leave"
					);
					setTimeout(connect, 2000);
				};

				ws.onerror = function (error) {
					console.error("WebSocket error:", error);
				};
			}

			function sendMessage() {
				let input = document.getElementById("messageInput");
				let content = input.value.trim();

				if (content && ws.readyState === WebSocket.OPEN) {
					let msg = {
						username: username,
						content: content,
						type: "message",
					};
					ws.send(JSON.stringify(msg));
					input.value = "";
				}
			}

			function addMessage(user, content, type) {
				let messages = document.getElementById("messages");
				let div = document.createElement("div");
				div.className = "message " + type;

				if (type === "message") {
					div.innerHTML = "<strong>" + user + ":</strong> " + content;
				} else {
					div.innerHTML = "<em>" + user + " " + content + "</em>";
				}

				messages.appendChild(div);
				messages.scrollTop = messages.scrollHeight;
			}

			// Gửi message khi nhấn Enter
			document
				.getElementById("messageInput")
				.addEventListener("keypress", function (e) {
					if (e.key === "Enter") {
						sendMessage();
					}
				});

			// Kết nối khi trang load
			connect();
		</script>
	</body>
</html>
`
