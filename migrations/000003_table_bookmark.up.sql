-- Tabel Bookmarks
CREATE TABLE IF NOT EXISTS `bookmarks` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `folder_id` INT NOT NULL,
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  FOREIGN KEY (`folder_id`) REFERENCES `new_folders`(`id`) ON DELETE CASCADE
);
