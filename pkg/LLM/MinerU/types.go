package mineru

// CreateTaskRequest 对应 POST /api/v4/extract/task 的请求体
type CreateTaskRequest struct {
	URL           string   `json:"url"`
	ModelVersion  string   `json:"model_version,omitempty"`
	IsOCR         bool     `json:"is_ocr,omitempty"`
	EnableFormula bool     `json:"enable_formula,omitempty"`
	EnableTable   bool     `json:"enable_table,omitempty"`
	Language      string   `json:"language,omitempty"`
	DataID        string   `json:"data_id,omitempty"`
	Callback      string   `json:"callback,omitempty"`
	Seed          string   `json:"seed,omitempty"`
	ExtraFormats  []string `json:"extra_formats,omitempty"`
	PageRanges    string   `json:"page_ranges,omitempty"`
}

// CreateTaskResponse 对应创建任务返回
type CreateTaskResponse struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	TraceID string `json:"trace_id"`
	Data    struct {
		TaskID string `json:"task_id"`
	} `json:"data"`
}

// GetTaskResponse 对应 GET /api/v4/extract/task/{task_id}
type GetTaskResponse struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	TraceID string `json:"trace_id"`
	Data    struct {
		TaskID       string `json:"task_id"`
		DataID       string `json:"data_id"`
		State        string `json:"state"`
		FullZipURL   string `json:"full_zip_url"`
		ErrMsg       string `json:"err_msg"`
		ProgressInfo struct {
			ExtractedPages int    `json:"extracted_pages"`
			StartTime      string `json:"start_time"`
			TotalPages     int    `json:"total_pages"`
		} `json:"extract_progress"`
	} `json:"data"`
}
