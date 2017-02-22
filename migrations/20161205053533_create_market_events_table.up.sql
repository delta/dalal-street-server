CREATE TABLE IF NOT EXISTS MarketEvents (
	id int(11) UNSIGNED NOT NULL AUTO_INCREMENT,
	stockId int(11) UNSIGNED NOT NULL,
	emotionScore int(11) SIGNED NOT NULL,
	`text` text,
    headline varchar(255),
    isGlobal boolean,
	createdAt varchar(255) NOT NULL DEFAULT "0000-00-00T00:00:00+05:30",
	PRIMARY KEY (id)
) AUTO_INCREMENT=1;
