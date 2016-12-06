CREATE TABLE IF NOT EXISTS Sessions (
	sessionId int(11) NOT NULL AUTO_INCREMENT,
	`key` varchar(255) NOT NULL,
	value varchar(255) NOT NULL,
	PRIMARY KEY (sessionId)
) AUTO_INCREMENT=1;