CREATE TABLE IF NOT EXISTS `experiment_runs` (
    `id` INTEGER PRIMARY KEY,
    `experiment_id` VARCHAR(100) NOT NULL,
    `run_id` VARCHAR(100) NOT NULL,
    `created` BOOLEAN NOT NULL DEFAULT False, -- save this so that we can know that a create is requested but still in progress
    `updated` BOOLEAN NOT NULL DEFAULT False, -- save this so that we can know that an update is requested but still in progress
    `deleted` BOOLEAN NOT NULL DEFAULT False, -- save this so that we can know that a delete is requested but still in progress
    `created_ts` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_ts` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(`experiment_id`, `run_id`)
);
