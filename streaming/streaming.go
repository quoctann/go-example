package streaming

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

/*
Note

- Flusher: đẩy data ngay lập tức đến client
- Context.Done(): client disconnected
- "Transfer-Encoding: chunked": cho phép streaming ko biết trước kích thước
- text/event-stream: content type cho SSE

Test SSE
curl -N http://localhost:8080/stream/sse

Test Chunked
curl http://localhost:8080/stream/chunked

Test File Stream
curl http://localhost:8080/stream/file

*/

type Message struct {
	ID        int       `json:"id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

/*
Server Sent Event (SSE) streaming endpoint.
Real time updates from server.
Auto reconnect when lose connection.
Use for notification/updates.
*/
func handleSSEStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	log.Printf("Client connected: %s\n", r.RemoteAddr)

	// Stream 10 messages, every 1 sec
	for i := 1; i <= 10; i++ {
		select {
		case <-r.Context().Done():
			log.Printf("Client disconnected: %s\n", r.RemoteAddr)
			return

		default:
			msg := Message{
				ID:        i,
				Content:   fmt.Sprintf("Message No. %d", i),
				Timestamp: time.Now(),
			}
			data, _ := json.Marshal(msg)

			fmt.Fprintf(w, "Data: %s\n\n", data)
			flusher.Flush()
			time.Sleep(1 * time.Second)
		}
	}

	fmt.Fprintf(w, "Event: done\nData: Stream completed\n\n")
	flusher.Flush()
	log.Printf("Stream complete for: %s\n", r.RemoteAddr)
}

/*
Chunked Transfer Encoding streaming.
Stream data in trunks.
No need to know content-length.
Use for large API response.
*/
func handleChunkedStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	log.Printf("Chunked client connected %s", r.RemoteAddr)

	// Stream array of messages
	fmt.Fprintf(w, "[")
	for i := 1; i < 5; i++ {
		select {
		case <-r.Context().Done():
			log.Printf("Chunked client disconnected %s", r.RemoteAddr)
			return

		default:
			msg := Message{
				ID:        i,
				Content:   fmt.Sprintf("Chunked message %d", i),
				Timestamp: time.Now(),
			}
			data, _ := json.Marshal(msg)

			if i > 1 {
				fmt.Fprintf(w, ",")
			}
			fmt.Fprintf(w, "%s", data)

			flusher.Flush()
			time.Sleep(500 * time.Millisecond)
		}
	}
	fmt.Fprintf(w, "]")
	flusher.Flush()
	log.Printf("Chunked stream completed for: %s", r.RemoteAddr)
}

/*
File Streaming.
Stream content line by line.
Use for log files, csv processing.
*/
func handleFileStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Example
	lines := []string{
		"Line 1: First",
		"Line 2: 2",
		"Line 3: 3",
		"Line 4: 4",
		"Line 5: 5",
		"Line 6: 6",
		"Line 7: 7",
		"Line 8: End",
	}

	for _, line := range lines {
		select {
		case <-r.Context().Done():
			return
		default:
			fmt.Fprintf(w, "%s\n", line)
			flusher.Flush()
			time.Sleep(300 * time.Millisecond)
		}
	}
}

// Test Server Sent Event (SSE)
func handleIndex(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Go Streaming Demo</title>
    <style>
        body { font-family: Arial; max-width: 800px; margin: 50px auto; padding: 20px; }
        .section { margin: 20px 0; padding: 20px; border: 1px solid #ddd; border-radius: 5px; }
        button { padding: 10px 20px; margin: 5px; cursor: pointer; }
        #sse-output, #chunked-output, #file-output { 
            background: #f5f5f5; 
            padding: 10px; 
            margin: 10px 0; 
            min-height: 100px;
            border-radius: 3px;
        }
        .message { margin: 5px 0; padding: 5px; background: white; border-left: 3px solid #4CAF50; }
    </style>
</head>
<body>
    <h1>Go Streaming Examples</h1>
    
    <div class="section">
        <h2>1. Server-Sent Events (SSE)</h2>
        <button onclick="startSSE()">Start SSE Stream</button>
        <button onclick="stopSSE()">Stop Stream</button>
        <div id="sse-output"></div>
    </div>

    <div class="section">
        <h2>2. Chunked Transfer Encoding</h2>
        <button onclick="startChunked()">Start Chunked Stream</button>
        <div id="chunked-output"></div>
    </div>

    <div class="section">
        <h2>3. File Stream</h2>
        <button onclick="startFileStream()">Start File Stream</button>
        <div id="file-output"></div>
    </div>

    <script>
        let eventSource;

        function startSSE() {
            const output = document.getElementById('sse-output');
            output.innerHTML = '<div>Connecting...</div>';
            
            eventSource = new EventSource('/stream/sse');
            
            eventSource.onmessage = function(event) {
                const data = JSON.parse(event.data);
                output.innerHTML += '<div class="message">Message #' + data.id + ': ' + data.content + '</div>';
                output.scrollTop = output.scrollHeight;
            };

            eventSource.addEventListener('done', function(event) {
                output.innerHTML += '<div style="color: green; font-weight: bold;">✓ ' + event.data + '</div>';
                eventSource.close();
            });

            eventSource.onerror = function() {
                output.innerHTML += '<div style="color: red;">Error</div>';
                eventSource.close();
            };
        }

        function stopSSE() {
            if (eventSource) {
                eventSource.close();
                document.getElementById('sse-output').innerHTML += '<div style="color: orange;">Stream stopped</div>';
            }
        }

        async function startChunked() {
            const output = document.getElementById('chunked-output');
            output.innerHTML = '<div>Streaming...</div>';
            
            try {
                const response = await fetch('/stream/chunked');
                const reader = response.body.getReader();
                const decoder = new TextDecoder();
                let buffer = '';

                while (true) {
                    const {done, value} = await reader.read();
                    if (done) break;
                    
                    buffer += decoder.decode(value, {stream: true});
                }

                const messages = JSON.parse(buffer);
                output.innerHTML = '';
                messages.forEach(msg => {
                    output.innerHTML += '<div class="message">Message #' + msg.id + ': ' + msg.content + '</div>';
                });
            } catch (error) {
                output.innerHTML = '<div style="color: red;">Error: ' + error.message + '</div>';
            }
        }

        async function startFileStream() {
            const output = document.getElementById('file-output');
            output.innerHTML = '<div>Streaming file...</div>';
            
            try {
                const response = await fetch('/stream/file');
                const reader = response.body.getReader();
                const decoder = new TextDecoder();

                output.innerHTML = '';
                
                while (true) {
                    const {done, value} = await reader.read();
                    if (done) break;
                    
                    const text = decoder.decode(value, {stream: true});
                    output.innerHTML += text.replace(/\n/g, '<br>');
                    output.scrollTop = output.scrollHeight;
                }
            } catch (error) {
                output.innerHTML = '<div style="color: red;">Error: ' + error.message + '</div>';
            }
        }
    </script>
</body>
</html>
`
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

func RunExample(skip bool) {
	if skip {
		return
	}

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/stream/sse", handleSSEStream)
	http.HandleFunc("/stream/chunked", handleChunkedStream)
	http.HandleFunc("/stream/file", handleFileStream)

	port := ":8080"
	fmt.Printf("Test endpoint: http://localhost%s\n", port)

	log.Fatal(http.ListenAndServe(port, nil))
}
