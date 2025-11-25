package usecases

import (
	"encoder-service/internal/infrastructure/ffmpeg"
	"encoder-service/pkg/types"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
)

// SeparateAVEncoder handles encoding with separate audio and video tracks
type SeparateAVEncoder struct {
	audioProcessor *ffmpeg.AudioProcessor
	videoEncoder   *ffmpeg.VideoOnlyEncoder
	playlistGen    *ffmpeg.MasterPlaylistGenerator
}

func NewSeparateAVEncoder() *SeparateAVEncoder {
	return &SeparateAVEncoder{
		audioProcessor: ffmpeg.NewAudioProcessor(),
		videoEncoder:   ffmpeg.NewVideoOnlyEncoder(),
		playlistGen:    ffmpeg.NewMasterPlaylistGenerator(),
	}
}

// EncodeSeparateAV encodes video and audio separately
func (e *SeparateAVEncoder) EncodeSeparateAV(jobID, inputPath, outputDir string, videoInfo *types.VideoInfo, chunkDuration int) (*SeparateAVResult, error) {
	result := &SeparateAVResult{}

	// Create directory structure
	videoDir := filepath.Join(outputDir, "video")
	audioDir := filepath.Join(outputDir, "audio", "default")

	// Step 1: Extract and process audio
	log.Printf("Job %s: Extracting and processing audio", jobID)
	rawAudioPath := filepath.Join(outputDir, "raw_audio.aac")
	if err := e.audioProcessor.ExtractAudio(inputPath, rawAudioPath); err != nil {
		return nil, fmt.Errorf("failed to extract audio: %w", err)
	}

	// Process audio with normalization
	processedAudioPath := filepath.Join(outputDir, "processed_audio.aac")
	if err := e.audioProcessor.ProcessAudio(rawAudioPath, processedAudioPath, "192k"); err != nil {
		return nil, fmt.Errorf("failed to process audio: %w", err)
	}

	// Generate audio HLS tracks
	audioQualities, err := e.audioProcessor.GenerateMultiQualityAudio(processedAudioPath, audioDir, chunkDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate audio HLS: %w", err)
	}

	for _, aq := range audioQualities {
		result.AudioTracks = append(result.AudioTracks, types.AudioTrack{
			Language:  "default",
			Bitrate:   aq.Bitrate,
			Playlist:  aq.Playlist,
			IsDefault: aq.Bitrate == "192k",
		})
	}

	// Step 2: Encode video-only tracks
	log.Printf("Job %s: Encoding video-only tracks", jobID)
	qualities := selectQualities(videoInfo.Width, videoInfo.Height)

	for _, q := range qualities {
		qualityDir := filepath.Join(videoDir, q.Name)
		if err := ensureDirExists(qualityDir); err != nil {
			return nil, err
		}

		if err := e.videoEncoder.EncodeVideoOnly(inputPath, qualityDir, q.Width, q.Height, q.Bitrate, chunkDuration); err != nil {
			log.Printf("Warning: Failed to encode %s: %v", q.Name, err)
			continue
		}

		result.VideoTracks = append(result.VideoTracks, types.VideoTrack{
			Quality:    q.Name,
			Resolution: fmt.Sprintf("%dx%d", q.Width, q.Height),
			Bandwidth:  q.BandwidthKbps * 1000,
			Playlist:   filepath.Join("video", q.Name, "main.m3u8"),
		})
	}

	// Step 3: Generate master playlist
	log.Printf("Job %s: Generating master playlist", jobID)
	var videoQualities []string
	for _, vt := range result.VideoTracks {
		videoQualities = append(videoQualities, vt.Quality)
	}

	if err := e.playlistGen.GenerateSeparateAudioVideoMaster(outputDir, videoQualities, []string{"default"}); err != nil {
		return nil, fmt.Errorf("failed to generate master playlist: %w", err)
	}

	result.MasterPlaylist = filepath.Join(outputDir, "master.m3u8")

	return result, nil
}

// SeparateAVResult contains the result of separate A/V encoding
type SeparateAVResult struct {
	VideoTracks    []types.VideoTrack
	AudioTracks    []types.AudioTrack
	MasterPlaylist string
}

type qualitySpec struct {
	Name          string
	Width         int
	Height        int
	Bitrate       string
	BandwidthKbps int
}

func selectQualities(width, height int) []qualitySpec {
	allQualities := []qualitySpec{
		{"240p", 426, 240, "400k", 400},
		{"360p", 640, 360, "800k", 800},
		{"480p", 854, 480, "1200k", 1200},
		{"720p", 1280, 720, "2500k", 2500},
		{"1080p", 1920, 1080, "5000k", 5000},
		{"1440p", 2560, 1440, "8000k", 8000},
		{"4k", 3840, 2160, "12000k", 12000},
	}

	var selected []qualitySpec
	for _, q := range allQualities {
		if q.Width <= width && q.Height <= height {
			selected = append(selected, q)
		}
	}

	return selected
}

func ensureDirExists(path string) error {
	return exec.Command("mkdir", "-p", path).Run()
}
