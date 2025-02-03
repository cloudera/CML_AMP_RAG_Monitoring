CREATE TABLE IF NOT EXISTS experiments (
    id SERIAL PRIMARY KEY,
    experiment_id VARCHAR(100) NOT NULL,
    remote_experiment_id VARCHAR(100),
    created BOOLEAN NOT NULL DEFAULT False, -- save this so that we can know that a create is requested but still in progress
    updated BOOLEAN NOT NULL DEFAULT False, -- save this so that we can know that an update is requested but still in progress
    deleted BOOLEAN NOT NULL DEFAULT False, -- save this so that we can know that a delete is requested but still in progress
    created_ts TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_ts TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(experiment_id)
);
