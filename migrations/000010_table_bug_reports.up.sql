CREATE TABLE IF NOT EXISTS `bug_reports` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `folder_id` INT NOT NULL,
  `description` TEXT NOT NULL,
  `status` ENUM('open','fixed') NOT NULL DEFAULT 'open',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_bug_reports_status` (`status`),
  FOREIGN KEY (`folder_id`) REFERENCES `new_folders`(`id`) ON DELETE CASCADE
);
