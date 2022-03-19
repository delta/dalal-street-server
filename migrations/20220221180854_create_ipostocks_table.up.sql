CREATE TABLE IF NOT EXISTS IpoStocks (
	id int(11) UNSIGNED NOT NULL AUTO_INCREMENT,
	shortName varchar(255) NOT NULL,
	fullName varchar(255) NOT NULL,
	description text NOT NULL,
	slotPrice bigint(11) UNSIGNED NOT NULL,
	stockPrice bigint(11) UNSIGNED NOT NULL,
	slotQuantity bigint(11) UNSIGNED NOT NULL,
	stocksPerSlot bigint(11) UNSIGNED NOT NULL,
	isBiddable BOOLEAN NOT NULL DEFAULT FALSE,
	givesDividends BOOLEAN NOT NULL DEFAULT FALSE,
	createdAt varchar(255) NOT NULL DEFAULT "0000-00-00T00:00:00+05:30",
	updatedAt varchar(255) NOT NULL DEFAULT "0000-00-00T00:00:00+05:30",
	PRIMARY KEY (id)
) AUTO_INCREMENT=1;
