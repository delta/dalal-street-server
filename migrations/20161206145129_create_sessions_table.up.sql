CREATE TABLE IF NOT EXISTS Sessions (
	id varchar(255) NOT NULL,
	`key` varchar(255) NOT NULL,
	value varchar(255) NOT NULL,
	PRIMARY KEY (id, `key`)
);
