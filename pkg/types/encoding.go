package types

type QualityVariant struct {
	Name    string
	Width   int
	Height  int
	Bitrate string
	MaxRate string
	BufSize string
}

type VideoInfo struct {
	Width    int
	Height   int
	Duration float64
	HasAudio bool
}

type EncodeResult struct {
	MasterPlaylist string
	QualityOutputs []QualityOutputInfo
}

type QualityOutputInfo struct {
	Quality    string
	Playlist   string
	ChunksDir  string
	ChunkCount int
}

var QualityLadder = []QualityVariant{
	{Name: "240p", Width: 426, Height: 240, Bitrate: "400k", MaxRate: "428k", BufSize: "600k"},
	{Name: "360p", Width: 640, Height: 360, Bitrate: "800k", MaxRate: "856k", BufSize: "1200k"},
	{Name: "480p", Width: 854, Height: 480, Bitrate: "1200k", MaxRate: "1284k", BufSize: "1800k"},
	{Name: "720p", Width: 1280, Height: 720, Bitrate: "2500k", MaxRate: "2675k", BufSize: "3750k"},
	{Name: "1080p", Width: 1920, Height: 1080, Bitrate: "5000k", MaxRate: "5350k", BufSize: "7500k"},
	{Name: "1440p", Width: 2560, Height: 1440, Bitrate: "8000k", MaxRate: "8560k", BufSize: "12000k"},
	{Name: "4K", Width: 3840, Height: 2160, Bitrate: "15000k", MaxRate: "16050k", BufSize: "22500k"},
}
