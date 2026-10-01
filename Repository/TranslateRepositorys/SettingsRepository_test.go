package TranslateRepositorys

import (
	"testing"

	dto "web_backend/DTO"
)

func validRequest() dto.TranslatorSettingsRequest {
	d := DefaultSettings
	return dto.TranslatorSettingsRequest{
		Translator: d.Translator, TargetLang: d.TargetLang, Detector: d.Detector, OCR: d.OCR,
		Inpainter: d.Inpainter, DetectionSize: d.DetectionSize, InpaintingSize: d.InpaintingSize,
		InpaintingPrecision: d.InpaintingPrecision, GPUMode: d.GPUMode,
	}
}

func TestValidateSettings(t *testing.T) {
	if err := validateSettings(validRequest()); err != nil {
		t.Fatalf("defaults must be valid: %v", err)
	}

	bad := map[string]func(*dto.TranslatorSettingsRequest){
		"translator": func(r *dto.TranslatorSettingsRequest) { r.Translator = "google" },
		"ocr":        func(r *dto.TranslatorSettingsRequest) { r.OCR = "xxx" },
		"gpu_mode":   func(r *dto.TranslatorSettingsRequest) { r.GPUMode = "turbo" },
		"size":       func(r *dto.TranslatorSettingsRequest) { r.InpaintingSize = 100 },
		"api_base":   func(r *dto.TranslatorSettingsRequest) { r.APIBase = "api.anthropic.com" },
	}
	for name, mutate := range bad {
		r := validRequest()
		mutate(&r)
		if validateSettings(r) == nil {
			t.Errorf("%s: expected validation error", name)
		}
	}
}
