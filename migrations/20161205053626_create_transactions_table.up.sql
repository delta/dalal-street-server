CREATE TABLE IF NOT EXISTS Transactions (
	id int(11) UNSIGNED NOT NULL AUTO_INCREMENT,
	userId int(11) UNSIGNED NOT NULL,
	stockId int(11) UNSIGNED NOT NULL,
	type enum('FromExchangeTransaction', 'OrderFillTransaction', 'MortgageTransaction', 'DividendTransaction'),
	stockQuantity bigint(11) SIGNED NOT NULL,
	price bigint(11) UNSIGNED NOT NULL,
	total bigint(11) SIGNED NOT NULL,
	createdAt varchar(255) NOT NULL DEFAULT "0000-00-00T00:00:00+05:30",
	PRIMARY KEY (id),
	FOREIGN KEY (userId) REFERENCES Users(id),
	FOREIGN KEY (stockId) REFERENCES Stocks(id)
) AUTO_INCREMENT=1;
