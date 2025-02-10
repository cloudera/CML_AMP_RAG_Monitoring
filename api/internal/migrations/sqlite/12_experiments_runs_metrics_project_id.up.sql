ALTER TABLE `experiments` ADD COLUMN `project_id` VARCHAR(100);
ALTER TABLE `experiment_runs` ADD COLUMN `project_id` VARCHAR(100);
ALTER TABLE `metrics` ADD COLUMN `project_id` VARCHAR(100);