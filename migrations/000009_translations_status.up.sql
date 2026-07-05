ALTER TABLE translations
  ADD COLUMN status ENUM('pending','processing','completed','failed') NOT NULL DEFAULT 'pending' AFTER folder_id,
  ADD COLUMN error_message TEXT NULL AFTER status,
  ADD COLUMN updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP AFTER created_at,
  ADD UNIQUE KEY unique_folder_id (folder_id);
