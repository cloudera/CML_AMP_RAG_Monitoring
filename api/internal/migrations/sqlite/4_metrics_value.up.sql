ALTER TABLE `metrics` RENAME COLUMN `value` TO `value_numeric`;
ALTER TABLE `metrics` ADD COLUMN `value_text` TEXT;
