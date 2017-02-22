CREATE TABLE IF NOT EXISTS Notifications (
	id int(11) UNSIGNED NOT NULL AUTO_INCREMENT,
	userId int(11) UNSIGNED NOT NULL,
	`text` text,
    `isBroadcast` boolean,
	createdAt varchar(255) NOT NULL DEFAULT "0000-00-00T00:00:00+05:30",
	PRIMARY KEY (id)
) AUTO_INCREMENT=1;
