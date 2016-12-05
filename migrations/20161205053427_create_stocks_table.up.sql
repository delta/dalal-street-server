CREATE TABLE IF NOT EXISTS Stocks (
	id int(11) NOT NULL AUTO_INCREMENT,
	shortName varchar(255) NOT NULL,
	fullName varchar(255) NOT NULL,
	currentPrice int(11) NOT NULL,
	dayHigh int(11) NOT NULL,
	dayLow int(11) NOT NULL,
	allTimeHigh int(11) NOT NULL,
	allTimeLow int(11) NOT NULL,
	stocksInExchange int(11) NOT NULL,
	upOrDown tinyint(1) NOT NULL,
	createdAt varchar(255) NOT NULL DEFAULT "0000-00-00T00:00:00+05:30",
	updatedAt varchar(255) NOT NULL DEFAULT "0000-00-00T00:00:00+05:30",
	PRIMARY KEY (id)
) AUTO_INCREMENT=1;