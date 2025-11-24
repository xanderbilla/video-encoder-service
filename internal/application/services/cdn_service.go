package services

import (
	"encoder-service/pkg/config"
	"encoder-service/pkg/types"
	"fmt"
	"path/filepath"
	"strings"
)

// CDNService handles CDN integration
type CDNService struct {
	config *config.Config
}

// CDNProvider represents a CDN provider
type CDNProvider interface {
	PushContent(localPath string, remotePath string) (string, error)
	InvalidateCache(paths []string) error
	GetURL(path string) string
}

func NewCDNService(cfg *config.Config) *CDNService {
	return &CDNService{
		config: cfg,
	}
}

// GetCDNURLs generates CDN URLs for job outputs
func (s *CDNService) GetCDNURLs(job *types.Job, cdnConfig types.CDNConfig) (map[string]string, error) {
	if !cdnConfig.Enabled {
		return nil, nil
	}

	urls := make(map[string]string)

	// Generate master playlist URL
	if job.Output != nil && job.Output.MasterPlaylist != "" {
		masterPath := s.getCDNPath(job.ID, "master.m3u8")
		urls["master"] = cdnConfig.BaseURL + "/" + masterPath
	}

	// Generate quality-specific URLs
	if job.Output != nil {
		for _, quality := range job.Output.Qualities {
			qualityPath := s.getCDNPath(job.ID, quality.Quality+"/index.m3u8")
			urls[quality.Quality] = cdnConfig.BaseURL + "/" + qualityPath
		}
	}

	return urls, nil
}

// PushToCDN pushes job outputs to CDN
func (s *CDNService) PushToCDN(job *types.Job, cdnConfig types.CDNConfig) error {
	if !cdnConfig.Enabled {
		return nil
	}

	provider, err := s.getProvider(cdnConfig)
	if err != nil {
		return fmt.Errorf("failed to get CDN provider: %w", err)
	}

	// Push master playlist
	if job.Output != nil && job.Output.MasterPlaylist != "" {
		remotePath := s.getCDNPath(job.ID, "master.m3u8")
		if _, err := provider.PushContent(job.Output.MasterPlaylist, remotePath); err != nil {
			return fmt.Errorf("failed to push master playlist: %w", err)
		}
	}

	// Push quality playlists and chunks
	if job.Output != nil {
		for _, quality := range job.Output.Qualities {
			// Push playlist
			playlistRemotePath := s.getCDNPath(job.ID, quality.Quality+"/index.m3u8")
			if _, err := provider.PushContent(quality.Playlist, playlistRemotePath); err != nil {
				return fmt.Errorf("failed to push %s playlist: %w", quality.Quality, err)
			}

			// Push chunks directory
			chunksRemotePath := s.getCDNPath(job.ID, quality.Quality+"/chunks/")
			if _, err := provider.PushContent(quality.ChunksDir, chunksRemotePath); err != nil {
				return fmt.Errorf("failed to push %s chunks: %w", quality.Quality, err)
			}
		}
	}

	return nil
}

// InvalidateCache invalidates CDN cache for job outputs
func (s *CDNService) InvalidateCache(job *types.Job, cdnConfig types.CDNConfig) error {
	if !cdnConfig.Enabled {
		return nil
	}

	provider, err := s.getProvider(cdnConfig)
	if err != nil {
		return fmt.Errorf("failed to get CDN provider: %w", err)
	}

	paths := []string{
		s.getCDNPath(job.ID, "*"),
	}

	return provider.InvalidateCache(paths)
}

func (s *CDNService) getProvider(cdnConfig types.CDNConfig) (CDNProvider, error) {
	switch strings.ToLower(cdnConfig.Provider) {
	case "cloudflare":
		return NewCloudflareProvider(cdnConfig), nil
	case "cloudfront":
		return NewCloudFrontProvider(cdnConfig), nil
	default:
		return NewGenericCDNProvider(cdnConfig), nil
	}
}

func (s *CDNService) getCDNPath(jobID string, file string) string {
	return filepath.Join("videos", jobID, file)
}

// GenericCDNProvider is a generic CDN provider implementation
type GenericCDNProvider struct {
	config types.CDNConfig
}

func NewGenericCDNProvider(config types.CDNConfig) *GenericCDNProvider {
	return &GenericCDNProvider{
		config: config,
	}
}

func (p *GenericCDNProvider) PushContent(localPath string, remotePath string) (string, error) {
	// Generic implementation - just return URL
	// In production, implement actual file upload
	return p.GetURL(remotePath), nil
}

func (p *GenericCDNProvider) InvalidateCache(paths []string) error {
	// Generic implementation - no-op
	return nil
}

func (p *GenericCDNProvider) GetURL(path string) string {
	return p.config.BaseURL + "/" + path
}

// CloudflareProvider implements Cloudflare CDN
type CloudflareProvider struct {
	config types.CDNConfig
}

func NewCloudflareProvider(config types.CDNConfig) *CloudflareProvider {
	return &CloudflareProvider{
		config: config,
	}
}

func (p *CloudflareProvider) PushContent(localPath string, remotePath string) (string, error) {
	// TODO: Implement Cloudflare R2 or Workers KV upload
	return p.GetURL(remotePath), nil
}

func (p *CloudflareProvider) InvalidateCache(paths []string) error {
	// TODO: Implement Cloudflare cache purge API
	return nil
}

func (p *CloudflareProvider) GetURL(path string) string {
	return p.config.BaseURL + "/" + path
}

// CloudFrontProvider implements AWS CloudFront CDN
type CloudFrontProvider struct {
	config types.CDNConfig
}

func NewCloudFrontProvider(config types.CDNConfig) *CloudFrontProvider {
	return &CloudFrontProvider{
		config: config,
	}
}

func (p *CloudFrontProvider) PushContent(localPath string, remotePath string) (string, error) {
	// TODO: Implement S3 upload + CloudFront distribution
	return p.GetURL(remotePath), nil
}

func (p *CloudFrontProvider) InvalidateCache(paths []string) error {
	// TODO: Implement CloudFront invalidation API
	return nil
}

func (p *CloudFrontProvider) GetURL(path string) string {
	return p.config.BaseURL + "/" + path
}
