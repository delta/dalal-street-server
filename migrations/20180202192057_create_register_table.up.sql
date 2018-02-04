CREATE TABLE IF NOT EXISTS Register(
	id int(11) UNSIGNED NOT NULL AUTO_INCREMENT,
	userId int(11) UNSIGNED NOT NULL,
	email varchar(70) NOT NULL,
	userName varchar(100) NOT NULL,
	fullName varchar(100) NOT NULL,
	isPragyan BOOL NOT NULL,
	isVerified BOOL NOT NULL,
	pass varchar(80) NOT NULL,
	FOREIGN KEY (userId) REFERENCES Users(id),
	PRIMARY KEY (id)
	) AUTO_INCREMENT = 1;