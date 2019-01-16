CREATE TABLE IF NOT EXISTS Stocks (
	id int(11) UNSIGNED NOT NULL AUTO_INCREMENT,
	shortName varchar(255) NOT NULL,
	fullName varchar(255) NOT NULL,
	description text NOT NULL,
	currentPrice bigint(11) UNSIGNED NOT NULL,
	dayHigh bigint(11) UNSIGNED NOT NULL,
	dayLow bigint(11) UNSIGNED NOT NULL,
	allTimeHigh bigint(11) UNSIGNED NOT NULL,
	allTimeLow bigint(11) UNSIGNED NOT NULL,
	stocksInExchange bigint(11) UNSIGNED NOT NULL,
	stocksInMarket bigint(11) UNSIGNED NOT NULL,
	upOrDown tinyint(1) NOT NULL,
    previousDayClose bigint(11) UNSIGNED NOT NULL,
	createdAt varchar(255) NOT NULL DEFAULT "0000-00-00T00:00:00+05:30",
	updatedAt varchar(255) NOT NULL DEFAULT "0000-00-00T00:00:00+05:30",
	PRIMARY KEY (id)
) AUTO_INCREMENT=1;
