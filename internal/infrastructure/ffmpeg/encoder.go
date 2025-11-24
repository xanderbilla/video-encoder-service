package ffmpeg

import (
	"encoder-service/pkg/constants"
	"encoder-service/pkg/types"
	"encoder-service/pkg/utils"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type HLSEncoder struct {
	outputsDir string
}

func NewHLSEncoder(outputsDir string) *HLSEncoder {
	return &HLSEncoder{
		outputsDir: outputsDir,
	}
}

func (e *HLSEncoder) GetVideoInfo(inputPath string) (*types.VideoInfo, error) {
	return GetVideoInfo(inputPath)
}

func (e *HLSEncoder) SelectVariants(inputWidth, inputHeight int) []types.QualityVariant {
	var variants []types.QualityVariant

	for _, variant := range types.QualityLadder {
		if variant.Height <= inputHeight {
			variants = append(variants, variant)
		}
	}

	// Always include at least one variant (lowest quality)
	if len(variants) == 0 && len(types.QualityLadder) > 0 {
		variants = append(variants, types.QualityLadder[0])
	}

	return variants
}

func (e *HLSEncoder) EncodeHLS(jobID, inputPath string, chunkDuration int) (*types.EncodeResult, []types.QualityVariant, error) {
	// Get video info
	videoInfo, err := e.GetVideoInfo(inputPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to detect resolution: %w", err)
	}

	fmt.Printf("Detected input resolution: %dx%d\n", videoInfo.Width, videoInfo.Height)
	fmt.Printf("Audio stream present: %v\n", videoInfo.HasAudio)

	// Select appropriate quality variants
	variants := e.SelectVariants(videoInfo.Width, videoInfo.Height)
	fmt.Printf("Selected %d quality variants: ", len(variants))
	for _, v := range variants {
		fmt.Printf("%s ", v.Name)
	}
	fmt.Println()

	// Create output directories
	jobOutputDir := filepath.Join(e.outputsDir, jobID)
	hlsDir := filepath.Join(jobOutputDir, constants.HLSDir)
	if err := utils.EnsureDir(hlsDir); err != nil {
		return nil, nil, fmt.Errorf("failed to create HLS dir: %w", err)
	}

	// Encode each quality separately with error tracking
	return e.EncodeHLSWithPartialSuccess(jobID, inputPath, chunkDuration, variants, videoInfo.HasAudio)
}

// EncodeHLSWithPartialSuccess encodes all qualities and tracks failures
func (e *HLSEncoder) EncodeHLSWithPartialSuccess(jobID, inputPath string, chunkDuration int, variants []types.QualityVariant, hasAudio bool) (*types.EncodeResult, []types.QualityVariant, error) {
	result := &types.EncodeResult{
		QualityOutputs: []types.QualityOutputInfo{},
	}

	var successfulVariants []types.QualityVariant
	var failedQualities []string
	var lastError error

	// Encode each quality independently
	for _, variant := range variants {
		log.Printf("Encoding %s for job %s...", variant.Name, jobID)
		
		qualityResult := EncodeQuality(jobID, inputPath, variant, chunkDuration, e.outputsDir, hasAudio)
		
		if qualityResult.Success {
			result.QualityOutputs = append(result.QualityOutputs, types.QualityOutputInfo{
				Quality:    qualityResult.Quality,
				Playlist:   qualityResult.Playlist,
				ChunksDir:  qualityResult.ChunksDir,
				ChunkCount: qualityResult.ChunkCount,
			})
			successfulVariants = append(successfulVariants, variant)
		} else {
			log.Printf("Failed to encode %s: %v", variant.Name, qualityResult.Error)
			failedQualities = append(failedQualities, variant.Name)
			lastError = qualityResult.Error
		}
	}

	// If no qualities succeeded, return error
	if len(successfulVariants) == 0 {
		return nil, nil, fmt.Errorf("all quality encodings failed: %w", lastError)
	}

	// Generate master playlist for successful qualities
	hlsDir := filepath.Join(e.outputsDir, jobID, constants.HLSDir)
	masterPlaylist := filepath.Join(hlsDir, constants.MasterPlaylist)
	
	if err := GenerateMasterPlaylist(masterPlaylist, successfulVariants, hasAudio); err != nil {
		return nil, nil, fmt.Errorf("failed to generate master playlist: %w", err)
	}

	result.MasterPlaylist = masterPlaylist

	// If some qualities failed, return partial success
	if len(failedQualities) > 0 {
		log.Printf("Job %s completed with partial success. Failed qualities: %v", jobID, failedQualities)
		return result, successfulVariants, &PartialSuccessError{
			FailedQualities: failedQualities,
			SuccessCount:    len(successfulVariants),
			TotalCount:      len(variants),
		}
	}

	return result, successfulVariants, nil
}

// PartialSuccessError indicates some qualities failed but others succeeded
type PartialSuccessError struct {
	FailedQualities []string
	SuccessCount    int
	TotalCount      int
}

func (e *PartialSuccessError) Error() string {
	return fmt.Sprintf("partial success: %d/%d qualities encoded successfully, failed: %v", 
		e.SuccessCount, e.TotalCount, e.FailedQualities)
}

func GenerateMasterPlaylist(masterPath string, variants []types.QualityVariant, hasAudio bool) error {
	file, err := os.Create(masterPath)
	if err != nil {
		return fmt.Errorf("failed to create master playlist: %w", err)
	}
	defer file.Close()

	file.WriteString("#EXTM3U\n")
	file.WriteString("#EXT-X-VERSION:3\n")

	for _, variant := range variants {
		bandwidth := parseBitrate(variant.Bitrate)
		if hasAudio {
			bandwidth += 128000 // Add audio bitrate
		}

		codecStr := "avc1.640028"
		if hasAudio {
			codecStr += ",mp4a.40.2"
		}

		file.WriteString(fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d,CODECS=\"%s\"\n",
			bandwidth, variant.Width, variant.Height, codecStr))
		file.WriteString(fmt.Sprintf("%s/main.m3u8\n\n", variant.Name))
	}

	return nil
}

