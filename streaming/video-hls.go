package streaming

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

/*

HLS = HTTP Live Streaming: chia video thành trunks (MB) gửi đi từng mảnh (giống utube, netflx)

WebRTC: p2p, độ trễ thấp, dùng cho conferencing, gaming, live

*/

type HLSServer struct {
	videoDir    string
	segmentDir  string
	segmentTime int // second in every segment
}

func newHLSServer() *HLSServer {
	return &HLSServer{
		videoDir:    "./resources/videos",
		segmentDir:  "./resources/hls_output",
		segmentTime: 6, // 6s per chunk
	}
}

func (s *HLSServer) setup() error {
	// create if not exist
	os.MkdirAll(s.videoDir, 0755)
	os.MkdirAll(s.segmentDir, 0755)

	sampleVid := filepath.Join(s.videoDir, "demo.mp4")
	if _, err := os.Stat(sampleVid); os.IsNotExist(err) {
		log.Println("Creating sample video...")
		if err := s.createDemoVideo(sampleVid); err != nil {
			return fmt.Errorf("cannot create demo video: %v", err)
		}
	}

	return nil
}

func (s *HLSServer) createDemoVideo(output string) error {
	// need ffmpeg installed: brew install ffmpeg | apt install ffmpeg
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi",
		"-i", "testsrc=duration=30:size=1280x720:rate=30",
		"-f", "lavfi",
		"-i", "sine=frequency=1000:duration=30",
		"-vf", "drawtext=text='HLS Demo Video':fontsize=50:fontcolor=white:x=(w-text_w)/2:y=(h-text_h)/2",
		"-c:v", "libx264",
		"-c:a", "aac",
		"-preset", "fast",
		"-t", "30",
		"-y",
		output,
	)

	return cmd.Run()
}

type variant struct {
	name       string
	resolution string
	videoBR    string
	audioBR    string
}

func (s *HLSServer) convertToHLS(videoPath string) error {
	log.Println("Converting video to HLS format...")

	outputDir := s.segmentDir
	masterPlaylist := filepath.Join(outputDir, "master.m3u8")

	// create 3 variants for adaptive streaming
	// 720p (high)
	// 480p (medium)
	// 360p (low)
	variants := []variant{
		{"720p", "1280x720", "2500k", "128k"},
		{"480p", "854x480", "1000k", "96k"},
		{"360p", "640x360", "500k", "64k"},
	}

	// covert for each variant
	for _, v := range variants {
		variantDir := filepath.Join(outputDir, v.name)
		os.MkdirAll(variantDir, 0755)

		playlistPath := filepath.Join(variantDir, "playlist.m3u8")

		log.Printf("Encoding %s ...", v.name)

		// need ffmpeg installed, careful to use command like this to prevent
		// command injection
		cmd := exec.Command("ffmpeg",
			"-i", videoPath,
			"-vf", fmt.Sprintf("scale=%s", v.resolution),
			"-c:v", "libx264",
			"-b:v", v.videoBR,
			"-c:a", "aac",
			"-b:a", v.audioBR,
			"-hls_time", fmt.Sprintf("%d", s.segmentTime),
			"-hls_playlist_type", "vod",
			"-hls_segment_filename", filepath.Join(variantDir, "segment_%03d.ts"),
			"-y",
			playlistPath,
		)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("encoding %s failed: %v", v.name, err)
		}
	}

	return s.createMasterPlaylist(masterPlaylist, variants)
}

// Create m3u8 master playlist file, in HLS Adaptive Bitrate Streaming. This is
// the playlist that browser should read it first to know what kind of video
// quality to adaptive switching
func (s *HLSServer) createMasterPlaylist(output string, variants []variant) error {
	log.Println("Create master playlist...")
	/*
		#EXTM3U
		#EXT-X-VERSION:3
		---> required header of all HLS files

		#EXT-X-STREAM-INF:BANDWIDTH=2500000,RESOLUTION=1280x720
		720p/playlist.m3u8
		---> available 720p (2.5Mbps)

		#EXT-X-STREAM-INF:BANDWIDTH=1000000,RESOLUTION=854x480
		480p/playlist.m3u8
		---> available 480p (1Mbps)

		#EXT-X-STREAM-INF:BANDWIDTH=500000,RESOLUTION=640x360
		360p/playlist.m3u8
		---> available 360 (0.5Mbps)
	*/

	content := "#EXTM3U\n#EXT-X-VERSION:3\n\n"
	for _, v := range variants {
		// parse bandwidth from string "2500k" -> 2500000
		bandwidth := strings.TrimSuffix(v.videoBR, "k")
		content += fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%s000,RESOLUTION=%s\n", bandwidth, v.resolution)
		content += fmt.Sprintf("%s/playlist.m3u8\n\n", v.name)
	}

	return os.WriteFile(output, []byte(content), 0644)
}

func (s *HLSServer) handleHLS(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	cleanPath := filepath.Clean(strings.TrimPrefix(path, "/hls/"))
	// prevent path traversal attack (../../etc/passwd)
	if strings.Contains(cleanPath, "..") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	/*
		set proper MIME types for HLS (MIME = Multipurpose Internet Mail Extensions)

		.m3u8	: application/vnd.apple.mpegurl | application/x-mpegURL
		.ts		: video/MP2T | video/mp2t
	*/
	if strings.HasSuffix(path, ".m3u8") {
		w.Header().Set("Content-Type", "application/vnd.apple.mpeurl")
	} else if strings.HasSuffix(path, ".ts") {
		w.Header().Set("Content-Type", "video/MP2T")
	}

	// CORS headers for cross-origin streaming

	// allow all other domains play this video in their side
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	// allow CDN (nginx, browser,...) cache HLS file for 1H -> reduce server overload
	w.Header().Set("Cache-Control", "max-age=3600")

	// convert url to server local file path
	filepath := filepath.Join(s.segmentDir, cleanPath)
	// serve file
	http.ServeFile(w, r, filepath)
}

func (s *HLSServer) start(port string) error {
	if err := s.setup(); err != nil {
		return err
	}

	videoFile := filepath.Join(s.videoDir, "demo.mp4")

	// check video conversion, skip if already
	masterPlaylist := filepath.Join(s.segmentDir, "master.m3u8")
	if _, err := os.Stat(masterPlaylist); os.IsNotExist(err) {
		if err := s.convertToHLS(videoFile); err != nil {
			return fmt.Errorf("HLS conversion failed: %v", err)
		}
		log.Println("Video ready")
	} else {
		log.Println("Use exists HLS segments")
	}

	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/hls/", s.handleHLS)

	log.Printf("HLS server running at: http://localhost%s", port)
	return http.ListenAndServe(port, nil)
}

func RunHLSExample(skip bool) {
	if skip {
		return
	}

	// check ffmpeg
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		log.Fatal("ffmpeg not installed")
	}

	// first run can take couple of minutes to encode the video
	server := newHLSServer()
	if err := server.start(":8080"); err != nil {
		log.Fatal(err)
	}
}
