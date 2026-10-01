package TranslatorSetting

import "time"

// Single-row table (id is always 1) holding the translate worker's
// manga_translator config. No row = DefaultSettings in TranslateRepositorys.
type TranslatorSetting struct {
	ID                  uint      `json:"id" gorm:"primaryKey"`
	Translator          string    `json:"translator"`
	TargetLang          string    `json:"target_lang"`
	Detector            string    `json:"detector"`
	OCR                 string    `json:"ocr" gorm:"column:ocr"`
	Inpainter           string    `json:"inpainter"`
	DetectionSize       int       `json:"detection_size"`
	InpaintingSize      int       `json:"inpainting_size"`
	InpaintingPrecision string    `json:"inpainting_precision"`
	GPUMode             string    `json:"gpu_mode" gorm:"column:gpu_mode"`
	APIBase             string    `json:"api_base" gorm:"column:api_base"`
	APIModel            string    `json:"api_model" gorm:"column:api_model"`
	APIKey              string    `json:"api_key" gorm:"column:api_key"`
	UpdatedAt           time.Time `json:"updated_at"`
}