func parseBitrate(bitrate string) int {
	// Simple parser for bitrate strings like "5000k"
	var value int
	fmt.Sscanf(bitrate, "%dk", &value)
	return value * 1000
}

func buildFFmpegArgs(inputPath, hlsDir string, chunkDuration int, variants []types.QualityVariant, hasAudio bool) []string {
	args := []string{"-i", inputPath}

	// Add video filters and encoding settings for each variant
	for i, variant := range variants {
		args = append(args,
			"-filter:v:"+fmt.Sprintf("%d", i), fmt.Sprintf("scale=%d:%d", variant.Width, variant.Height),
			"-c:v:"+fmt.Sprintf("%d", i), constants.FFmpegVideoCodec,
			"-b:v:"+fmt.Sprintf("%d", i), variant.Bitrate,
			"-maxrate:v:"+fmt.Sprintf("%d", i), variant.MaxRate,
			"-bufsize:v:"+fmt.Sprintf("%d", i), variant.BufSize,
		)
	}

	// Audio settings (same for all variants) - only if audio exists
	if hasAudio {
		args = append(args,
			"-c:a", constants.FFmpegAudioCodec,
			"-ar", constants.FFmpegAudioSampleRate,
			"-b:a", constants.FFmpegAudioBitrate,
		)
	}

	// Map streams (video + audio for each variant)
	for range variants {
		args = append(args, "-map", "0:v")
		if hasAudio {
			args = append(args, "-map", "0:a")
		}
	}

	// HLS settings
	args = append(args,
		"-f", "hls",
		"-hls_time", fmt.Sprintf("%d", chunkDuration),
		"-hls_playlist_type", constants.FFmpegHLSPlaylistType,
		"-master_pl_name", constants.MasterPlaylist,
	)

	// Build var_stream_map
	varStreamMap := buildVarStreamMap(variants, hasAudio)
	args = append(args, "-var_stream_map", varStreamMap)

	// Add segment filename and playlist patterns
	args = append(args,
		"-hls_segment_filename", filepath.Join(hlsDir, "%v", constants.ChunkPattern),
		"-y",
		filepath.Join(hlsDir, "%v", "main.m3u8"),
	)

	return args
}

func buildVarStreamMap(variants []types.QualityVariant, hasAudio bool) string {
	varStreamMap := ""
	for i, variant := range variants {
		if i > 0 {
			varStreamMap += " "
		}
		if hasAudio {
			varStreamMap += fmt.Sprintf("v:%d,a:%d,name:%s", i, i, variant.Name)
		} else {
			varStreamMap += fmt.Sprintf("v:%d,name:%s", i, variant.Name)
		}
	}
	return varStreamMap
}

func collectEncodingResults(hlsDir string, variants []types.QualityVariant) (*types.EncodeResult, error) {
	result := &types.EncodeResult{
		MasterPlaylist: filepath.Join(hlsDir, constants.MasterPlaylist),
		QualityOutputs: []types.QualityOutputInfo{},
	}

	for _, variant := range variants {
		variantDir := filepath.Join(hlsDir, variant.Name)
		chunks, err := filepath.Glob(filepath.Join(variantDir, "chunk_*.ts"))
		if err != nil {
			return nil, fmt.Errorf("failed to list chunks for variant %s: %w", variant.Name, err)
		}

		result.QualityOutputs = append(result.QualityOutputs, types.QualityOutputInfo{
			Quality:    variant.Name,
			Playlist:   filepath.Join(variantDir, "main.m3u8"),
			ChunksDir:  variantDir,
			ChunkCount: len(chunks),
		})
	}

	return result, nil
}
