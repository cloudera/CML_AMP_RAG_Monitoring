ALTER TABLE `metrics` RENAME COLUMN `value_numeric` TO `value_numeric_non_nullable`;
ALTER TABLE `metrics` ADD COLUMN `value_numeric` REAL;
UPDATE `metrics` SET `value_numeric` = `value_numeric_non_nullable`;
ALTER TABLE `metrics` DROP COLUMN `value_numeric_non_nullable`;
