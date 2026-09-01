ALTER TABLE translations
  DROP KEY unique_folder_id,
  DROP COLUMN updated_at,
  DROP COLUMN error_message,
  DROP COLUMN status;
