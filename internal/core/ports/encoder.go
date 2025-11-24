package ports

import "encoder-service/pkg/types"

// VideoEncoder defines the interface for video encoding operations
type VideoEncoder interface {
	GetVideoInfo(inputPath string) (*types.VideoInfo, error)
	SelectVariants(inputWidth, inputHeight int) []types.QualityVariant
	EncodeHLS(jobID, inputPath string, chunkDuration int) (*types.EncodeResult, []types.QualityVariant, error)
}
