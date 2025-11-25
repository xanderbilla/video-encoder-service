package ffmpeg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MasterPlaylistGenerator creates HLS master playlist
type MasterPlaylistGenerator struct{}

// VideoVariant represents a video quality variant
type VideoVariant struct {
	Resolution string
	Bandwidth  int
	Width      int
	Height     int
	Playlist   string
}

// AudioVariant represents an audio track variant
type AudioVariant struct {
	Language  string
	Name      string
	Bitrate   int
	IsDefault bool
	Playlist  string
}

func NewMasterPlaylistGenerator() *MasterPlaylistGenerator {
	return &MasterPlaylistGenerator{}
}

// GenerateMasterPlaylist creates master.m3u8 with video and audio variants
func (g *MasterPlaylistGenerator) GenerateMasterPlaylist(outputPath string, videoVariants []VideoVariant, audioVariants []AudioVariant) error {
	var content strings.Builder

	content.WriteString("#EXTM3U\n")
	content.WriteString("#EXT-X-VERSION:6\n\n")

	// Audio renditions
	if len(audioVariants) > 0 {
		content.WriteString("# Audio Renditions\n")
		for _, audio := range audioVariants {
			autoSelect := "NO"
			defaultFlag := "NO"
			if audio.IsDefault {
				autoSelect = "YES"
				defaultFlag = "YES"
			}

			content.WriteString(fmt.Sprintf(
				"#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID=\"audio\",NAME=\"%s\",LANGUAGE=\"%s\",AUTOSELECT=%s,DEFAULT=%s,URI=\"%s\"\n",
				audio.Name,
				audio.Language,
				autoSelect,
				defaultFlag,
				audio.Playlist,
			))
		}
		content.WriteString("\n")
	}

	// Video renditions
	content.WriteString("# Video Renditions\n")
	for _, video := range videoVariants {
		audioGroup := ""
		if len(audioVariants) > 0 {
			audioGroup = ",AUDIO=\"audio\""
		}

		content.WriteString(fmt.Sprintf(
			"#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d%s\n%s\n",
			video.Bandwidth,
			video.Width,
			video.Height,
			audioGroup,
			video.Playlist,
		))
	}

	// Write to file
	if err := os.WriteFile(outputPath, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to write master playlist: %w", err)
	}

	return nil
}

// GenerateSeparateAudioVideoMaster generates master playlist for separate audio/video
func (g *MasterPlaylistGenerator) GenerateSeparateAudioVideoMaster(baseDir string, videoQualities []string, audioLanguages []string) error {
	masterPath := filepath.Join(baseDir, "master.m3u8")

	// Build video variants
	var videoVariants []VideoVariant
	qualityMap := map[string]struct{ width, height, bandwidth int }{
		"240p":  {426, 240, 400000},
		"360p":  {640, 360, 800000},
		"480p":  {854, 480, 1200000},
		"720p":  {1280, 720, 2500000},
		"1080p": {1920, 1080, 5000000},
		"1440p": {2560, 1440, 8000000},
		"4k":    {3840, 2160, 12000000},
	}

	for _, quality := range videoQualities {
		if info, ok := qualityMap[quality]; ok {
			videoVariants = append(videoVariants, VideoVariant{
				Resolution: quality,
				Bandwidth:  info.bandwidth,
				Width:      info.width,
				Height:     info.height,
				Playlist:   fmt.Sprintf("video/%s/main.m3u8", quality),
			})
		}
	}

	// Build audio variants
	var audioVariants []AudioVariant
	for i, lang := range audioLanguages {
		audioVariants = append(audioVariants, AudioVariant{
			Language:  lang,
			Name:      strings.Title(lang),
			Bitrate:   192000,
			IsDefault: i == 0,
			Playlist:  fmt.Sprintf("audio/%s/192k/main.m3u8", lang),
		})
	}

	return g.GenerateMasterPlaylist(masterPath, videoVariants, audioVariants)
}
