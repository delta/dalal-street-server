CREATE TABLE IF NOT EXISTS Users (
	id int(11) NOT NULL AUTO_INCREMENT,
	pragyanId int(11) NOT NULL,
	name varchar(255) NOT NULL,
	cash int(11) NOT NULL,
	total int(11) NOT NULL,
	createdAt varchar(255) NOT NULL DEFAULT "0000-00-00T00:00:00+05:30",
	PRIMARY KEY (id),
	UNIQUE (pragyanId)
) AUTO_INCREMENT=1;