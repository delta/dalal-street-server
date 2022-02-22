CREATE TABLE IF NOT EXISTS IpoStocks (
	id int(11) UNSIGNED NOT NULL AUTO_INCREMENT,
	shortName varchar(255) NOT NULL,
	fullName varchar(255) NOT NULL,
	description text NOT NULL,
	SlotPrice bigint(11) UNSIGNED NOT NULL,
	StockPrice bigint(11) UNSIGNED NOT NULL,
	StockPrice bigint(11) UNSIGNED NOT NULL,
	StocksPerSlot bigint(11) UNSIGNED NOT NULL,
	GivesDividends BOOLEAN NOT NULL DEFAULT FALSE,
	createdAt varchar(255) NOT NULL DEFAULT "0000-00-00T00:00:00+05:30",
	updatedAt varchar(255) NOT NULL DEFAULT "0000-00-00T00:00:00+05:30",
	PRIMARY KEY (id),
) AUTO_INCREMENT=1;
