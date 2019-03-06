CREATE TABLE IF NOT EXISTS GeneralLogs (
	id varchar(255) NOT NULL,
	`key` varchar(255) NOT NULL,
	value text,
	PRIMARY KEY (id, `key`)
);
