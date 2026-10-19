package model

type HttpDirectUploadInfo struct {
	UploadURL      string            `json:"upload_url"`                 // The URL to upload the file
	FallbackURL    string            `json:"fallback_url,omitempty"`    // Alternative/CDN/ESA upload URL if direct upload is slow
	FallbackMethod string            `json:"fallback_method,omitempty"` // Fallback HTTP method
	ChunkSize      int64             `json:"chunk_size"`                 // The chunk size for uploading, 0 means no chunking required
	Headers        map[string]string `json:"headers,omitempty"`          // Optional headers to include in the upload request
	Method         string            `json:"method,omitempty"`           // HTTP method, default is PUT
}
