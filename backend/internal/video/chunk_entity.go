package video

const ChunkSize = 5 << 20 // 5 MB

type ChunkUploadSession struct {
	UploadID        string `json:"upload_id"`
	StorageUploadID string `json:"storage_upload_id"`
	ObjectKey       string `json:"object_key"`
	AccountID       uint   `json:"account_id"`
	Filename        string `json:"filename"`
	FileSize        int64  `json:"file_size"`
	ChunkSize       int64  `json:"chunk_size"`
	TotalChunks     int    `json:"total_chunks"`
	FileHash        string `json:"file_hash"`
}

type InitChunkUploadRequest struct {
	Filename    string `json:"filename" binding:"required"`
	FileSize    int64  `json:"file_size" binding:"required,min=1"`
	ChunkSize   int64  `json:"chunk_size" binding:"required,min=1"`
	TotalChunks int    `json:"total_chunks" binding:"required,min=1"`
	FileHash    string `json:"file_hash" binding:"required"`
}

type UploadChunkRequest struct {
	UploadID   string `form:"upload_id" binding:"required"`
	ChunkIndex int    `form:"chunk_index" binding:"min=0"`
	ChunkHash  string `form:"chunk_hash" binding:"required"`
}

type ChunkStatusRequest struct {
	UploadID string `json:"upload_id" binding:"required"`
}

type CompleteChunkUploadRequest struct {
	UploadID string `json:"upload_id" binding:"required"`
}

type AbortChunkUploadRequest struct {
	UploadID string `json:"upload_id" binding:"required"`
}
