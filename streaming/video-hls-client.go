package streaming

import (
	"fmt"
	"net/http"
)

func (s *HLSServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>HLS Video Streaming Demo</title>
    <style>
        body {
            font-family: Arial;
            max-width: 1200px;
            margin: 50px auto;
            padding: 20px;
            background: #1a1a1a;
            color: #fff;
        }
        h1 { color: #ff6b6b; }
        .container {
            background: #2d2d2d;
            padding: 30px;
            border-radius: 10px;
            margin: 20px 0;
        }
        video {
            width: 100%;
            max-width: 800px;
            border-radius: 5px;
            box-shadow: 0 4px 20px rgba(0,0,0,0.5);
        }
        .info {
            margin: 20px 0;
            padding: 15px;
            background: #3d3d3d;
            border-radius: 5px;
            border-left: 4px solid #4CAF50;
        }
        .stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
            margin: 20px 0;
        }
        .stat-box {
            background: #3d3d3d;
            padding: 15px;
            border-radius: 5px;
            text-align: center;
        }
        .stat-value {
            font-size: 24px;
            font-weight: bold;
            color: #4CAF50;
        }
        .stat-label {
            color: #888;
            margin-top: 5px;
        }
    </style>
    <script src="https://cdn.jsdelivr.net/npm/hls.js@latest"></script>
</head>
<body>
    <h1>HLS Video Streaming Demo</h1>
    
    <div class="container">
        <video id="video" controls></video>
        
        <div class="info">
            <h3>How HLS Adaptive Streaming Works</h3>
            <ul>
                <li>The video is split into segments (each segment is 6 seconds long).</li>
                <li>There are three quality levels: 720p (2.5 Mbps), 480p (1 Mbps), and 360p (500 Kbps).</li>
                <li>The player automatically selects the appropriate quality based on the network speed.</li>
                <li>When the network is slow → it switches down to 360p to avoid buffering.</li>
                <li>When the network is fast → it switches up to 720p for better clarity.</li>
            </ul>
        </div>

        <div class="stats">
            <div class="stat-box">
                <div class="stat-value" id="quality">-</div>
                <div class="stat-label">Current Quality</div>
            </div>
            <div class="stat-box">
                <div class="stat-value" id="buffer">-</div>
                <div class="stat-label">Buffer (seconds)</div>
            </div>
            <div class="stat-box">
                <div class="stat-value" id="bandwidth">-</div>
                <div class="stat-label">Bandwidth</div>
            </div>
            <div class="stat-box">
                <div class="stat-value" id="dropped">0</div>
                <div class="stat-label">Dropped Frames</div>
            </div>
        </div>
    </div>

    <script>
        const video = document.getElementById('video');
        const qualityEl = document.getElementById('quality');
        const bufferEl = document.getElementById('buffer');
        const bandwidthEl = document.getElementById('bandwidth');
        const droppedEl = document.getElementById('dropped');

        if (Hls.isSupported()) {
            const hls = new Hls({
                debug: false,
                enableWorker: true,
                lowLatencyMode: false,
            });

            hls.loadSource('/hls/master.m3u8');
            hls.attachMedia(video);

            // Track quality changes
            hls.on(Hls.Events.LEVEL_SWITCHED, function(event, data) {
                const level = hls.levels[data.level];
                qualityEl.textContent = level.height + 'p';
            });

            // Track bandwidth
            hls.on(Hls.Events.FRAG_LOADED, function(event, data) {
                const bandwidth = (data.frag.loaded * 8 / data.frag.duration / 1000).toFixed(0);
                bandwidthEl.textContent = bandwidth + ' Kbps';
            });

            // Update buffer info
            setInterval(() => {
                if (video.buffered.length > 0) {
                    const buffer = video.buffered.end(0) - video.currentTime;
                    bufferEl.textContent = buffer.toFixed(1);
                }

                // Dropped frames
                if (video.getVideoPlaybackQuality) {
                    const quality = video.getVideoPlaybackQuality();
                    droppedEl.textContent = quality.droppedVideoFrames;
                }
            }, 1000);

        } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
            // Native HLS support (Safari)
            video.src = '/hls/master.m3u8';
        }

        video.addEventListener('error', function(e) {
            console.error('Video error:', e);
        });
    </script>
</body>
</html>
`
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}
