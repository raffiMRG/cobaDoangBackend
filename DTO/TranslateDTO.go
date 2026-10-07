package dto

type PendingTranslationItem struct {
	FolderID  int    `gorm:"column:folder_id" json:"folder_id"`
	Name      string `gorm:"column:name" json:"name"`
	Thumbnail string `gorm:"column:thumbnail" json:"thumbnail"`
	Status    string `gorm:"column:status" json:"status"`
}

type UpdateTranslationStatusRequest struct {
	Status       string `json:"status" binding:"required"`
	ErrorMessage string `json:"error_message"`
}

type TranslatorSettingsRequest struct {
	Translator          string `json:"translator" binding:"required"`
	TargetLang          string `json:"target_lang" binding:"required"`
	Detector            string `json:"detector" binding:"required"`
	OCR                 string `json:"ocr" binding:"required"`
	Inpainter           string `json:"inpainter" binding:"required"`
	DetectionSize       int    `json:"detection_size" binding:"required"`
	InpaintingSize      int    `json:"inpainting_size" binding:"required"`
	InpaintingPrecision string `json:"inpainting_precision" binding:"required"`
	GPUMode             string `json:"gpu_mode" binding:"required"`
	APIBase             string `json:"api_base"`
	APIModel            string `json:"api_model"`
	APIKey              string `json:"api_key"`       // empty = keep the stored key
	ClearAPIKey         bool   `json:"clear_api_key"` // true = remove the stored key
}

// TranslatorSettingsResponse is what the settings page sees — never the key itself.
type TranslatorSettingsResponse struct {
	Translator          string `json:"translator"`
	TargetLang          string `json:"target_lang"`
	Detector            string `json:"detector"`
	OCR                 string `json:"ocr"`
	Inpainter           string `json:"inpainter"`
	DetectionSize       int    `json:"detection_size"`
	InpaintingSize      int    `json:"inpainting_size"`
	InpaintingPrecision string `json:"inpainting_precision"`
	GPUMode             string `json:"gpu_mode"`
	APIBase             string `json:"api_base"`
	APIModel            string `json:"api_model"`
	HasAPIKey           bool   `json:"has_api_key"`
	IsDefault           bool   `json:"is_default"`
}
