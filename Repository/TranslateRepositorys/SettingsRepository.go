package TranslateRepositorys

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm/clause"

	dto "web_backend/DTO"
	model "web_backend/Model"
	connection "web_backend/Model/Connection"
	"web_backend/Model/TranslatorSetting"
)

const settingsRowID = 1

// DefaultSettings is the only place defaults live: exactly what the worker
// ran with before settings moved into the DB (old translate_config.json +
// manga_translator/config.py defaults + the hardcoded --use-gpu-limited,
// which a 4GB-VRAM GPU needs to avoid CUDA OOM).
var DefaultSettings = TranslatorSetting.TranslatorSetting{
	ID:                  settingsRowID,
	Translator:          "offline",
	TargetLang:          "ENG",
	Detector:            "default",
	OCR:                 "48px",
	Inpainter:           "lama_large",
	DetectionSize:       2048,
	InpaintingSize:      2048,
	InpaintingPrecision: "bf16",
	GPUMode:             "limited",
}

// Allow-lists mirror the enums in manga-image-translator's
// manga_translator/config.py — update both together.
var (
	allowedTranslators = set("offline", "sugoi", "m2m100", "m2m100_big", "nllb", "nllb_big",
		"jparacrawl", "jparacrawl_big", "mbart50", "qwen2", "qwen2_big",
		"chatgpt", "custom_openai", "deepseek", "gemini", "none")
	allowedTargetLangs = set("ENG", "IND", "JPN", "CHS", "CHT", "KOR")
	allowedDetectors   = set("default", "dbconvnext", "ctd", "craft", "paddle")
	allowedOCRs        = set("32px", "48px", "48px_ctc", "mocr")
	allowedInpainters  = set("default", "lama_large", "lama_mpe", "none", "original")
	allowedPrecisions  = set("bf16", "fp16", "fp32")
	allowedGPUModes    = set("limited", "full", "cpu")
	minSize, maxSize   = 512, 4096
)

func set(values ...string) map[string]bool {
	m := make(map[string]bool, len(values))
	for _, v := range values {
		m[v] = true
	}
	return m
}

func validateSettings(req dto.TranslatorSettingsRequest) error {
	checks := []struct {
		field   string
		value   string
		allowed map[string]bool
	}{
		{"translator", req.Translator, allowedTranslators},
		{"target_lang", req.TargetLang, allowedTargetLangs},
		{"detector", req.Detector, allowedDetectors},
		{"ocr", req.OCR, allowedOCRs},
		{"inpainter", req.Inpainter, allowedInpainters},
		{"inpainting_precision", req.InpaintingPrecision, allowedPrecisions},
		{"gpu_mode", req.GPUMode, allowedGPUModes},
	}
	for _, c := range checks {
		if !c.allowed[c.value] {
			return fmt.Errorf("invalid %s: %q", c.field, c.value)
		}
	}
	for field, size := range map[string]int{"detection_size": req.DetectionSize, "inpainting_size": req.InpaintingSize} {
		if size < minSize || size > maxSize {
			return fmt.Errorf("invalid %s: must be between %d and %d", field, minSize, maxSize)
		}
	}
	if req.APIBase != "" && !strings.HasPrefix(req.APIBase, "http://") && !strings.HasPrefix(req.APIBase, "https://") {
		return errors.New("invalid api_base: must start with http:// or https://")
	}
	return nil
}

// loadSettings returns the stored row, or DefaultSettings (isDefault=true)
// when nothing has been saved yet.
func loadSettings() (row TranslatorSetting.TranslatorSetting, isDefault bool, err error) {
	// Find (not First): an empty table is the normal default state, and
	// First would log it as a "record not found" error on every read.
	result := connection.DB.Where("id = ?", settingsRowID).Limit(1).Find(&row)
	if result.Error != nil {
		return row, false, result.Error
	}
	if result.RowsAffected == 0 {
		return DefaultSettings, true, nil
	}
	return row, false, nil
}

// GetSettings backs the settings page: the API key is never sent back,
// only whether one is stored.
func GetSettings() model.BaseResponseModel {
	row, isDefault, err := loadSettings()
	if err != nil {
		return model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil}
	}
	return model.BaseResponseModel{
		CodeResponse:  200,
		HeaderMessage: "Success",
		Message:       "ok",
		Data: dto.TranslatorSettingsResponse{
			Translator:          row.Translator,
			TargetLang:          row.TargetLang,
			Detector:            row.Detector,
			OCR:                 row.OCR,
			Inpainter:           row.Inpainter,
			DetectionSize:       row.DetectionSize,
			InpaintingSize:      row.InpaintingSize,
			InpaintingPrecision: row.InpaintingPrecision,
			GPUMode:             row.GPUMode,
			APIBase:             row.APIBase,
			APIModel:            row.APIModel,
			HasAPIKey:           row.APIKey != "",
			IsDefault:           isDefault,
		},
	}
}

// GetWorkerSettings backs the translate worker daemon, which needs the
// full API key to pass to manga_translator via env.
func GetWorkerSettings() model.BaseResponseModel {
	row, _, err := loadSettings()
	if err != nil {
		return model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil}
	}
	return model.BaseResponseModel{CodeResponse: 200, HeaderMessage: "Success", Message: "ok", Data: row}
}

func SaveSettings(req dto.TranslatorSettingsRequest) model.BaseResponseModel {
	if err := validateSettings(req); err != nil {
		return model.BaseResponseModel{CodeResponse: 400, HeaderMessage: "Bad Request", Message: err.Error(), Data: nil}
	}

	current, _, err := loadSettings()
	if err != nil {
		return model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil}
	}

	apiKey := current.APIKey
	switch {
	case req.ClearAPIKey:
		apiKey = ""
	case req.APIKey != "":
		apiKey = req.APIKey
	}

	row := TranslatorSetting.TranslatorSetting{
		ID:                  settingsRowID,
		Translator:          req.Translator,
		TargetLang:          req.TargetLang,
		Detector:            req.Detector,
		OCR:                 req.OCR,
		Inpainter:           req.Inpainter,
		DetectionSize:       req.DetectionSize,
		InpaintingSize:      req.InpaintingSize,
		InpaintingPrecision: req.InpaintingPrecision,
		GPUMode:             req.GPUMode,
		APIBase:             strings.TrimSpace(req.APIBase),
		APIModel:            strings.TrimSpace(req.APIModel),
		APIKey:              strings.TrimSpace(apiKey),
	}
	if err := connection.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error; err != nil {
		return model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil}
	}

	return GetSettings()
}

// ResetSettings deletes the row, so every read falls back to DefaultSettings.
func ResetSettings() model.BaseResponseModel {
	if err := connection.DB.Where("id = ?", settingsRowID).Delete(&TranslatorSetting.TranslatorSetting{}).Error; err != nil {
		return model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil}
	}
	return GetSettings()
}
