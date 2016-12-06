CREATE TABLE IF NOT EXISTS Sessions (
	sessionId varchar(255) NOT NULL,
	`key` varchar(255) NOT NULL,
	value varchar(255) NOT NULL,
	PRIMARY KEY (sessionId)
);