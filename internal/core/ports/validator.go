package ports

// FileValidator defines the interface for file validation
type FileValidator interface {
	ValidateFile(filePath string) error
	ValidateVideoStream(filePath string) error
	TestDecode(filePath string) error
}
