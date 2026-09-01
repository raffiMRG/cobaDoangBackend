CREATE TABLE IF NOT EXISTS `duplicate_candidates` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(255) NOT NULL,
  `existing_new_folder_id` INT NOT NULL,
  `incoming_path` VARCHAR(500) NOT NULL,
  `existing_page_count` INT NOT NULL DEFAULT 0,
  `incoming_page_count` INT NOT NULL DEFAULT 0,
  `status` VARCHAR(20) NOT NULL DEFAULT 'pending',
  `resolution_note` VARCHAR(255) NULL,
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `resolved_at` TIMESTAMP NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_pending_name` (`name`, `status`),
  FOREIGN KEY (`existing_new_folder_id`) REFERENCES `new_folders`(`id`) ON DELETE CASCADE
);
