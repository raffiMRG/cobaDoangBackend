CREATE TABLE IF NOT EXISTS `translator_settings` (
  `id` INT NOT NULL,
  `translator` VARCHAR(32) NOT NULL,
  `target_lang` VARCHAR(8) NOT NULL,
  `detector` VARCHAR(32) NOT NULL,
  `ocr` VARCHAR(32) NOT NULL,
  `inpainter` VARCHAR(32) NOT NULL,
  `detection_size` INT NOT NULL,
  `inpainting_size` INT NOT NULL,
  `inpainting_precision` VARCHAR(8) NOT NULL,
  `gpu_mode` VARCHAR(16) NOT NULL,
  `api_base` VARCHAR(255) NOT NULL DEFAULT '',
  `api_model` VARCHAR(128) NOT NULL DEFAULT '',
  `api_key` TEXT NULL,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
);
